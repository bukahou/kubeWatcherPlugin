package command

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestSummarizeList_PodList(t *testing.T) {
	input := map[string]interface{}{
		"kind":       "PodList",
		"apiVersion": "v1",
		"items": []interface{}{
			map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":              "nginx-abc123",
					"namespace":         "default",
					"creationTimestamp": time.Now().Add(-3 * 24 * time.Hour).Format(time.RFC3339),
				},
				"status": map[string]interface{}{
					"phase": "Running",
					"containerStatuses": []interface{}{
						map[string]interface{}{
							"name":         "nginx",
							"restartCount": float64(2),
							"state": map[string]interface{}{
								"running": map[string]interface{}{},
							},
						},
					},
				},
			},
			map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":              "redis-xyz789",
					"namespace":         "cache",
					"creationTimestamp": time.Now().Add(-10 * time.Hour).Format(time.RFC3339),
				},
				"status": map[string]interface{}{
					"phase": "Running",
					"containerStatuses": []interface{}{
						map[string]interface{}{
							"name":         "redis",
							"restartCount": float64(0),
							"state": map[string]interface{}{
								"running": map[string]interface{}{},
							},
						},
					},
				},
			},
		},
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal input: %v", err)
	}

	output := summarizeList(data)

	// Check column headers
	expectedHeaders := []string{"NAME", "NAMESPACE", "STATUS", "RESTARTS", "AGE"}
	for _, h := range expectedHeaders {
		if !strings.Contains(output, h) {
			t.Errorf("output missing header %q, got:\n%s", h, output)
		}
	}

	// Check item names present
	if !strings.Contains(output, "nginx-abc123") {
		t.Errorf("output missing pod name 'nginx-abc123', got:\n%s", output)
	}
	if !strings.Contains(output, "redis-xyz789") {
		t.Errorf("output missing pod name 'redis-xyz789', got:\n%s", output)
	}

	// Check count in title
	if !strings.Contains(output, "Pod (2)") {
		t.Errorf("output missing 'Pod (2)' title, got:\n%s", output)
	}
}

func TestSummarizeList_EmptyItems(t *testing.T) {
	input := `{"kind":"PodList","items":[]}`
	output := summarizeList([]byte(input))

	expected := "Pod: 0 items"
	if output != expected {
		t.Errorf("expected %q, got %q", expected, output)
	}
}

func TestSummarizeList_InvalidJSON(t *testing.T) {
	input := []byte(`{invalid json!!!}`)
	output := summarizeList(input)

	if output != string(input) {
		t.Errorf("expected input returned as-is for invalid JSON, got %q", output)
	}
}

func TestSummarizeList_NodeList(t *testing.T) {
	input := map[string]interface{}{
		"kind":       "NodeList",
		"apiVersion": "v1",
		"items": []interface{}{
			map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":              "node-1",
					"creationTimestamp": time.Now().Add(-30 * 24 * time.Hour).Format(time.RFC3339),
					"labels": map[string]interface{}{
						"node-role.kubernetes.io/control-plane": "",
					},
				},
				"status": map[string]interface{}{
					"conditions": []interface{}{
						map[string]interface{}{
							"type":   "Ready",
							"status": "True",
						},
					},
					"nodeInfo": map[string]interface{}{
						"kubeletVersion": "v1.29.0",
					},
				},
			},
			map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":              "worker-1",
					"creationTimestamp": time.Now().Add(-5 * 24 * time.Hour).Format(time.RFC3339),
					"labels":            map[string]interface{}{},
				},
				"status": map[string]interface{}{
					"conditions": []interface{}{
						map[string]interface{}{
							"type":   "Ready",
							"status": "True",
						},
					},
					"nodeInfo": map[string]interface{}{
						"kubeletVersion": "v1.29.0",
					},
				},
			},
		},
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal input: %v", err)
	}

	output := summarizeList(data)

	expectedHeaders := []string{"NAME", "STATUS", "ROLES", "VERSION", "AGE"}
	for _, h := range expectedHeaders {
		if !strings.Contains(output, h) {
			t.Errorf("output missing header %q, got:\n%s", h, output)
		}
	}

	if !strings.Contains(output, "node-1") {
		t.Errorf("output missing node name 'node-1', got:\n%s", output)
	}
	if !strings.Contains(output, "Ready") {
		t.Errorf("output missing status 'Ready', got:\n%s", output)
	}
	if !strings.Contains(output, "control-plane") {
		t.Errorf("output missing role 'control-plane', got:\n%s", output)
	}
	if !strings.Contains(output, "Node (2)") {
		t.Errorf("output missing 'Node (2)' title, got:\n%s", output)
	}
}

func TestFormatAge(t *testing.T) {
	tests := []struct {
		name      string
		timestamp string
		expected  string
	}{
		{
			name:      "10 days ago",
			timestamp: time.Now().Add(-10 * 24 * time.Hour).Format(time.RFC3339),
			expected:  "10d",
		},
		{
			name:      "5 hours ago",
			timestamp: time.Now().Add(-5 * time.Hour).Format(time.RFC3339),
			expected:  "5h",
		},
		{
			name:      "30 minutes ago",
			timestamp: time.Now().Add(-30 * time.Minute).Format(time.RFC3339),
			expected:  "30m",
		},
		{
			name:      "15 seconds ago",
			timestamp: time.Now().Add(-15 * time.Second).Format(time.RFC3339),
			expected:  "15s",
		},
		{
			name:      "empty string",
			timestamp: "",
			expected:  "<unknown>",
		},
		{
			name:      "invalid timestamp",
			timestamp: "not-a-timestamp",
			expected:  "<unknown>",
		},
		{
			name:      "future timestamp",
			timestamp: time.Now().Add(1 * time.Hour).Format(time.RFC3339),
			expected:  "0s",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatAge(tt.timestamp)
			if got != tt.expected {
				t.Errorf("formatAge(%q) = %q, want %q", tt.timestamp, got, tt.expected)
			}
		})
	}
}

func TestFormatTable_EmptyRows(t *testing.T) {
	cols := []column{
		{"NAME", nil},
		{"NAMESPACE", nil},
		{"STATUS", nil},
	}
	result := formatTable("Pod", cols, nil)

	expected := "Pod: 0 items"
	if result != expected {
		t.Errorf("expected %q, got %q", expected, result)
	}

	// Also test with explicit empty slice
	result2 := formatTable("Deployment", cols, [][]string{})
	expected2 := "Deployment: 0 items"
	if result2 != expected2 {
		t.Errorf("expected %q, got %q", expected2, result2)
	}
}

func TestFormatTable_Alignment(t *testing.T) {
	cols := []column{
		{"NAME", nil},
		{"STATUS", nil},
		{"AGE", nil},
	}
	rows := [][]string{
		{"short", "Running", "5d"},
		{"a-very-long-resource-name", "Pending", "12h"},
		{"mid", "Failed", "30m"},
	}

	result := formatTable("Pod", cols, rows)

	// Verify title line
	if !strings.Contains(result, "Pod (3):") {
		t.Errorf("expected title 'Pod (3):', got:\n%s", result)
	}

	lines := strings.Split(result, "\n")
	if len(lines) < 5 { // title + header + 3 data rows + trailing newline
		t.Fatalf("expected at least 5 lines, got %d:\n%s", len(lines), result)
	}

	// Header line (index 1, after title)
	headerLine := lines[1]
	if !strings.Contains(headerLine, "NAME") || !strings.Contains(headerLine, "STATUS") || !strings.Contains(headerLine, "AGE") {
		t.Errorf("header line missing expected columns: %q", headerLine)
	}

	// Verify all data rows have consistent column positions
	// The NAME column should be padded to at least the width of "a-very-long-resource-name" (25 chars)
	dataLine1 := lines[2]
	dataLine2 := lines[3]

	// Find position of STATUS value in both lines - they should start at the same column
	statusIdx1 := strings.Index(dataLine1, "Running")
	statusIdx2 := strings.Index(dataLine2, "Pending")
	if statusIdx1 != statusIdx2 {
		t.Errorf("columns not aligned: 'Running' at index %d, 'Pending' at index %d\nLine1: %q\nLine2: %q",
			statusIdx1, statusIdx2, dataLine1, dataLine2)
	}
}

func TestSummarizeList_DeploymentList(t *testing.T) {
	input := map[string]interface{}{
		"kind":       "DeploymentList",
		"apiVersion": "apps/v1",
		"items": []interface{}{
			map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":              "web-app",
					"namespace":         "production",
					"creationTimestamp": time.Now().Add(-7 * 24 * time.Hour).Format(time.RFC3339),
				},
				"spec": map[string]interface{}{
					"replicas": float64(3),
				},
				"status": map[string]interface{}{
					"readyReplicas":     float64(3),
					"updatedReplicas":   float64(3),
					"availableReplicas": float64(3),
				},
			},
		},
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal input: %v", err)
	}

	output := summarizeList(data)

	expectedHeaders := []string{"NAME", "NAMESPACE", "READY", "UP-TO-DATE", "AVAILABLE", "AGE"}
	for _, h := range expectedHeaders {
		if !strings.Contains(output, h) {
			t.Errorf("output missing header %q, got:\n%s", h, output)
		}
	}

	if !strings.Contains(output, "web-app") {
		t.Errorf("output missing deployment name 'web-app', got:\n%s", output)
	}
	if !strings.Contains(output, "3/3") {
		t.Errorf("output missing ready count '3/3', got:\n%s", output)
	}
}

func TestSummarizeList_NoItemsKey(t *testing.T) {
	// JSON object without "items" key should return as-is
	input := `{"kind":"SomeList","apiVersion":"v1"}`
	output := summarizeList([]byte(input))

	if output != input {
		t.Errorf("expected input returned as-is when no items key, got %q", output)
	}
}

func TestFormatTable_SingleRow(t *testing.T) {
	cols := []column{
		{"NAME", nil},
		{"STATUS", nil},
	}
	rows := [][]string{
		{"nginx", "Running"},
	}

	result := formatTable("Pod", cols, rows)

	if !strings.Contains(result, "Pod (1):") {
		t.Errorf("expected title 'Pod (1):', got:\n%s", result)
	}
	if !strings.Contains(result, "nginx") {
		t.Errorf("expected row data 'nginx', got:\n%s", result)
	}
}

func TestFormatTable_HeaderWiderThanData(t *testing.T) {
	cols := []column{
		{"VERY-LONG-HEADER-NAME", nil},
		{"S", nil},
	}
	rows := [][]string{
		{"a", "b"},
	}

	result := formatTable("Test", cols, rows)

	lines := strings.Split(result, "\n")
	if len(lines) < 3 {
		t.Fatalf("expected at least 3 lines, got %d", len(lines))
	}

	headerLine := lines[1]
	dataLine := lines[2]

	// The "S" header and "b" data should start at the same position
	sIdx := strings.Index(headerLine, "S")
	bIdx := strings.Index(dataLine, "b")
	if sIdx != bIdx {
		t.Errorf("columns not aligned when header wider than data: 'S' at %d, 'b' at %d\nHeader: %q\nData:   %q",
			sIdx, bIdx, headerLine, dataLine)
	}
}

func TestSummarizeList_ServiceList(t *testing.T) {
	input := map[string]interface{}{
		"kind":       "ServiceList",
		"apiVersion": "v1",
		"items": []interface{}{
			map[string]interface{}{
				"metadata": map[string]interface{}{
					"name":              "my-svc",
					"namespace":         "default",
					"creationTimestamp": time.Now().Add(-2 * 24 * time.Hour).Format(time.RFC3339),
				},
				"spec": map[string]interface{}{
					"type":      "ClusterIP",
					"clusterIP": "10.96.0.1",
					"ports": []interface{}{
						map[string]interface{}{
							"port":     float64(80),
							"protocol": "TCP",
						},
					},
				},
			},
		},
	}

	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("failed to marshal input: %v", err)
	}

	output := summarizeList(data)

	for _, h := range []string{"NAME", "NAMESPACE", "TYPE", "CLUSTER-IP", "PORTS", "AGE"} {
		if !strings.Contains(output, h) {
			t.Errorf("output missing header %q, got:\n%s", h, output)
		}
	}

	if !strings.Contains(output, "ClusterIP") {
		t.Errorf("output missing type 'ClusterIP', got:\n%s", output)
	}
	if !strings.Contains(output, "80/TCP") {
		t.Errorf("output missing port '80/TCP', got:\n%s", output)
	}
}

func TestSummarizeList_GenericUnknownKind(t *testing.T) {
	// Unknown kinds use genericColumns which returns NAME, NAMESPACE, AGE
	input := fmt.Sprintf(`{"kind":"CustomResourceList","items":[{"metadata":{"name":"cr-1","namespace":"ns","creationTimestamp":"%s"}}]}`,
		time.Now().Add(-1*time.Hour).Format(time.RFC3339))
	output := summarizeList([]byte(input))

	if !strings.Contains(output, "NAME") {
		t.Errorf("generic output missing 'NAME' header, got:\n%s", output)
	}
	if !strings.Contains(output, "cr-1") {
		t.Errorf("generic output missing item 'cr-1', got:\n%s", output)
	}
}
