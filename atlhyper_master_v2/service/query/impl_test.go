package query

import (
	"context"
	"testing"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/datahub"
	"AtlHyper/atlhyper_master_v2/mq"
	agentmodel "AtlHyper/model_v3/agent"
	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/command"
)

// ==================== Minimal Mocks ====================

// --- datahub.Store ---
type mockStore struct{}

func (m *mockStore) SetSnapshot(clusterID string, snapshot *cluster.ClusterSnapshot) error {
	return nil
}
func (m *mockStore) GetSnapshot(clusterID string) (*cluster.ClusterSnapshot, error) { return nil, nil }
func (m *mockStore) UpdateHeartbeat(clusterID string) error                         { return nil }
func (m *mockStore) GetAgentStatus(clusterID string) (*agentmodel.AgentStatus, error) {
	return nil, nil
}
func (m *mockStore) ListAgents() ([]agentmodel.AgentInfo, error)         { return nil, nil }
func (m *mockStore) GetEvents(clusterID string) ([]cluster.Event, error) { return nil, nil }
func (m *mockStore) GetOTelTimeline(clusterID string, since time.Time) ([]cluster.OTelEntry, error) {
	return nil, nil
}
func (m *mockStore) Start() error { return nil }
func (m *mockStore) Stop() error  { return nil }

var _ datahub.Store = (*mockStore)(nil)

// --- mq.Producer ---
type mockBus struct{}

func (m *mockBus) EnqueueCommand(clusterID, topic string, cmd *command.Command) error { return nil }
func (m *mockBus) GetCommandStatus(cmdID string) (*command.Status, error)             { return nil, nil }
func (m *mockBus) WaitCommandResult(ctx context.Context, cmdID string, timeout time.Duration) (*command.Result, error) {
	return nil, nil
}

var _ mq.Producer = (*mockBus)(nil)

// --- database.ClusterEventRepository ---
type mockEventRepo struct{}

func (m *mockEventRepo) Upsert(ctx context.Context, event *database.ClusterEvent) error { return nil }
func (m *mockEventRepo) UpsertBatch(ctx context.Context, events []*database.ClusterEvent) error {
	return nil
}
func (m *mockEventRepo) ListByCluster(ctx context.Context, clusterID string, opts database.EventQueryOpts) ([]*database.ClusterEvent, error) {
	return nil, nil
}
func (m *mockEventRepo) ListByInvolvedResource(ctx context.Context, clusterID, kind, namespace, name string) ([]*database.ClusterEvent, error) {
	return nil, nil
}
func (m *mockEventRepo) ListByType(ctx context.Context, clusterID, eventType string, since time.Time) ([]*database.ClusterEvent, error) {
	return nil, nil
}
func (m *mockEventRepo) GetLatestEventID(ctx context.Context) (int64, error) { return 0, nil }
func (m *mockEventRepo) GetEventsSince(ctx context.Context, sinceID int64) ([]*database.ClusterEvent, error) {
	return nil, nil
}
func (m *mockEventRepo) DeleteBefore(ctx context.Context, clusterID string, before time.Time) (int64, error) {
	return 0, nil
}
func (m *mockEventRepo) DeleteOldest(ctx context.Context, clusterID string, keepCount int) (int64, error) {
	return 0, nil
}
func (m *mockEventRepo) CountByCluster(ctx context.Context, clusterID string) (int64, error) {
	return 0, nil
}
func (m *mockEventRepo) CountByHour(ctx context.Context, clusterID string, hours int) ([]database.HourlyEventCount, error) {
	return nil, nil
}
func (m *mockEventRepo) CountByHourAndKind(ctx context.Context, clusterID string, hours int) ([]database.HourlyKindCount, error) {
	return nil, nil
}

var _ database.ClusterEventRepository = (*mockEventRepo)(nil)

// --- aiops.Engine ---
type mockAIOpsEngine struct{}

func (m *mockAIOpsEngine) OnSnapshot(clusterID string)                      {}
func (m *mockAIOpsEngine) GetGraph(clusterID string) *aiops.DependencyGraph { return nil }
func (m *mockAIOpsEngine) GetGraphTrace(clusterID, fromKey, direction string, maxDepth int) *aiops.TraceResult {
	return nil
}
func (m *mockAIOpsEngine) GetBaseline(entityKey string) *aiops.EntityBaseline { return nil }
func (m *mockAIOpsEngine) GetClusterRisk(clusterID string) *aiops.ClusterRisk { return nil }
func (m *mockAIOpsEngine) GetEntityRisks(clusterID, sortBy string, limit int) []*aiops.EntityRisk {
	return nil
}
func (m *mockAIOpsEngine) GetEntityRisk(clusterID, entityKey string) *aiops.EntityRiskDetail {
	return nil
}
func (m *mockAIOpsEngine) GetIncidents(ctx context.Context, opts aiops.IncidentQueryOpts) ([]*aiops.Incident, int, error) {
	return nil, 0, nil
}
func (m *mockAIOpsEngine) GetIncidentDetail(ctx context.Context, incidentID string) *aiops.IncidentDetail {
	return nil
}
func (m *mockAIOpsEngine) GetIncidentStats(ctx context.Context, clusterID string, since time.Time) *aiops.IncidentStats {
	return nil
}
func (m *mockAIOpsEngine) GetIncidentPatterns(ctx context.Context, entityKey string, since time.Time) []*aiops.IncidentPattern {
	return nil
}
func (m *mockAIOpsEngine) SetIncidentNotify(fn func(incidentID, severity, trigger string)) {}
func (m *mockAIOpsEngine) Start(ctx context.Context) error                                 { return nil }
func (m *mockAIOpsEngine) Stop() error                                                     { return nil }

var _ aiops.Engine = (*mockAIOpsEngine)(nil)

// --- Admin repo mocks ---
type mockAuditRepo struct{}

func (m *mockAuditRepo) Create(ctx context.Context, log *database.AuditLog) error { return nil }
func (m *mockAuditRepo) List(ctx context.Context, opts database.AuditQueryOpts) ([]*database.AuditLog, error) {
	return nil, nil
}
func (m *mockAuditRepo) Count(ctx context.Context, opts database.AuditQueryOpts) (int64, error) {
	return 0, nil
}

type mockCommandRepo struct{}

func (m *mockCommandRepo) Create(ctx context.Context, cmd *database.CommandHistory) error { return nil }
func (m *mockCommandRepo) Update(ctx context.Context, cmd *database.CommandHistory) error { return nil }
func (m *mockCommandRepo) GetByCommandID(ctx context.Context, cmdID string) (*database.CommandHistory, error) {
	return nil, nil
}
func (m *mockCommandRepo) ListByCluster(ctx context.Context, clusterID string, limit, offset int) ([]*database.CommandHistory, error) {
	return nil, nil
}
func (m *mockCommandRepo) ListByUser(ctx context.Context, userID int64, limit, offset int) ([]*database.CommandHistory, error) {
	return nil, nil
}
func (m *mockCommandRepo) List(ctx context.Context, opts database.CommandQueryOpts) ([]*database.CommandHistory, error) {
	return nil, nil
}
func (m *mockCommandRepo) Count(ctx context.Context, opts database.CommandQueryOpts) (int64, error) {
	return 0, nil
}

type mockNotifyRepo struct{}

func (m *mockNotifyRepo) Create(ctx context.Context, ch *database.NotifyChannel) error { return nil }
func (m *mockNotifyRepo) Update(ctx context.Context, ch *database.NotifyChannel) error { return nil }
func (m *mockNotifyRepo) Delete(ctx context.Context, id int64) error                   { return nil }
func (m *mockNotifyRepo) GetByID(ctx context.Context, id int64) (*database.NotifyChannel, error) {
	return nil, nil
}
func (m *mockNotifyRepo) GetByType(ctx context.Context, channelType string) (*database.NotifyChannel, error) {
	return nil, nil
}
func (m *mockNotifyRepo) List(ctx context.Context) ([]*database.NotifyChannel, error) {
	return nil, nil
}
func (m *mockNotifyRepo) ListEnabled(ctx context.Context) ([]*database.NotifyChannel, error) {
	return nil, nil
}

type mockSettingsRepo struct{}

func (m *mockSettingsRepo) Get(ctx context.Context, key string) (*database.Setting, error) {
	return nil, nil
}
func (m *mockSettingsRepo) GetByPrefix(ctx context.Context, prefix string) ([]*database.Setting, error) {
	return nil, nil
}
func (m *mockSettingsRepo) Set(ctx context.Context, setting *database.Setting) error { return nil }
func (m *mockSettingsRepo) Delete(ctx context.Context, key string) error             { return nil }
func (m *mockSettingsRepo) List(ctx context.Context) ([]*database.Setting, error)    { return nil, nil }

type mockAIProviderRepo struct{}

func (m *mockAIProviderRepo) Create(ctx context.Context, p *database.AIProvider) error { return nil }
func (m *mockAIProviderRepo) Update(ctx context.Context, p *database.AIProvider) error { return nil }
func (m *mockAIProviderRepo) Delete(ctx context.Context, id int64) error               { return nil }
func (m *mockAIProviderRepo) GetByID(ctx context.Context, id int64) (*database.AIProvider, error) {
	return nil, nil
}
func (m *mockAIProviderRepo) List(ctx context.Context) ([]*database.AIProvider, error) {
	return nil, nil
}
func (m *mockAIProviderRepo) UpdateRoles(ctx context.Context, id int64, roles []string) error {
	return nil
}
func (m *mockAIProviderRepo) FindByRole(ctx context.Context, role string) (*database.AIProvider, error) {
	return nil, nil
}
func (m *mockAIProviderRepo) IncrementUsage(ctx context.Context, id int64, requests, tokens int64, cost float64) error {
	return nil
}
func (m *mockAIProviderRepo) UpdateStatus(ctx context.Context, id int64, status, errorMsg string) error {
	return nil
}

type mockAISettingsRepo struct{}

func (m *mockAISettingsRepo) Get(ctx context.Context) (*database.AISettings, error)      { return nil, nil }
func (m *mockAISettingsRepo) Update(ctx context.Context, cfg *database.AISettings) error { return nil }

type mockAIModelRepo struct{}

func (m *mockAIModelRepo) Create(ctx context.Context, mod *database.AIProviderModel) error {
	return nil
}
func (m *mockAIModelRepo) Delete(ctx context.Context, id int64) error { return nil }
func (m *mockAIModelRepo) GetByID(ctx context.Context, id int64) (*database.AIProviderModel, error) {
	return nil, nil
}
func (m *mockAIModelRepo) ListByProvider(ctx context.Context, provider string) ([]*database.AIProviderModel, error) {
	return nil, nil
}
func (m *mockAIModelRepo) ListAll(ctx context.Context) ([]*database.AIProviderModel, error) {
	return nil, nil
}
func (m *mockAIModelRepo) GetDefaultModel(ctx context.Context, provider string) (*database.AIProviderModel, error) {
	return nil, nil
}

type mockAIBudgetRepo struct{}

func (m *mockAIBudgetRepo) Get(ctx context.Context, role string) (*database.AIRoleBudget, error) {
	return nil, nil
}
func (m *mockAIBudgetRepo) ListAll(ctx context.Context) ([]*database.AIRoleBudget, error) {
	return nil, nil
}
func (m *mockAIBudgetRepo) Upsert(ctx context.Context, budget *database.AIRoleBudget) error {
	return nil
}
func (m *mockAIBudgetRepo) Delete(ctx context.Context, role string) error { return nil }
func (m *mockAIBudgetRepo) IncrementUsage(ctx context.Context, role string, inputTokens, outputTokens int) error {
	return nil
}
func (m *mockAIBudgetRepo) ResetDailyUsage(ctx context.Context, role string) error   { return nil }
func (m *mockAIBudgetRepo) ResetMonthlyUsage(ctx context.Context, role string) error { return nil }

type mockAIReportRepo struct{}

func (m *mockAIReportRepo) Create(ctx context.Context, report *database.AIReport) error { return nil }
func (m *mockAIReportRepo) GetByID(ctx context.Context, id int64) (*database.AIReport, error) {
	return nil, nil
}
func (m *mockAIReportRepo) ListByIncident(ctx context.Context, incidentID string) ([]*database.AIReport, error) {
	return nil, nil
}
func (m *mockAIReportRepo) ListByCluster(ctx context.Context, clusterID, role string, limit int) ([]*database.AIReport, error) {
	return nil, nil
}
func (m *mockAIReportRepo) ListRecent(ctx context.Context, role string, limit, offset int) ([]*database.AIReport, int, error) {
	return nil, 0, nil
}
func (m *mockAIReportRepo) CountByClusterAndRole(ctx context.Context, clusterID, role string, since time.Time) (int, error) {
	return 0, nil
}
func (m *mockAIReportRepo) DeleteBefore(ctx context.Context, before time.Time) (int64, error) {
	return 0, nil
}
func (m *mockAIReportRepo) UpdateInvestigationSteps(ctx context.Context, id int64, steps string) error {
	return nil
}
func (m *mockAIReportRepo) UpdateResult(ctx context.Context, id int64, report *database.AIReport) error {
	return nil
}

// ==================== 测试用例 ====================

func TestNewQueryService_AllDeps(t *testing.T) {
	store := &mockStore{}
	bus := &mockBus{}
	eventRepo := &mockEventRepo{}
	sloRepo := &mockSLORepo{} // from slo_test.go (same package)
	engine := &mockAIOpsEngine{}

	auditRepo := &mockAuditRepo{}
	commandRepo := &mockCommandRepo{}
	notifyRepo := &mockNotifyRepo{}
	settingsRepo := &mockSettingsRepo{}
	aiProviderRepo := &mockAIProviderRepo{}
	aiSettingsRepo := &mockAISettingsRepo{}
	aiModelRepo := &mockAIModelRepo{}
	aiBudgetRepo := &mockAIBudgetRepo{}
	aiReportRepo := &mockAIReportRepo{}

	svc := NewQueryService(QueryServiceDeps{
		Store:       store,
		Bus:         bus,
		EventRepo:   eventRepo,
		SLORepo:     sloRepo,
		AIOpsEngine: engine,
		AdminRepos: AdminRepos{
			Audit:      auditRepo,
			Command:    commandRepo,
			Notify:     notifyRepo,
			Settings:   settingsRepo,
			AIProvider: aiProviderRepo,
			AISettings: aiSettingsRepo,
			AIModel:    aiModelRepo,
			AIBudget:   aiBudgetRepo,
			AIReport:   aiReportRepo,
		},
	})

	// 验证所有字段正确注入（在 package query 内部可访问未导出字段）
	if svc.store == nil {
		t.Error("store not injected")
	}
	if svc.bus == nil {
		t.Error("bus not injected")
	}
	if svc.eventRepo == nil {
		t.Error("eventRepo not injected")
	}
	if svc.sloRepo == nil {
		t.Error("sloRepo not injected")
	}
	if svc.aiopsEngine == nil {
		t.Error("aiopsEngine not injected")
	}
	if svc.auditRepo == nil {
		t.Error("auditRepo not injected")
	}
	if svc.commandRepo == nil {
		t.Error("commandRepo not injected")
	}
	if svc.notifyRepo == nil {
		t.Error("notifyRepo not injected")
	}
	if svc.settingsRepo == nil {
		t.Error("settingsRepo not injected")
	}
	if svc.aiProviderRepo == nil {
		t.Error("aiProviderRepo not injected")
	}
	if svc.aiSettingsRepo == nil {
		t.Error("aiSettingsRepo not injected")
	}
	if svc.aiModelRepo == nil {
		t.Error("aiModelRepo not injected")
	}
	if svc.aiBudgetRepo == nil {
		t.Error("aiBudgetRepo not injected")
	}
	if svc.aiReportRepo == nil {
		t.Error("aiReportRepo not injected")
	}
}

func TestNewQueryService_OptionalNil(t *testing.T) {
	store := &mockStore{}
	bus := &mockBus{}
	eventRepo := &mockEventRepo{}
	sloRepo := &mockSLORepo{}

	// AIOpsEngine 和 AIOpsAI 为 nil — 可选依赖
	svc := NewQueryService(QueryServiceDeps{
		Store:     store,
		Bus:       bus,
		EventRepo: eventRepo,
		SLORepo:   sloRepo,
		// AIOpsEngine: nil（省略）
		// AIOpsAI:     nil（省略）
		AdminRepos: AdminRepos{
			Audit:      &mockAuditRepo{},
			Command:    &mockCommandRepo{},
			Notify:     &mockNotifyRepo{},
			Settings:   &mockSettingsRepo{},
			AIProvider: &mockAIProviderRepo{},
			AISettings: &mockAISettingsRepo{},
			AIModel:    &mockAIModelRepo{},
			AIBudget:   &mockAIBudgetRepo{},
			AIReport:   &mockAIReportRepo{},
		},
	})

	// 必需字段正确注入
	if svc.store == nil {
		t.Error("store not injected")
	}
	if svc.bus == nil {
		t.Error("bus not injected")
	}
	if svc.eventRepo == nil {
		t.Error("eventRepo not injected")
	}
	if svc.sloRepo == nil {
		t.Error("sloRepo not injected")
	}

	// 可选字段为 nil，不 panic
	if svc.aiopsEngine != nil {
		t.Error("aiopsEngine should be nil when not provided")
	}
	if svc.aiopsAI != nil {
		t.Error("aiopsAI should be nil when not provided")
	}
}
