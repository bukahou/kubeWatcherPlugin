// atlhyper_master_v2/aiops/correlator/builder.go
// 从 ClusterSnapshot 构建依赖图 DAG
package correlator

import (
	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/model_v3/cluster"
)

// BuildFromSnapshot 从快照构建完整依赖图
// otel 参数独立传入（而非从 snap.OTel 读取），便于 AIOps 引擎控制数据源
func BuildFromSnapshot(clusterID string, snap *cluster.ClusterSnapshot, otel *cluster.OTelSnapshot) *aiops.DependencyGraph {
	g := aiops.NewDependencyGraph(clusterID)

	// 1. Pod → Node (runs_on)
	for i := range snap.Pods {
		pod := &snap.Pods[i]
		podKey := aiops.EntityKey(pod.Summary.Namespace, "pod", pod.Summary.Name)
		g.AddNode(podKey, "pod", pod.Summary.Namespace, pod.Summary.Name, nil)

		if pod.Summary.NodeName != "" {
			nodeKey := aiops.EntityKey("_cluster", "node", pod.Summary.NodeName)
			g.AddNode(nodeKey, "node", "_cluster", pod.Summary.NodeName, nil)
			g.AddEdge(podKey, nodeKey, "runs_on", 1.0)
		}
	}

	// 2. Service → Pod (selects)
	for i := range snap.Services {
		svc := &snap.Services[i]
		svcKey := aiops.EntityKey(svc.Summary.Namespace, "service", svc.Summary.Name)
		g.AddNode(svcKey, "service", svc.Summary.Namespace, svc.Summary.Name, nil)

		if len(svc.Selector) == 0 {
			continue
		}
		for j := range snap.Pods {
			pod := &snap.Pods[j]
			if svc.Summary.Namespace != pod.Summary.Namespace {
				continue
			}
			if matchSelector(svc.Selector, pod.Labels) {
				podKey := aiops.EntityKey(pod.Summary.Namespace, "pod", pod.Summary.Name)
				g.AddEdge(svcKey, podKey, "selects", 1.0)
			}
		}
	}

	// 3. Ingress → Service (routes_to)
	for i := range snap.Ingresses {
		ing := &snap.Ingresses[i]
		ingKey := aiops.EntityKey(ing.Summary.Namespace, "ingress", ing.Summary.Name)
		g.AddNode(ingKey, "ingress", ing.Summary.Namespace, ing.Summary.Name, nil)

		for _, rule := range ing.Spec.Rules {
			for _, path := range rule.Paths {
				if path.Backend != nil && path.Backend.Service != nil {
					svcKey := aiops.EntityKey(ing.Summary.Namespace, "service", path.Backend.Service.Name)
					g.AddEdge(ingKey, svcKey, "routes_to", 1.0)
				}
			}
		}
	}

	// 4. Service → Service (calls, 从 SLO Edge + APM Topology，去重)
	if otel != nil {
		edgeSet := make(map[string]bool) // "srcKey->dstKey" 去重

		// 4b. APM 拓扑边
		if otel.APMTopology != nil {
			// 构建 nodeId → namespace 索引
			nsIndex := make(map[string]string, len(otel.APMTopology.Nodes))
			for _, n := range otel.APMTopology.Nodes {
				nsIndex[n.Id] = n.Namespace
			}

			for _, edge := range otel.APMTopology.Edges {
				srcNs := nsIndex[edge.Source]
				dstNs := nsIndex[edge.Target]
				if srcNs == "" {
					srcNs = "_cluster"
				}
				if dstNs == "" {
					dstNs = "_cluster"
				}
				srcKey := aiops.EntityKey(srcNs, "service", edge.Source)
				dstKey := aiops.EntityKey(dstNs, "service", edge.Target)
				dedupKey := srcKey + "->" + dstKey
				if edgeSet[dedupKey] {
					continue
				}
				edgeSet[dedupKey] = true
				g.AddNode(srcKey, "service", srcNs, edge.Source, nil)
				g.AddNode(dstKey, "service", dstNs, edge.Target, nil)
				g.AddEdge(srcKey, dstKey, "calls", 1.0)
			}
		}
	}

	// 确保 Node 节点存在（可能没有 Pod 调度到的独立节点）
	for i := range snap.Nodes {
		node := &snap.Nodes[i]
		nodeKey := aiops.EntityKey("_cluster", "node", node.GetName())
		g.AddNode(nodeKey, "node", "_cluster", node.GetName(), nil)
	}

	// 5. 虚拟实体：logs/global（OTel 日志聚合节点）
	if otel != nil && otel.LogsSummary != nil {
		logsKey := aiops.EntityKey("_cluster", "logs", "global")
		g.AddNode(logsKey, "logs", "_cluster", "Log Summary", nil)
	}

	g.RebuildIndex()
	return g
}

// matchSelector 检查 Pod Labels 是否匹配 Service Selector
func matchSelector(selector, labels map[string]string) bool {
	if len(selector) == 0 {
		return false
	}
	for k, v := range selector {
		if labels[k] != v {
			return false
		}
	}
	return true
}
