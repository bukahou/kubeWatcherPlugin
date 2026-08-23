// atlhyper_master_v2/service/query/impl.go
// QueryService 结构体与构造函数
//
// 各功能域实现分布在:
//
//	k8s.go        — K8s 资源快照查询 (19 个方法)
//	otel.go       — OTel 快照/时间线查询
//	overview.go   — 集群概览、Agent 状态、事件、单资源查询
//	slo.go        — SLO 服务网格查询
//	aiops.go      — AIOps 查询与 AI 增强
package query

import (
	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/aiops/enricher"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/datahub"
	"AtlHyper/atlhyper_master_v2/mq"
)

// QueryService Query 层实现
type QueryService struct {
	store       datahub.Store
	bus         mq.Producer
	eventRepo   database.ClusterEventRepository
	sloRepo     database.SLORepository
	aiopsEngine aiops.Engine
	aiopsAI     *enricher.Enricher

	// Admin repositories（管理查询）
	auditRepo      database.AuditRepository
	commandRepo    database.CommandHistoryRepository
	notifyRepo     database.NotifyChannelRepository
	settingsRepo   database.SettingsRepository
	aiProviderRepo database.AIProviderRepository
	aiSettingsRepo database.AISettingsRepository
	aiModelRepo    database.AIProviderModelRepository
	aiBudgetRepo   database.AIRoleBudgetRepository
	aiReportRepo   database.AIReportRepository
}

// AdminRepos 管理查询所需的 Repository 集合
// 对应 QueryAdmin 接口的所有方法所需依赖
type AdminRepos struct {
	Audit      database.AuditRepository
	Command    database.CommandHistoryRepository
	Notify     database.NotifyChannelRepository
	Settings   database.SettingsRepository
	AIProvider database.AIProviderRepository
	AISettings database.AISettingsRepository
	AIModel    database.AIProviderModelRepository
	AIBudget   database.AIRoleBudgetRepository
	AIReport   database.AIReportRepository
}

// QueryServiceDeps QueryService 全部依赖
// 严格限定：每个字段对应 QueryService 已有的 struct 字段，禁止新增未使用的依赖
type QueryServiceDeps struct {
	Store       datahub.Store                   // 必需
	Bus         mq.Producer                     // 必需
	EventRepo   database.ClusterEventRepository // 必需（Alert Trends）
	SLORepo     database.SLORepository          // 必需（Phase 2 新增）
	AIOpsEngine aiops.Engine                    // 可选，nil = AIOps 查询返回空
	AIOpsAI     *enricher.Enricher              // 可选，nil = AI 增强禁用
	AdminRepos  AdminRepos                      // 必需（管理查询）
}

// NewQueryService 创建 QueryService（全部依赖通过构造函数注入）
func NewQueryService(deps QueryServiceDeps) *QueryService {
	return &QueryService{
		store:          deps.Store,
		bus:            deps.Bus,
		eventRepo:      deps.EventRepo,
		sloRepo:        deps.SLORepo,
		aiopsEngine:    deps.AIOpsEngine,
		aiopsAI:        deps.AIOpsAI,
		auditRepo:      deps.AdminRepos.Audit,
		commandRepo:    deps.AdminRepos.Command,
		notifyRepo:     deps.AdminRepos.Notify,
		settingsRepo:   deps.AdminRepos.Settings,
		aiProviderRepo: deps.AdminRepos.AIProvider,
		aiSettingsRepo: deps.AdminRepos.AISettings,
		aiModelRepo:    deps.AdminRepos.AIModel,
		aiBudgetRepo:   deps.AdminRepos.AIBudget,
		aiReportRepo:   deps.AdminRepos.AIReport,
	}
}
