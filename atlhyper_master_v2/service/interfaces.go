// atlhyper_master_v2/service/interfaces.go
// Service 接口定义
// Query: 只读查询 (datahub.Store + mq.Producer 读取)
// Ops:   写入操作 (mq.Producer 写入)
package service

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/atlhyper_master_v2/aiops/enricher"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v3/agent"
	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/command"
)

// ================================================================
// 子接口（按功能域划分）
// ================================================================

// QueryK8s K8s 资源快照查询
type QueryK8s interface {
	GetSnapshot(ctx context.Context, clusterID string) (*cluster.ClusterSnapshot, error)
	GetPods(ctx context.Context, clusterID string, opts model.PodQueryOpts) ([]cluster.Pod, error)
	GetNodes(ctx context.Context, clusterID string) ([]cluster.Node, error)
	GetDeployments(ctx context.Context, clusterID string, namespace string) ([]cluster.Deployment, error)
	GetServices(ctx context.Context, clusterID string, namespace string) ([]cluster.Service, error)
	GetIngresses(ctx context.Context, clusterID string, namespace string) ([]cluster.Ingress, error)
	GetConfigMaps(ctx context.Context, clusterID string, namespace string) ([]cluster.ConfigMap, error)
	GetSecrets(ctx context.Context, clusterID string, namespace string) ([]cluster.Secret, error)
	GetNamespaces(ctx context.Context, clusterID string) ([]cluster.Namespace, error)
	GetDaemonSets(ctx context.Context, clusterID string, namespace string) ([]cluster.DaemonSet, error)
	GetStatefulSets(ctx context.Context, clusterID string, namespace string) ([]cluster.StatefulSet, error)
	GetJobs(ctx context.Context, clusterID string, namespace string) ([]cluster.Job, error)
	GetCronJobs(ctx context.Context, clusterID string, namespace string) ([]cluster.CronJob, error)
	GetPersistentVolumes(ctx context.Context, clusterID string) ([]cluster.PersistentVolume, error)
	GetPersistentVolumeClaims(ctx context.Context, clusterID string, namespace string) ([]cluster.PersistentVolumeClaim, error)
	GetNetworkPolicies(ctx context.Context, clusterID string, namespace string) ([]cluster.NetworkPolicy, error)
	GetResourceQuotas(ctx context.Context, clusterID string, namespace string) ([]cluster.ResourceQuota, error)
	GetLimitRanges(ctx context.Context, clusterID string, namespace string) ([]cluster.LimitRange, error)
	GetServiceAccounts(ctx context.Context, clusterID string, namespace string) ([]cluster.ServiceAccount, error)
}

// QueryOTel OTel 快照/时间线查询
type QueryOTel interface {
	GetOTelSnapshot(ctx context.Context, clusterID string) (*cluster.OTelSnapshot, error)
	GetOTelTimeline(ctx context.Context, clusterID string, since time.Time) ([]cluster.OTelEntry, error)
	// GetHardwareHealth 硬件健康矩阵（阈值判定已在 Master 完成，前端只渲染）
	GetHardwareHealth(ctx context.Context, clusterID string) (*model.HardwareHealthResponse, error)
	// GetNodeComparison 节点横向对比表（硬件列复用 GetHardwareHealth 的判定）
	GetNodeComparison(ctx context.Context, clusterID string) (*model.NodeComparisonResponse, error)
	// GetSignalFreshness 各信号的数据新鲜度，用于区分「没有流量」和「采集挂了」
	GetSignalFreshness(ctx context.Context, clusterID string) (*model.FreshnessResponse, error)
}

// QuerySLO SLO 服务网格查询 + SLO 目标/路由映射查询
type QuerySLO interface {
	// SLO 目标查询（返回 model 类型，非 database 类型）
	GetSLOTargets(ctx context.Context, clusterID string) ([]model.SLOTargetResponse, error)
	// SLO 路由映射查询（返回 model 类型，非 database 类型）
	GetSLORouteMappingByServiceKey(ctx context.Context, clusterID, serviceKey string) (*model.SLORouteMapping, error)
	GetSLORouteMappingsByDomain(ctx context.Context, clusterID, domain string) ([]*model.SLORouteMapping, error)
	GetSLOAllDomains(ctx context.Context, clusterID string) ([]string, error)
}

// QueryAIOps AIOps 查询与 AI 增强
type QueryAIOps interface {
	GetAIOpsGraph(ctx context.Context, clusterID string) (*aiops.DependencyGraph, error)
	GetAIOpsGraphTrace(ctx context.Context, clusterID, fromKey, direction string, maxDepth int) (*aiops.TraceResult, error)
	GetAIOpsBaseline(ctx context.Context, clusterID, entityKey string) (*aiops.EntityBaseline, error)
	GetAIOpsClusterRisk(ctx context.Context, clusterID string) (*aiops.ClusterRisk, error)
	GetAIOpsEntityRisks(ctx context.Context, clusterID, sortBy string, limit int) ([]*aiops.EntityRisk, error)
	GetAIOpsEntityRisk(ctx context.Context, clusterID, entityKey string) (*aiops.EntityRiskDetail, error)
	GetAIOpsIncidents(ctx context.Context, opts aiops.IncidentQueryOpts) ([]*aiops.Incident, int, error)
	GetAIOpsIncidentDetail(ctx context.Context, incidentID string) (*aiops.IncidentDetail, error)
	GetAIOpsIncidentStats(ctx context.Context, clusterID string, since time.Time) (*aiops.IncidentStats, error)
	GetAIOpsIncidentPatterns(ctx context.Context, entityKey string, since time.Time) ([]*aiops.IncidentPattern, error)
	SummarizeIncident(ctx context.Context, incidentID string) (*enricher.SummarizeResponse, error)
	// AI 报告查询
	ListAIReports(ctx context.Context, incidentID string) ([]*database.AIReport, error)
	GetAIReport(ctx context.Context, id int64) (*database.AIReport, error)
}

// QueryOverview 集群概览、Agent 状态、事件、单资源查询
type QueryOverview interface {
	ListClusters(ctx context.Context) ([]agent.ClusterInfo, error)
	GetCluster(ctx context.Context, clusterID string) (*agent.ClusterDetail, error)
	GetAgentStatus(ctx context.Context, clusterID string) (*agent.AgentStatus, error)
	GetCommandStatus(ctx context.Context, commandID string) (*command.Status, error)
	GetOverview(ctx context.Context, clusterID string) (*cluster.ClusterOverview, error)
	GetEvents(ctx context.Context, clusterID string, opts model.EventQueryOpts) ([]cluster.Event, error)
	GetEventsByResource(ctx context.Context, clusterID, kind, namespace, name string) ([]cluster.Event, error)
	// 单资源查询 (Event Alert Enrichment)
	GetPod(ctx context.Context, clusterID, namespace, name string) (*cluster.Pod, error)
	GetNode(ctx context.Context, clusterID, name string) (*cluster.Node, error)
	GetDeployment(ctx context.Context, clusterID, namespace, name string) (*cluster.Deployment, error)
	GetDeploymentByReplicaSet(ctx context.Context, clusterID, namespace, rsName string) (*cluster.Deployment, error)
}

// QueryAdmin 管理查询（审计日志、命令历史、事件历史、通知渠道、设置、AI Provider）
type QueryAdmin interface {
	// Audit
	ListAuditLogs(ctx context.Context, opts database.AuditQueryOpts) ([]*database.AuditLog, error)
	CountAuditLogs(ctx context.Context, opts database.AuditQueryOpts) (int64, error)
	// Command History
	ListCommandHistory(ctx context.Context, opts database.CommandQueryOpts) ([]*database.CommandHistory, error)
	CountCommandHistory(ctx context.Context, opts database.CommandQueryOpts) (int64, error)
	// Event History
	ListEventHistory(ctx context.Context, clusterID string, opts database.EventQueryOpts) ([]*database.ClusterEvent, error)
	CountEventHistory(ctx context.Context, clusterID string) (int64, error)
	// Notify
	ListNotifyChannels(ctx context.Context) ([]*database.NotifyChannel, error)
	GetNotifyChannelByType(ctx context.Context, channelType string) (*database.NotifyChannel, error)
	// Settings
	GetSetting(ctx context.Context, key string) (*database.Setting, error)
	// AI Provider
	ListAIProviders(ctx context.Context) ([]*database.AIProvider, error)
	GetAIProviderByID(ctx context.Context, id int64) (*database.AIProvider, error)
	GetAISettings(ctx context.Context) (*database.AISettings, error)
	ListAIModels(ctx context.Context) ([]*database.AIProviderModel, error)
	// AI Role Budget
	ListAIRoleBudgets(ctx context.Context) ([]*database.AIRoleBudget, error)
	// AI Reports (调用历史)
	ListRecentAIReports(ctx context.Context, role string, limit, offset int) ([]*database.AIReport, int, error)
}

// OpsAdmin 管理写入操作（通知渠道、设置、AI Provider）
type OpsAdmin interface {
	CreateNotifyChannel(ctx context.Context, ch *database.NotifyChannel) error
	UpdateNotifyChannel(ctx context.Context, ch *database.NotifyChannel) error
	SetSetting(ctx context.Context, setting *database.Setting) error
	CreateAIProvider(ctx context.Context, p *database.AIProvider) error
	UpdateAIProvider(ctx context.Context, p *database.AIProvider) error
	DeleteAIProvider(ctx context.Context, id int64) error
	UpdateAISettings(ctx context.Context, cfg *database.AISettings) error
	UpdateAIProviderRoles(ctx context.Context, id int64, roles []string) error
	UpdateAIRoleBudget(ctx context.Context, budget *database.AIRoleBudget) error
}

// ================================================================
// 组合接口（向后兼容，现有代码无需修改）
// ================================================================

// Query 只读查询接口
type Query interface {
	QueryK8s
	QueryOTel
	QuerySLO
	QueryAIOps
	QueryOverview
	QueryAdmin
}

// OpsSLO SLO 写入操作
type OpsSLO interface {
	UpsertSLOTarget(ctx context.Context, req *model.UpdateSLOTargetRequest) error
}

// Ops 写入操作接口
type Ops interface {
	CreateCommand(req *model.CreateCommandRequest) (*model.CreateCommandResponse, error)
	// ExecuteCommandSync 同步执行指令（创建 + 等待 Agent 结果）
	ExecuteCommandSync(ctx context.Context, req *model.CreateCommandRequest, timeout time.Duration) (*command.Result, error)
	OpsAdmin
	OpsSLO
}

// Service 组合接口 (master.go 持有)
type Service interface {
	Query
	Ops
}
