// routemap.go 从 K8s 路由资源反查「后端服务 → 对外域名」映射。
//
// 为什么需要：
//
//	SLO 的 displayName 要显示真实域名（geass-api.bukahou.com），但任何 ingress
//	实现的【指标】都不带域名维度 —— 实测 Cilium Envoy 未启用 vhost 统计、
//	Hubble 的 httpV2 无 host label、Traefik 也只有 service 名。
//
// 为什么从路由资源取而不是指标标签：
//
//	① 抽象：HTTPRoute / Ingress 都是 K8s 标准资源，换 ingress 实现照样有；
//	   而指标标签是各实现自己决定的，等于把域名维度也绑死在实现上。
//	② 完整：一个服务挂多域名时路由资源能全部表达，指标标签只能带一个。
//
// 支持两种标准路由资源，按集群实际使用情况择一或并存：
//   - Gateway API HTTPRoute（当前集群使用）
//   - 原生 Ingress（Nginx / Traefik 等传统 ingress 控制器）
package k8s

import (
	"context"
	"fmt"
	"sort"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/rest"

	"AtlHyper/model_v3/cluster"
)

// RouteEntry 是从任意路由资源抽取出的中间形态 —— 域名与后端服务的对应关系。
//
// 这一层的存在意义是隔离资源差异：HTTPRoute 与 Ingress 的字段结构完全不同，
// 但都能归约成「一组域名 → 一组后端」，下游只处理这个统一形态。
type RouteEntry struct {
	// Hostnames 是该路由对外暴露的域名列表。
	Hostnames []string
	// Backends 是后端服务标识，格式 "namespace/service"，与 SLO 的 serviceKey 一致。
	Backends []string
}

// BuildRouteMap 汇总多条路由，产出 serviceKey → 域名字符串 的映射。
//
// 同一后端被多条路由引用时，域名合并并去重、按字典序排列后以 ", " 连接
// （排序是为了输出稳定，否则 map 遍历顺序会让结果每次不同）。
//
// 无域名（通配路由）或无后端的条目直接跳过，不产生空键。
func BuildRouteMap(routes []RouteEntry) map[string]string {
	// 先收集到 set，避免重复域名
	acc := make(map[string]map[string]struct{})

	for _, r := range routes {
		for _, backend := range r.Backends {
			if backend == "" {
				continue
			}
			for _, host := range r.Hostnames {
				if host == "" {
					continue
				}
				if acc[backend] == nil {
					acc[backend] = make(map[string]struct{})
				}
				acc[backend][host] = struct{}{}
			}
		}
	}

	result := make(map[string]string, len(acc))
	for backend, hostSet := range acc {
		hosts := make([]string, 0, len(hostSet))
		for h := range hostSet {
			hosts = append(hosts, h)
		}
		sort.Strings(hosts)
		result[backend] = strings.Join(hosts, ", ")
	}
	return result
}

// ──────────────────────────────────────────────────────────────
// 从集群实际路由资源抽取 RouteEntry
// ──────────────────────────────────────────────────────────────

// httpRouteGVR 是 Gateway API HTTPRoute 的资源坐标。
//
// 用 dynamic client 而非引入 sigs.k8s.io/gateway-api 依赖：只读两个字段
// （hostnames / backendRefs），为此拖进一整个 CRD 类型库不划算，
// 且 dynamic 在集群未装 Gateway API 时能优雅降级（返回 NotFound 而非编译期强依赖）。
var httpRouteGVR = schema.GroupVersionResource{
	Group:    "gateway.networking.k8s.io",
	Version:  "v1",
	Resource: "httproutes",
}

// FetchHTTPRoutes 列出全集群 HTTPRoute 并归约为 RouteEntry。
//
// 集群未安装 Gateway API 时返回空切片而非错误 —— 该情形下 ingress 多半由
// 原生 Ingress 承担，调用方会退回 FetchIngressRoutes。
func FetchHTTPRoutes(ctx context.Context, cfg *rest.Config) ([]RouteEntry, error) {
	if cfg == nil {
		return nil, nil
	}
	dyn, err := dynamic.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("create dynamic client: %w", err)
	}

	list, err := dyn.Resource(httpRouteGVR).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) || meta.IsNoMatchError(err) {
			return nil, nil // 集群未装 Gateway API
		}
		return nil, fmt.Errorf("list httproutes: %w", err)
	}

	entries := make([]RouteEntry, 0, len(list.Items))
	for i := range list.Items {
		entries = append(entries, httpRouteToEntry(&list.Items[i]))
	}
	return entries, nil
}

// httpRouteToEntry 从 HTTPRoute 的 unstructured 形态抽取域名与后端。
//
// backendRefs 的 namespace 可省略（默认与 HTTPRoute 同 namespace），
// 这是 Gateway API 的规范行为，必须补全否则 serviceKey 对不上。
func httpRouteToEntry(u *unstructured.Unstructured) RouteEntry {
	ns := u.GetNamespace()
	var e RouteEntry

	hosts, _, _ := unstructured.NestedStringSlice(u.Object, "spec", "hostnames")
	e.Hostnames = hosts

	rules, _, _ := unstructured.NestedSlice(u.Object, "spec", "rules")
	seen := make(map[string]struct{})
	for _, r := range rules {
		rule, ok := r.(map[string]any)
		if !ok {
			continue
		}
		refs, _, _ := unstructured.NestedSlice(rule, "backendRefs")
		for _, b := range refs {
			ref, ok := b.(map[string]any)
			if !ok {
				continue
			}
			name, _, _ := unstructured.NestedString(ref, "name")
			if name == "" {
				continue
			}
			backendNS, found, _ := unstructured.NestedString(ref, "namespace")
			if !found || backendNS == "" {
				backendNS = ns // 规范默认：同 namespace
			}
			key := backendNS + "/" + name
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			e.Backends = append(e.Backends, key)
		}
	}
	return e
}

// IngressesToRouteEntries 从原生 Ingress 归约 RouteEntry。
//
// 与 HTTPRoute 并列的第二种标准来源：集群用 Nginx / Traefik 等传统 ingress
// 控制器时走这条路。两者产出同一中间形态，下游无需区分。
func IngressesToRouteEntries(ingresses []cluster.Ingress) []RouteEntry {
	entries := make([]RouteEntry, 0, len(ingresses))
	for i := range ingresses {
		ing := &ingresses[i]
		var e RouteEntry
		seen := make(map[string]struct{})
		for _, rule := range ing.Spec.Rules {
			if rule.Host != "" {
				e.Hostnames = append(e.Hostnames, rule.Host)
			}
			for _, path := range rule.Paths {
				if path.Backend == nil || path.Backend.Service == nil || path.Backend.Service.Name == "" {
					continue
				}
				key := ing.Summary.Namespace + "/" + path.Backend.Service.Name
				if _, dup := seen[key]; dup {
					continue
				}
				seen[key] = struct{}{}
				e.Backends = append(e.Backends, key)
			}
		}
		entries = append(entries, e)
	}
	return entries
}
