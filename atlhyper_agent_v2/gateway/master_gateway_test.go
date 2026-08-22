package gateway

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/command"
)

// newTestGateway creates a masterGateway pointing at the given test server URL.
func newTestGateway(serverURL string) *masterGateway {
	return &masterGateway{
		masterURL:  serverURL,
		clusterID:  "test-cluster",
		httpClient: &http.Client{Timeout: 5 * time.Second},
	}
}

// ---------------------------------------------------------------------------
// PushSnapshot
// ---------------------------------------------------------------------------

func TestPushSnapshot_RequestFormat(t *testing.T) {
	var capturedReq *http.Request
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	gw := newTestGateway(server.URL)
	snapshot := &cluster.ClusterSnapshot{ClusterID: "test-cluster"}

	err := gw.PushSnapshot(context.Background(), snapshot)
	if err != nil {
		t.Fatalf("PushSnapshot returned error: %v", err)
	}

	// Method
	if capturedReq.Method != http.MethodPost {
		t.Errorf("method: got %q, want %q", capturedReq.Method, http.MethodPost)
	}

	// Path
	if capturedReq.URL.Path != "/agent/snapshot" {
		t.Errorf("path: got %q, want %q", capturedReq.URL.Path, "/agent/snapshot")
	}

	// Content-Type
	if ct := capturedReq.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: got %q, want %q", ct, "application/json")
	}

	// Content-Encoding
	if ce := capturedReq.Header.Get("Content-Encoding"); ce != "gzip" {
		t.Errorf("Content-Encoding: got %q, want %q", ce, "gzip")
	}

	// X-Cluster-ID
	if cid := capturedReq.Header.Get("X-Cluster-ID"); cid != "test-cluster" {
		t.Errorf("X-Cluster-ID: got %q, want %q", cid, "test-cluster")
	}

	// Body should not be empty (gzip compressed)
	if len(capturedBody) == 0 {
		t.Error("request body is empty, expected gzip compressed data")
	}
}

func TestPushSnapshot_CompressedPayload(t *testing.T) {
	var decompressedData []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gr, err := gzip.NewReader(r.Body)
		if err != nil {
			t.Errorf("failed to create gzip reader: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		defer gr.Close()

		decompressedData, err = io.ReadAll(gr)
		if err != nil {
			t.Errorf("failed to read decompressed data: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	gw := newTestGateway(server.URL)
	snapshot := &cluster.ClusterSnapshot{
		ClusterID: "test-cluster",
		Nodes:     []cluster.Node{{Summary: cluster.NodeSummary{Name: "node-1"}}},
	}

	err := gw.PushSnapshot(context.Background(), snapshot)
	if err != nil {
		t.Fatalf("PushSnapshot returned error: %v", err)
	}

	// Decompressed body should be valid JSON
	if !json.Valid(decompressedData) {
		t.Fatal("decompressed body is not valid JSON")
	}

	// Decompressed JSON should contain the clusterID
	var parsed map[string]any
	if err := json.Unmarshal(decompressedData, &parsed); err != nil {
		t.Fatalf("failed to unmarshal decompressed JSON: %v", err)
	}

	if cid, ok := parsed["clusterId"].(string); !ok || cid != "test-cluster" {
		t.Errorf("clusterId in decompressed payload: got %v, want %q", parsed["clusterId"], "test-cluster")
	}
}

func TestPushSnapshot_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer server.Close()

	gw := newTestGateway(server.URL)
	snapshot := &cluster.ClusterSnapshot{ClusterID: "test-cluster"}

	err := gw.PushSnapshot(context.Background(), snapshot)
	if err == nil {
		t.Fatal("expected error for 500 response, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should contain status code 500, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// PollCommands
// ---------------------------------------------------------------------------

func TestPollCommands_HasCommand(t *testing.T) {
	respBody := `{"hasCommand":true,"command":{"id":"cmd-1","action":"scale","namespace":"default","name":"nginx","params":{"replicas":3}}}`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(respBody))
	}))
	defer server.Close()

	gw := newTestGateway(server.URL)
	cmds, err := gw.PollCommands(context.Background(), "ops")
	if err != nil {
		t.Fatalf("PollCommands returned error: %v", err)
	}

	if len(cmds) != 1 {
		t.Fatalf("expected 1 command, got %d", len(cmds))
	}

	if cmds[0].ID != "cmd-1" {
		t.Errorf("command ID: got %q, want %q", cmds[0].ID, "cmd-1")
	}

	if cmds[0].Action != "scale" {
		t.Errorf("command Action: got %q, want %q", cmds[0].Action, "scale")
	}

	if cmds[0].Namespace != "default" {
		t.Errorf("command Namespace: got %q, want %q", cmds[0].Namespace, "default")
	}

	if cmds[0].Name != "nginx" {
		t.Errorf("command Name: got %q, want %q", cmds[0].Name, "nginx")
	}
}

func TestPollCommands_NoContent(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	gw := newTestGateway(server.URL)
	cmds, err := gw.PollCommands(context.Background(), "ops")
	if err != nil {
		t.Fatalf("PollCommands returned error: %v", err)
	}
	if cmds != nil {
		t.Errorf("expected nil commands for 204, got %v", cmds)
	}
}

func TestPollCommands_NoCommand(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"has_command":false}`))
	}))
	defer server.Close()

	gw := newTestGateway(server.URL)
	cmds, err := gw.PollCommands(context.Background(), "ops")
	if err != nil {
		t.Fatalf("PollCommands returned error: %v", err)
	}
	if cmds != nil {
		t.Errorf("expected nil commands when has_command=false, got %v", cmds)
	}
}

func TestPollCommands_RequestFormat(t *testing.T) {
	var capturedReq *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	gw := newTestGateway(server.URL)
	_, err := gw.PollCommands(context.Background(), "ops")
	if err != nil {
		t.Fatalf("PollCommands returned error: %v", err)
	}

	// Method
	if capturedReq.Method != http.MethodGet {
		t.Errorf("method: got %q, want %q", capturedReq.Method, http.MethodGet)
	}

	// Query parameters
	rawQuery := capturedReq.URL.RawQuery
	if !strings.Contains(rawQuery, "cluster_id=test-cluster") {
		t.Errorf("URL should contain cluster_id=test-cluster, got query: %q", rawQuery)
	}
	if !strings.Contains(rawQuery, "topic=ops") {
		t.Errorf("URL should contain topic=ops, got query: %q", rawQuery)
	}

	// X-Cluster-ID header
	if cid := capturedReq.Header.Get("X-Cluster-ID"); cid != "test-cluster" {
		t.Errorf("X-Cluster-ID: got %q, want %q", cid, "test-cluster")
	}
}

func TestPollCommands_ContextCanceled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"has_command":false}`))
	}))
	defer server.Close()

	gw := newTestGateway(server.URL)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before making the request

	cmds, err := gw.PollCommands(ctx, "ops")
	// When context is canceled, PollCommands returns nil, nil
	if err != nil {
		t.Errorf("expected nil error for canceled context, got: %v", err)
	}
	if cmds != nil {
		t.Errorf("expected nil commands for canceled context, got: %v", cmds)
	}
}

// ---------------------------------------------------------------------------
// ReportResult
// ---------------------------------------------------------------------------

func TestReportResult_RequestFormat(t *testing.T) {
	var capturedReq *http.Request
	var capturedBody []byte

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		capturedBody, _ = io.ReadAll(r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	gw := newTestGateway(server.URL)
	result := &command.Result{
		CommandID:  "cmd-42",
		Success:    true,
		Output:     "scaled to 3 replicas",
		ExecutedAt: time.Now(),
	}

	err := gw.ReportResult(context.Background(), result)
	if err != nil {
		t.Fatalf("ReportResult returned error: %v", err)
	}

	// Method
	if capturedReq.Method != http.MethodPost {
		t.Errorf("method: got %q, want %q", capturedReq.Method, http.MethodPost)
	}

	// Path
	if capturedReq.URL.Path != "/agent/result" {
		t.Errorf("path: got %q, want %q", capturedReq.URL.Path, "/agent/result")
	}

	// Content-Type
	if ct := capturedReq.Header.Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type: got %q, want %q", ct, "application/json")
	}

	// X-Cluster-ID
	if cid := capturedReq.Header.Get("X-Cluster-ID"); cid != "test-cluster" {
		t.Errorf("X-Cluster-ID: got %q, want %q", cid, "test-cluster")
	}

	// Body should contain commandId
	bodyStr := string(capturedBody)
	if !strings.Contains(bodyStr, "cmd-42") {
		t.Errorf("request body should contain commandId 'cmd-42', got: %s", bodyStr)
	}

	// Verify the body is valid JSON with expected fields
	var parsed map[string]any
	if err := json.Unmarshal(capturedBody, &parsed); err != nil {
		t.Fatalf("request body is not valid JSON: %v", err)
	}
	if parsed["commandId"] != "cmd-42" {
		t.Errorf("commandId: got %v, want %q", parsed["commandId"], "cmd-42")
	}
	if parsed["success"] != true {
		t.Errorf("success: got %v, want true", parsed["success"])
	}
}

// ---------------------------------------------------------------------------
// Heartbeat
// ---------------------------------------------------------------------------

func TestHeartbeat_Success(t *testing.T) {
	var capturedReq *http.Request

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedReq = r
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	gw := newTestGateway(server.URL)

	err := gw.Heartbeat(context.Background())
	if err != nil {
		t.Fatalf("Heartbeat returned error: %v", err)
	}

	// Method
	if capturedReq.Method != http.MethodPost {
		t.Errorf("method: got %q, want %q", capturedReq.Method, http.MethodPost)
	}

	// Path
	if capturedReq.URL.Path != "/agent/heartbeat" {
		t.Errorf("path: got %q, want %q", capturedReq.URL.Path, "/agent/heartbeat")
	}

	// X-Cluster-ID
	if cid := capturedReq.Header.Get("X-Cluster-ID"); cid != "test-cluster" {
		t.Errorf("X-Cluster-ID: got %q, want %q", cid, "test-cluster")
	}
}

func TestHeartbeat_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer server.Close()

	gw := newTestGateway(server.URL)

	err := gw.Heartbeat(context.Background())
	if err == nil {
		t.Fatal("expected error for 503 response, got nil")
	}
	if !strings.Contains(err.Error(), "503") {
		t.Errorf("error should contain status code 503, got: %v", err)
	}
}
