package query

import (
	"context"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/database"
	agentmodel "AtlHyper/model_v3/agent"
	"AtlHyper/model_v3/cluster"
	slomodel "AtlHyper/model_v3/slo"
)

// ==================== Mock: database.SLORepository ====================

type mockSLORepo struct {
	targets  []*database.SLOTarget
	mappings []*database.SLORouteMapping
	domains  []string
	err      error
}

func (m *mockSLORepo) GetTargets(ctx context.Context, clusterID string) ([]*database.SLOTarget, error) {
	return m.targets, m.err
}
func (m *mockSLORepo) GetTargetsByHost(ctx context.Context, clusterID, host string) ([]*database.SLOTarget, error) {
	return m.targets, m.err
}
func (m *mockSLORepo) UpsertTarget(ctx context.Context, t *database.SLOTarget) error {
	return m.err
}
func (m *mockSLORepo) DeleteTarget(ctx context.Context, clusterID, host, timeRange string) error {
	return m.err
}
func (m *mockSLORepo) UpsertRouteMapping(ctx context.Context, rm *database.SLORouteMapping) error {
	return m.err
}
func (m *mockSLORepo) GetRouteMappingByServiceKey(ctx context.Context, clusterID, serviceKey string) (*database.SLORouteMapping, error) {
	if len(m.mappings) > 0 {
		return m.mappings[0], m.err
	}
	return nil, m.err
}
func (m *mockSLORepo) GetRouteMappingsByDomain(ctx context.Context, clusterID, domain string) ([]*database.SLORouteMapping, error) {
	return m.mappings, m.err
}
func (m *mockSLORepo) GetAllRouteMappings(ctx context.Context, clusterID string) ([]*database.SLORouteMapping, error) {
	return m.mappings, m.err
}
func (m *mockSLORepo) GetAllDomains(ctx context.Context, clusterID string) ([]string, error) {
	return m.domains, m.err
}
func (m *mockSLORepo) DeleteRouteMapping(ctx context.Context, clusterID, serviceKey string) error {
	return m.err
}

// ==================== 测试用例 ====================

func TestGetSLOTargets_Success(t *testing.T) {
	now := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	repo := &mockSLORepo{
		targets: []*database.SLOTarget{
			{
				ID: 1, ClusterID: "cluster-1", Host: "example.com",
				TimeRange: "1d", AvailabilityTarget: 99.9, P95LatencyTarget: 200,
				CreatedAt: now, UpdatedAt: now,
			},
			{
				ID: 2, ClusterID: "cluster-1", Host: "api.example.com",
				TimeRange: "7d", AvailabilityTarget: 99.5, P95LatencyTarget: 500,
				CreatedAt: now, UpdatedAt: now,
			},
		},
	}

	// QueryService 在 package query 内部，可以直接构造
	svc := &QueryService{sloRepo: repo}

	results, err := svc.GetSLOTargets(context.Background(), "cluster-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 targets, got %d", len(results))
	}

	// 验证 database.SLOTarget → model.SLOTargetResponse 转换
	r := results[0]
	if r.ID != 1 {
		t.Errorf("expected ID=1, got %d", r.ID)
	}
	if r.Host != "example.com" {
		t.Errorf("expected Host=example.com, got %s", r.Host)
	}
	if r.AvailabilityTarget != 99.9 {
		t.Errorf("expected AvailabilityTarget=99.9, got %f", r.AvailabilityTarget)
	}
	// 验证时间格式化（ISO 8601）
	expected := "2025-06-01T12:00:00Z"
	if r.CreatedAt != expected {
		t.Errorf("expected CreatedAt=%s, got %s", expected, r.CreatedAt)
	}
}

func TestGetSLOTargets_Empty(t *testing.T) {
	repo := &mockSLORepo{targets: []*database.SLOTarget{}}
	svc := &QueryService{sloRepo: repo}

	results, err := svc.GetSLOTargets(context.Background(), "cluster-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if results == nil {
		t.Fatal("expected non-nil empty slice, got nil")
	}
	if len(results) != 0 {
		t.Fatalf("expected 0 targets, got %d", len(results))
	}
}

func TestGetSLORouteMappingsByDomain_Success(t *testing.T) {
	repo := &mockSLORepo{
		mappings: []*database.SLORouteMapping{
			{
				Domain: "example.com", PathPrefix: "/api",
				IngressName: "ing-1", Namespace: "default", TLS: true,
				ServiceKey:  "default-svc-80@kubernetes",
				ServiceName: "svc", ServicePort: 80,
			},
		},
	}
	svc := &QueryService{sloRepo: repo}

	results, err := svc.GetSLORouteMappingsByDomain(context.Background(), "cluster-1", "example.com")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1 mapping, got %d", len(results))
	}

	// 验证 database.SLORouteMapping → model.SLORouteMapping 转换
	m := results[0]
	if m.Domain != "example.com" {
		t.Errorf("expected Domain=example.com, got %s", m.Domain)
	}
	if m.ServiceKey != "default-svc-80@kubernetes" {
		t.Errorf("expected ServiceKey=default-svc-80@kubernetes, got %s", m.ServiceKey)
	}
	if m.ServicePort != 80 {
		t.Errorf("expected ServicePort=80, got %d", m.ServicePort)
	}
}

func TestGetSLOAllDomains_Success(t *testing.T) {
	repo := &mockSLORepo{
		domains: []string{"example.com", "api.example.com"},
	}
	svc := &QueryService{sloRepo: repo}

	results, err := svc.GetSLOAllDomains(context.Background(), "cluster-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 domains, got %d", len(results))
	}
	if results[0] != "example.com" {
		t.Errorf("expected first domain=example.com, got %s", results[0])
	}
}

func TestGetSLORouteMappingByServiceKey_NotFound(t *testing.T) {
	repo := &mockSLORepo{mappings: nil}
	svc := &QueryService{sloRepo: repo}

	result, err := svc.GetSLORouteMappingByServiceKey(context.Background(), "cluster-1", "nonexistent@kubernetes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != nil {
		t.Errorf("expected nil for not-found, got %+v", result)
	}
}

// ==================== Phase 1: 纯辅助函数测试 ====================

func TestDetermineMeshStatus_Healthy(t *testing.T) {
	// errRate ≤ 1 且 p99 ≤ 500 → healthy
	tests := []struct {
		errRate, p99 float64
	}{
		{0, 100},
		{0.5, 500},
		{1.0, 500},
	}
	for _, tt := range tests {
		got := determineMeshStatus(tt.errRate, tt.p99)
		if got != "healthy" {
			t.Errorf("determineMeshStatus(%v, %v) = %q, want healthy", tt.errRate, tt.p99, got)
		}
	}
}

func TestDetermineMeshStatus_Warning(t *testing.T) {
	// errRate > 1 (但 ≤ 5) 或 p99 > 500 → warning
	tests := []struct {
		errRate, p99 float64
	}{
		{1.1, 200}, // errRate > 1
		{0.5, 501}, // p99 > 500
		{5.0, 100}, // errRate == 5（≤ 5 但 > 1）
	}
	for _, tt := range tests {
		got := determineMeshStatus(tt.errRate, tt.p99)
		if got != "warning" {
			t.Errorf("determineMeshStatus(%v, %v) = %q, want warning", tt.errRate, tt.p99, got)
		}
	}
}

func TestDetermineMeshStatus_Critical(t *testing.T) {
	// errRate > 5 → critical（优先于 warning）
	tests := []struct {
		errRate, p99 float64
	}{
		{5.1, 100},
		{10, 1000},
		{100, 0},
	}
	for _, tt := range tests {
		got := determineMeshStatus(tt.errRate, tt.p99)
		if got != "critical" {
			t.Errorf("determineMeshStatus(%v, %v) = %q, want critical", tt.errRate, tt.p99, got)
		}
	}
}

func TestTotalFromStatusCodes(t *testing.T) {
	codes := []slomodel.StatusCodeCount{
		{Code: "200", Count: 500},
		{Code: "404", Count: 30},
		{Code: "500", Count: 10},
	}
	got := totalFromStatusCodes(codes)
	if got != 540 {
		t.Fatalf("totalFromStatusCodes = %d, want 540", got)
	}
}

func TestTotalFromStatusCodes_Empty(t *testing.T) {
	if got := totalFromStatusCodes(nil); got != 0 {
		t.Fatalf("totalFromStatusCodes(nil) = %d, want 0", got)
	}
	if got := totalFromStatusCodes([]slomodel.StatusCodeCount{}); got != 0 {
		t.Fatalf("totalFromStatusCodes([]) = %d, want 0", got)
	}
}

func TestGetTimeStart(t *testing.T) {
	now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)

	tests := []struct {
		name      string
		timeRange string
		want      time.Duration // 期望的 now - result 差值
	}{
		{"1h", "1h", time.Hour},
		{"6h", "6h", 6 * time.Hour},
		{"24h", "24h", 24 * time.Hour},
		{"1d alias", "1d", 24 * time.Hour},
		{"7d", "7d", 7 * 24 * time.Hour},
		{"30d", "30d", 30 * 24 * time.Hour},
		{"unknown defaults to 24h", "unknown", 24 * time.Hour},
		{"empty defaults to 24h", "", 24 * time.Hour},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := getTimeStart(now, tt.timeRange)
			diff := now.Sub(result)
			if diff != tt.want {
				t.Errorf("getTimeStart(now, %q): diff = %v, want %v", tt.timeRange, diff, tt.want)
			}
		})
	}
}

// ==================== Phase 2: Mock Store + GetMeshTopology / GetServiceDetail ====================

// mockStoreForSLO 最小 mock，只实现 GetSnapshot，其余空实现
type mockStoreForSLO struct {
	snapshot *cluster.ClusterSnapshot
}

func (m *mockStoreForSLO) SetSnapshot(clusterID string, snapshot *cluster.ClusterSnapshot) error {
	return nil
}
func (m *mockStoreForSLO) GetSnapshot(clusterID string) (*cluster.ClusterSnapshot, error) {
	return m.snapshot, nil
}
func (m *mockStoreForSLO) UpdateHeartbeat(clusterID string) error { return nil }
func (m *mockStoreForSLO) GetAgentStatus(clusterID string) (*agentmodel.AgentStatus, error) {
	return nil, nil
}
func (m *mockStoreForSLO) ListAgents() ([]agentmodel.AgentInfo, error)         { return nil, nil }
func (m *mockStoreForSLO) GetEvents(clusterID string) ([]cluster.Event, error) { return nil, nil }
func (m *mockStoreForSLO) GetOTelTimeline(clusterID string, since time.Time) ([]cluster.OTelEntry, error) {
	return nil, nil
}
func (m *mockStoreForSLO) Start() error { return nil }
func (m *mockStoreForSLO) Stop() error  { return nil }
