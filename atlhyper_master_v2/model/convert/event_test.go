package convert

import (
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	model_v3 "AtlHyper/model_v3"
	"AtlHyper/model_v3/cluster"
)

func TestEventLogFromModel_FieldMapping(t *testing.T) {
	src := cluster.Event{
		CommonMeta: model_v3.CommonMeta{
			UID:       "uid-1",
			Name:      "event-1",
			Namespace: "default",
			Kind:      "Event",
			CreatedAt: time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
		},
		Type:    "Warning",
		Reason:  "FailedScheduling",
		Message: "0/6 nodes are available",
		Source:  "scheduler",
		InvolvedObject: model_v3.ResourceRef{
			Kind:      "Pod",
			Namespace: "default",
			Name:      "test-pod",
		},
		Count:          3,
		FirstTimestamp: time.Date(2025, 1, 15, 9, 0, 0, 0, time.UTC),
		LastTimestamp:  time.Date(2025, 1, 15, 10, 0, 0, 0, time.UTC),
	}

	result := EventLog(&src, "cluster-1")

	if result.ClusterID != "cluster-1" {
		t.Errorf("ClusterID: got %q, want %q", result.ClusterID, "cluster-1")
	}
	if result.Category != "FailedScheduling" {
		t.Errorf("Category: got %q, want %q", result.Category, "FailedScheduling")
	}
	if result.Kind != "Pod" {
		t.Errorf("Kind: got %q, want %q (should use involved_object.kind)", result.Kind, "Pod")
	}
	if result.Name != "test-pod" {
		t.Errorf("Name: got %q, want %q (should use involved_object.name)", result.Name, "test-pod")
	}
	if result.Namespace != "default" {
		t.Errorf("Namespace: got %q, want %q", result.Namespace, "default")
	}
	if result.Node != "scheduler" {
		t.Errorf("Node: got %q, want %q (should use source)", result.Node, "scheduler")
	}
	if result.Severity != "warning" {
		t.Errorf("Severity: got %q, want %q", result.Severity, "warning")
	}
	if result.Message != "0/6 nodes are available" {
		t.Errorf("Message: got %q, want %q", result.Message, "0/6 nodes are available")
	}
	if result.Reason != "FailedScheduling" {
		t.Errorf("Reason: got %q, want %q", result.Reason, "FailedScheduling")
	}
	// EventTime = last_timestamp, Time = first_timestamp
	if result.EventTime != "2025-01-15T10:00:00Z" {
		t.Errorf("EventTime: got %q, want %q", result.EventTime, "2025-01-15T10:00:00Z")
	}
	if result.Time != "2025-01-15T09:00:00Z" {
		t.Errorf("Time: got %q, want %q", result.Time, "2025-01-15T09:00:00Z")
	}
}

func TestEventLogFromModel_SeverityMapping(t *testing.T) {
	tests := []struct {
		eventType string
		want      string
	}{
		{"Warning", "warning"},
		{"Normal", "info"},
		{"Error", "error"},
		{"", "info"},
	}

	for _, tt := range tests {
		t.Run(tt.eventType, func(t *testing.T) {
			src := cluster.Event{Type: tt.eventType}
			result := EventLog(&src, "c1")
			if result.Severity != tt.want {
				t.Errorf("Type=%q: got severity %q, want %q", tt.eventType, result.Severity, tt.want)
			}
		})
	}
}

func TestEventLogFromModel_FallbackNames(t *testing.T) {
	// When InvolvedObject is empty, fall back to event's own fields
	src := cluster.Event{
		CommonMeta: model_v3.CommonMeta{
			Name: "event-name",
		},
		Type: "Normal",
	}

	result := EventLog(&src, "c1")
	if result.Kind != "Event" {
		t.Errorf("Kind fallback: got %q, want %q", result.Kind, "Event")
	}
	if result.Name != "event-name" {
		t.Errorf("Name fallback: got %q, want %q", result.Name, "event-name")
	}
}

func TestEventLogFromDB_FieldMapping(t *testing.T) {
	src := &database.ClusterEvent{
		DedupKey:          "dedup-1",
		ClusterID:         "cluster-2",
		Namespace:         "kube-system",
		Name:              "coredns-abc",
		Type:              "Warning",
		Reason:            "Unhealthy",
		Message:           "Liveness probe failed",
		SourceComponent:   "kubelet",
		InvolvedKind:      "Pod",
		InvolvedName:      "coredns-pod",
		InvolvedNamespace: "kube-system",
		FirstTimestamp:    time.Date(2025, 2, 1, 8, 0, 0, 0, time.UTC),
		LastTimestamp:     time.Date(2025, 2, 1, 9, 0, 0, 0, time.UTC),
		Count:             5,
		CreatedAt:         time.Date(2025, 2, 1, 8, 0, 0, 0, time.UTC),
	}

	result := EventLogFromDB(src, "cluster-2")

	if result.ClusterID != "cluster-2" {
		t.Errorf("ClusterID: got %q, want %q", result.ClusterID, "cluster-2")
	}
	if result.Kind != "Pod" {
		t.Errorf("Kind: got %q, want %q", result.Kind, "Pod")
	}
	if result.Name != "coredns-pod" {
		t.Errorf("Name: got %q, want %q", result.Name, "coredns-pod")
	}
	if result.Namespace != "kube-system" {
		t.Errorf("Namespace: got %q, want %q", result.Namespace, "kube-system")
	}
	if result.Severity != "warning" {
		t.Errorf("Severity: got %q, want %q", result.Severity, "warning")
	}
	if result.Node != "kubelet" {
		t.Errorf("Node: got %q, want %q", result.Node, "kubelet")
	}
	if result.EventTime != "2025-02-01T09:00:00Z" {
		t.Errorf("EventTime: got %q, want %q", result.EventTime, "2025-02-01T09:00:00Z")
	}
}

func TestEventOverview_Aggregation(t *testing.T) {
	events := []cluster.Event{
		{
			CommonMeta:     model_v3.CommonMeta{Name: "e1"},
			Type:           "Warning",
			Reason:         "FailedScheduling",
			InvolvedObject: model_v3.ResourceRef{Kind: "Pod", Name: "pod-1"},
		},
		{
			CommonMeta:     model_v3.CommonMeta{Name: "e2"},
			Type:           "Warning",
			Reason:         "Unhealthy",
			InvolvedObject: model_v3.ResourceRef{Kind: "Pod", Name: "pod-2"},
		},
		{
			CommonMeta:     model_v3.CommonMeta{Name: "e3"},
			Type:           "Normal",
			Reason:         "Scheduled",
			InvolvedObject: model_v3.ResourceRef{Kind: "Node", Name: "node-1"},
		},
		{
			CommonMeta:     model_v3.CommonMeta{Name: "e4"},
			Type:           "Normal",
			Reason:         "Pulled",
			InvolvedObject: model_v3.ResourceRef{Kind: "Pod", Name: "pod-3"},
		},
	}

	result := EventOverview(events, "cluster-1")

	// Cards
	if result.Cards.TotalEvents != 4 {
		t.Errorf("TotalEvents: got %d, want 4", result.Cards.TotalEvents)
	}
	if result.Cards.Warning != 2 {
		t.Errorf("Warning: got %d, want 2", result.Cards.Warning)
	}
	if result.Cards.Info != 2 {
		t.Errorf("Info: got %d, want 2", result.Cards.Info)
	}
	if result.Cards.Error != 0 {
		t.Errorf("Error: got %d, want 0", result.Cards.Error)
	}
	if result.Cards.TotalAlerts != 2 {
		t.Errorf("TotalAlerts: got %d, want 2 (warning + error)", result.Cards.TotalAlerts)
	}
	// Kinds: Pod, Node = 2
	if result.Cards.KindsCount != 2 {
		t.Errorf("KindsCount: got %d, want 2", result.Cards.KindsCount)
	}
	// Categories: FailedScheduling, Unhealthy, Scheduled, Pulled = 4
	if result.Cards.CategoriesCount != 4 {
		t.Errorf("CategoriesCount: got %d, want 4", result.Cards.CategoriesCount)
	}

	// Rows
	if len(result.Rows) != 4 {
		t.Fatalf("Rows length: got %d, want 4", len(result.Rows))
	}
}

func TestEventOverview_EmptyInput(t *testing.T) {
	result := EventOverview(nil, "c1")
	if result.Cards.TotalEvents != 0 {
		t.Errorf("TotalEvents: got %d, want 0", result.Cards.TotalEvents)
	}
	if result.Rows == nil {
		t.Error("Rows should be non-nil empty slice")
	}
	if len(result.Rows) != 0 {
		t.Errorf("Rows length: got %d, want 0", len(result.Rows))
	}
}

func TestEventLogs_Plural(t *testing.T) {
	events := []cluster.Event{
		{CommonMeta: model_v3.CommonMeta{Name: "e1"}, Type: "Normal"},
		{CommonMeta: model_v3.CommonMeta{Name: "e2"}, Type: "Warning"},
	}

	result := EventLogs(events, "c1")
	if len(result) != 2 {
		t.Fatalf("length: got %d, want 2", len(result))
	}
	if result[0].Severity != "info" {
		t.Errorf("[0].Severity: got %q, want %q", result[0].Severity, "info")
	}
	if result[1].Severity != "warning" {
		t.Errorf("[1].Severity: got %q, want %q", result[1].Severity, "warning")
	}
}

func TestEventLogs_NilInput(t *testing.T) {
	result := EventLogs(nil, "c1")
	if result == nil {
		t.Error("should return non-nil empty slice")
	}
}

func TestEventLogsFromDB_Plural(t *testing.T) {
	events := []*database.ClusterEvent{
		{Name: "e1", Type: "Warning", Reason: "OOM"},
		{Name: "e2", Type: "Normal", Reason: "Pulled"},
	}
	result := EventLogsFromDB(events, "c1")
	if len(result) != 2 {
		t.Fatalf("length: got %d, want 2", len(result))
	}
}
