// atlhyper_master_v2/aiops/correlator/builder_test.go
package correlator

import (
	"testing"

	"AtlHyper/model_v3/apm"
	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/log"
)

func TestBuildFromSnapshot_Empty(t *testing.T) {
	snap := &cluster.ClusterSnapshot{}
	graph := BuildFromSnapshot("test-cluster", snap, snap.OTel)

	if graph == nil {
		t.Fatal("graph should not be nil")
	}
	if graph.ClusterID != "test-cluster" {
		t.Fatalf("want clusterID=test-cluster, got %s", graph.ClusterID)
	}
	if len(graph.Nodes) != 0 {
		t.Fatalf("empty snapshot should produce 0 nodes, got %d", len(graph.Nodes))
	}
	if len(graph.Edges) != 0 {
		t.Fatalf("empty snapshot should produce 0 edges, got %d", len(graph.Edges))
	}
}

func TestBuildFromSnapshot_PodToNode(t *testing.T) {
	snap := &cluster.ClusterSnapshot{
		Pods: []cluster.Pod{
			{
				Summary: cluster.PodSummary{
					Name:      "api-server-abc",
					Namespace: "default",
					NodeName:  "worker-1",
				},
				Labels: map[string]string{"app": "api"},
			},
		},
		Nodes: []cluster.Node{
			{
				Summary: cluster.NodeSummary{Name: "worker-1"},
			},
		},
	}

	graph := BuildFromSnapshot("test", snap, snap.OTel)

	// 应有 pod + node 节点
	if len(graph.Nodes) < 2 {
		t.Fatalf("should have at least 2 nodes (pod+node), got %d", len(graph.Nodes))
	}

	// 检查 runs_on 边
	found := false
	for _, edge := range graph.Edges {
		if edge.Type == "runs_on" && edge.To == "_cluster/node/worker-1" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("should have runs_on edge from pod to node")
	}
}

func TestBuildFromSnapshot_ServiceSelectsPod(t *testing.T) {
	snap := &cluster.ClusterSnapshot{
		Services: []cluster.Service{
			{
				Summary:  cluster.ServiceSummary{Name: "api-svc", Namespace: "default"},
				Selector: map[string]string{"app": "api"},
			},
		},
		Pods: []cluster.Pod{
			{
				Summary: cluster.PodSummary{Name: "api-pod-1", Namespace: "default"},
				Labels:  map[string]string{"app": "api", "version": "v1"},
			},
			{
				Summary: cluster.PodSummary{Name: "other-pod", Namespace: "default"},
				Labels:  map[string]string{"app": "other"},
			},
		},
	}

	graph := BuildFromSnapshot("test", snap, snap.OTel)

	// 应有 selects 边从 service 到匹配的 pod
	selectsEdges := 0
	for _, edge := range graph.Edges {
		if edge.Type == "selects" {
			selectsEdges++
			if edge.From != "default/service/api-svc" {
				t.Fatalf("selects edge from should be service, got %s", edge.From)
			}
			if edge.To != "default/pod/api-pod-1" {
				t.Fatalf("selects edge to should be matching pod, got %s", edge.To)
			}
		}
	}
	if selectsEdges != 1 {
		t.Fatalf("should have exactly 1 selects edge, got %d", selectsEdges)
	}
}

func TestBuildFromSnapshot_IngressRoutesToService(t *testing.T) {
	snap := &cluster.ClusterSnapshot{
		Ingresses: []cluster.Ingress{
			{
				Summary: cluster.IngressSummary{Name: "my-ingress", Namespace: "default"},
				Spec: cluster.IngressSpec{
					Rules: []cluster.IngressRule{
						{
							Host: "api.example.com",
							Paths: []cluster.IngressPath{
								{
									Path: "/",
									Backend: &cluster.IngressBackend{
										Service: &cluster.IngressServiceBackend{
											Name:       "api-svc",
											PortNumber: 80,
										},
									},
								},
							},
						},
					},
				},
			},
		},
		Services: []cluster.Service{
			{
				Summary: cluster.ServiceSummary{Name: "api-svc", Namespace: "default"},
			},
		},
	}

	graph := BuildFromSnapshot("test", snap, snap.OTel)

	// 检查 routes_to 边: ingress 节点 key 使用 ingress 资源名而非 host
	found := false
	for _, edge := range graph.Edges {
		if edge.Type == "routes_to" {
			found = true
			if edge.From != "default/ingress/my-ingress" {
				t.Fatalf("routes_to edge from should be ingress resource key, got %s", edge.From)
			}
			if edge.To != "default/service/api-svc" {
				t.Fatalf("routes_to edge to should be service, got %s", edge.To)
			}
		}
	}
	if !found {
		t.Fatal("should have routes_to edge from ingress to service")
	}
}

// ==================== Enhanced: APM 拓扑边 ====================

func TestBuildFromSnapshot_APMTopologyEdges(t *testing.T) {
	otel := &cluster.OTelSnapshot{
		APMTopology: &apm.Topology{
			Nodes: []apm.TopologyNode{
				{Id: "api-gateway", Name: "api-gateway", Namespace: "default"},
				{Id: "user-svc", Name: "user-svc", Namespace: "default"},
			},
			Edges: []apm.TopologyEdge{
				{Source: "api-gateway", Target: "user-svc", CallCount: 1000, AvgMs: 50, ErrorRate: 0.02},
			},
		},
	}
	snap := &cluster.ClusterSnapshot{}
	graph := BuildFromSnapshot("test", snap, otel)

	// 检查 calls 边
	found := false
	for _, edge := range graph.Edges {
		if edge.Type == "calls" {
			found = true
			if edge.From != "default/service/api-gateway" {
				t.Fatalf("APM calls edge from should be src service, got %s", edge.From)
			}
			if edge.To != "default/service/user-svc" {
				t.Fatalf("APM calls edge to should be dst service, got %s", edge.To)
			}
		}
	}
	if !found {
		t.Fatal("should have calls edge from APM topology")
	}
}

func TestBuildFromSnapshot_NilOTel(t *testing.T) {
	snap := &cluster.ClusterSnapshot{
		Pods: []cluster.Pod{
			{Summary: cluster.PodSummary{Name: "pod-1", Namespace: "default", NodeName: "node-1"}},
		},
		Nodes: []cluster.Node{
			{Summary: cluster.NodeSummary{Name: "node-1"}},
		},
	}
	graph := BuildFromSnapshot("test", snap, nil)

	// 不 panic，且 K8s 边仍正常
	if graph == nil {
		t.Fatal("graph should not be nil")
	}
	runsOnFound := false
	for _, edge := range graph.Edges {
		if edge.Type == "runs_on" {
			runsOnFound = true
		}
		if edge.Type == "calls" {
			t.Fatal("nil otel should not produce calls edges")
		}
	}
	if !runsOnFound {
		t.Fatal("should have runs_on edge even with nil otel")
	}
}

func TestBuildFromSnapshot_LogsGlobalNode(t *testing.T) {
	snap := &cluster.ClusterSnapshot{}
	otel := &cluster.OTelSnapshot{
		LogsSummary: &log.Summary{
			SeverityCounts: map[string]int64{"ERROR": 100},
		},
	}
	graph := BuildFromSnapshot("test", snap, otel)

	logsKey := "_cluster/logs/global"
	found := false
	for _, n := range graph.Nodes {
		if n.Key == logsKey && n.Type == "logs" {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("expected logs/global virtual node in dependency graph")
	}
}

func TestBuildFromSnapshot_NoLogsNodeWhenNilSummary(t *testing.T) {
	snap := &cluster.ClusterSnapshot{}
	otel := &cluster.OTelSnapshot{LogsSummary: nil}
	graph := BuildFromSnapshot("test", snap, otel)

	for _, n := range graph.Nodes {
		if n.Type == "logs" {
			t.Fatal("should not have logs node when LogsSummary is nil")
		}
	}
}
