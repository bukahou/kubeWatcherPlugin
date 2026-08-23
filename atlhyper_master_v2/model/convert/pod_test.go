package convert

import (
	"testing"
	"time"

	"AtlHyper/model_v3/cluster"
)

func TestPodItem_FieldMapping(t *testing.T) {
	src := cluster.Pod{
		Summary: cluster.PodSummary{
			Name:      "web-abc-123",
			Namespace: "default",
			NodeName:  "node-1",
			OwnerKind: "ReplicaSet",
			OwnerName: "web-abc",
			CreatedAt: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
			Age:       "2d",
		},
		Status: cluster.PodStatus{
			Phase:       "Running",
			Ready:       "1/1",
			Restarts:    3,
			CPUUsage:    "100m",
			MemoryUsage: "256Mi",
		},
		Labels: map[string]string{"app": "web"},
	}

	result := PodItem(&src)

	if result.Name != "web-abc-123" {
		t.Errorf("Name: got %q, want %q", result.Name, "web-abc-123")
	}
	if result.Namespace != "default" {
		t.Errorf("Namespace: got %q, want %q", result.Namespace, "default")
	}
	if result.Deployment != "web" {
		t.Errorf("Deployment: got %q, want %q (inferred from ReplicaSet)", result.Deployment, "web")
	}
	if result.Phase != "Running" {
		t.Errorf("Phase: got %q, want %q", result.Phase, "Running")
	}
	if result.Ready != "1/1" {
		t.Errorf("Ready: got %q, want %q", result.Ready, "1/1")
	}
	if result.Restarts != 3 {
		t.Errorf("Restarts: got %d, want %d", result.Restarts, 3)
	}
	if result.CPUText != "100m" {
		t.Errorf("CPUText: got %q, want %q", result.CPUText, "100m")
	}
	if result.MemoryText != "256Mi" {
		t.Errorf("MemoryText: got %q, want %q", result.MemoryText, "256Mi")
	}
	if result.Node != "node-1" {
		t.Errorf("Node: got %q, want %q", result.Node, "node-1")
	}
}

func TestPodItem_EmptyMetrics(t *testing.T) {
	src := cluster.Pod{
		Status: cluster.PodStatus{Phase: "Pending"},
	}
	result := PodItem(&src)
	if result.CPUText != "-" {
		t.Errorf("CPUText: got %q, want %q", result.CPUText, "-")
	}
	if result.MemoryText != "-" {
		t.Errorf("MemoryText: got %q, want %q", result.MemoryText, "-")
	}
}

func TestPodItem_DeploymentInference(t *testing.T) {
	tests := []struct {
		name      string
		ownerKind string
		ownerName string
		labels    map[string]string
		want      string
	}{
		{"ReplicaSet with hash", "ReplicaSet", "nginx-deploy-7b4f8c", nil, "nginx-deploy"},
		{"ReplicaSet single part", "ReplicaSet", "solo", nil, "solo"},
		{"Direct Deployment owner", "Deployment", "my-app", nil, "my-app"},
		{"Fallback to app label", "", "", map[string]string{"app": "label-app"}, "label-app"},
		{"Fallback to k8s label", "", "", map[string]string{"app.kubernetes.io/name": "k8s-app"}, "k8s-app"},
		{"No owner no labels", "", "", nil, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			src := cluster.Pod{
				Summary: cluster.PodSummary{OwnerKind: tt.ownerKind, OwnerName: tt.ownerName},
				Labels:  tt.labels,
			}
			result := PodItem(&src)
			if result.Deployment != tt.want {
				t.Errorf("got %q, want %q", result.Deployment, tt.want)
			}
		})
	}
}

func TestPodDetail_FieldMapping(t *testing.T) {
	src := cluster.Pod{
		Summary: cluster.PodSummary{
			Name:      "web-pod",
			Namespace: "production",
			NodeName:  "node-2",
			OwnerKind: "Deployment",
			OwnerName: "web",
			CreatedAt: time.Date(2025, 2, 1, 8, 0, 0, 0, time.UTC),
			Age:       "10d",
		},
		Spec: cluster.PodSpec{
			RestartPolicy:      "Always",
			ServiceAccountName: "default",
			DNSPolicy:          "ClusterFirst",
		},
		Status: cluster.PodStatus{
			Phase:       "Running",
			Ready:       "2/2",
			Restarts:    0,
			QoSClass:    "Burstable",
			PodIP:       "10.0.0.5",
			HostIP:      "192.168.1.1",
			CPUUsage:    "200m",
			MemoryUsage: "512Mi",
		},
		Containers: []cluster.PodContainerDetail{
			{
				Name:                  "app",
				Image:                 "nginx:1.25",
				State:                 "running",
				RestartCount:          0,
				LastTerminationReason: "OOMKilled",
			},
		},
		Volumes: []cluster.VolumeSpec{
			{Name: "config", Type: "ConfigMap", Source: "app-config"},
		},
	}

	result := PodDetail(&src)

	if result.Controller != "Deployment/web" {
		t.Errorf("Controller: got %q, want %q", result.Controller, "Deployment/web")
	}
	if result.PodIP != "10.0.0.5" {
		t.Errorf("PodIP: got %q, want %q", result.PodIP, "10.0.0.5")
	}
	if result.MemUsage != "512Mi" {
		t.Errorf("MemUsage: got %q, want %q (renamed from MemoryUsage)", result.MemUsage, "512Mi")
	}
	if len(result.Containers) != 1 {
		t.Fatalf("Containers length: got %d, want 1", len(result.Containers))
	}
	if result.Containers[0].LastTerminatedReason != "OOMKilled" {
		t.Errorf("Container LastTerminatedReason: got %q, want %q", result.Containers[0].LastTerminatedReason, "OOMKilled")
	}
	if len(result.Volumes) != 1 {
		t.Fatalf("Volumes length: got %d, want 1", len(result.Volumes))
	}
	if result.Volumes[0].SourceBrief != "app-config" {
		t.Errorf("Volume SourceBrief: got %q, want %q", result.Volumes[0].SourceBrief, "app-config")
	}
}

func TestPodItems_NilInput(t *testing.T) {
	result := PodItems(nil)
	if result == nil {
		t.Error("should return non-nil empty slice")
	}
}
