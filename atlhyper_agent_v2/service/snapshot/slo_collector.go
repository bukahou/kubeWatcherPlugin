// Package snapshot SLO 多窗口数据采集
//
// 本文件实现 SLO 多窗口缓存逻辑：
//   - 3 个窗口 (1d/7d/30d) 各有独立的缓存 TTL
//   - 顺序执行查询以避免 ClickHouse 资源竞争
//   - 每个窗口独立 3 分钟超时
package snapshot

import (
	"context"
	"time"

	"AtlHyper/model_v3/slo"
)

// sloWindowConfig 窗口配置
type sloWindowConfig struct {
	key      string
	since    time.Duration
	bucket   time.Duration
	cacheTTL time.Duration
}

// sloWindowConfigs 窗口配置列表
var sloWindowConfigs = []sloWindowConfig{
	{"1d", 24 * time.Hour, time.Hour, 5 * time.Minute},
	{"7d", 7 * 24 * time.Hour, 6 * time.Hour, 30 * time.Minute},
	{"30d", 30 * 24 * time.Hour, 24 * time.Hour, 2 * time.Hour},
}

// collectSLOWindows 采集多窗口 SLO 数据
func (s *snapshotService) collectSLOWindows(ctx context.Context, now time.Time) map[string]*slo.SLOWindowData {
	if s.sloWindowCaches == nil {
		s.sloWindowCaches = make(map[string]*sloWindowCache)
	}

	result := make(map[string]*slo.SLOWindowData, len(sloWindowConfigs))

	// 顺序执行每个窗口查询（不再并发）
	// 原因: Linkerd gauge 查询在大窗口下需要 30-60 秒，
	// 3 个窗口并发会导致 ClickHouse 资源竞争 → 全部超时。
	// 窗口有独立 TTL 缓存，大部分调用直接命中缓存，不会阻塞。
	for _, wc := range sloWindowConfigs {
		// 检查缓存
		if cache, ok := s.sloWindowCaches[wc.key]; ok && now.Sub(cache.fetchedAt) < wc.cacheTTL {
			result[wc.key] = cache.data
			continue
		}

		data := s.fetchSLOWindow(wc)
		if data != nil {
			s.sloWindowCaches[wc.key] = &sloWindowCache{data: data, fetchedAt: now}
			result[wc.key] = data
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// fetchSLOWindow 获取单个窗口的 SLO 数据（独立 3 分钟超时）
func (s *snapshotService) fetchSLOWindow(wc sloWindowConfig) *slo.SLOWindowData {
	windowCtx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	current, err := s.dashboardRepo.ListIngressSLO(windowCtx, wc.since)
	if err != nil {
		log.Warn("SLO 窗口 current 查询失败", "window", wc.key, "err", err)
		return nil
	}

	s.applyRouteHostnames(windowCtx, current)

	previous, _ := s.dashboardRepo.ListIngressSLOPrevious(windowCtx, wc.since)
	history, _ := s.dashboardRepo.GetIngressSLOHistory(windowCtx, wc.since, wc.bucket)

	// 服务网格 SLO 已移除：SLO 只做 ingress 外部视角，
	// 服务间调用质量由 APM 承担（见 slo-ingress-contract-design.md）。
	return &slo.SLOWindowData{
		Current:  current,
		Previous: previous,
		History:  history,
	}
}

// applyRouteHostnames 用路由映射把 IngressSLO 的 DisplayName 从 serviceKey
// 替换为对外域名（如 "geass-v3/geass-gateway" → "geass-api.bukahou.com"）。
//
// 为什么在 service 层做而不是 SQL 层：
//
//	域名来自 K8s 路由资源，指标来自 ClickHouse —— 两个不同数据源的组合是
//	service 层的职责。repository 只应关心自己那一个数据源，让 SLO 的 SQL
//	去查 K8s 会破坏分层。
//
// 查不到映射的服务保留 serviceKey 作为 DisplayName —— 有值总比空白好，
// 且能一眼看出"这个服务没有对外路由"（可能是内部服务或路由配置遗漏）。
func (s *snapshotService) applyRouteHostnames(ctx context.Context, items []slo.IngressSLO) {
	if len(items) == 0 || s.routeRepo == nil {
		return
	}
	hostMap, err := s.routeRepo.GetServiceHostMap(ctx)
	if err != nil || len(hostMap) == 0 {
		return // 映射不可用时静默保留 serviceKey，不影响 SLO 主数据
	}
	for i := range items {
		if host, ok := hostMap[items[i].ServiceKey]; ok && host != "" {
			items[i].DisplayName = host
		}
	}
}
