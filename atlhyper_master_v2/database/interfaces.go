// atlhyper_master_v2/database/interfaces.go
// Database 统一接口定义
// 包含: DB 结构体 + Repository 接口 + Dialect 接口
package database

import (
	"context"
	"database/sql"
	"time"
)

// ==================== DB 结构体 ====================

// DB 数据库统一访问点
// 通过 New() 工厂函数创建，repo.Init() 注入 Repository 实例
type DB struct {
	Audit          AuditRepository
	User           UserRepository
	Event          ClusterEventRepository
	Notify         NotifyChannelRepository
	Cluster        ClusterRepository
	Command        CommandHistoryRepository
	Settings       SettingsRepository
	AIConversation AIConversationRepository
	AIMessage      AIMessageRepository
	AIProvider     AIProviderRepository
	AISettings     AISettingsRepository
	AIModel        AIProviderModelRepository
	SLO            SLORepository

	AIOpsBaseline AIOpsBaselineRepository
	AIOpsGraph    AIOpsGraphRepository
	AIOpsIncident AIOpsIncidentRepository

	AIRoleBudget AIRoleBudgetRepository
	AIReport     AIReportRepository

	GitHubInstall GitHubInstallationRepository
	RepoConfig    RepoConfigRepository
	DeployConfig  DeployConfigRepository
	DeployHistory DeployHistoryRepository

	Conn *sql.DB // 导出供 repo 包使用
}

// Close 关闭数据库连接
func (db *DB) Close() error {
	return db.Conn.Close()
}

// ==================== Repository 接口 ====================

// AuditRepository 审计日志接口
type AuditRepository interface {
	Create(ctx context.Context, log *AuditLog) error
	List(ctx context.Context, opts AuditQueryOpts) ([]*AuditLog, error)
	Count(ctx context.Context, opts AuditQueryOpts) (int64, error)
}

// UserRepository 用户接口
type UserRepository interface {
	Create(ctx context.Context, user *User) error
	Update(ctx context.Context, user *User) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*User, error)
	GetByUsername(ctx context.Context, username string) (*User, error)
	List(ctx context.Context) ([]*User, error)
	UpdateLastLogin(ctx context.Context, id int64, ip string) error
}

// ClusterEventRepository Event 持久化接口
type ClusterEventRepository interface {
	// 写入
	Upsert(ctx context.Context, event *ClusterEvent) error
	UpsertBatch(ctx context.Context, events []*ClusterEvent) error

	// 查询
	ListByCluster(ctx context.Context, clusterID string, opts EventQueryOpts) ([]*ClusterEvent, error)
	ListByInvolvedResource(ctx context.Context, clusterID, kind, namespace, name string) ([]*ClusterEvent, error)
	ListByType(ctx context.Context, clusterID, eventType string, since time.Time) ([]*ClusterEvent, error)

	// 告警服务专用
	GetLatestEventID(ctx context.Context) (int64, error)
	GetEventsSince(ctx context.Context, sinceID int64) ([]*ClusterEvent, error)

	// 清理
	DeleteBefore(ctx context.Context, clusterID string, before time.Time) (int64, error)
	DeleteOldest(ctx context.Context, clusterID string, keepCount int) (int64, error)

	// 统计
	CountByCluster(ctx context.Context, clusterID string) (int64, error)
	CountByHour(ctx context.Context, clusterID string, hours int) ([]HourlyEventCount, error)
	CountByHourAndKind(ctx context.Context, clusterID string, hours int) ([]HourlyKindCount, error)
}

// NotifyChannelRepository 通知渠道接口
type NotifyChannelRepository interface {
	Create(ctx context.Context, channel *NotifyChannel) error
	Update(ctx context.Context, channel *NotifyChannel) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*NotifyChannel, error)
	GetByType(ctx context.Context, channelType string) (*NotifyChannel, error)
	List(ctx context.Context) ([]*NotifyChannel, error)
	ListEnabled(ctx context.Context) ([]*NotifyChannel, error)
}

// ClusterRepository 集群接口
type ClusterRepository interface {
	Create(ctx context.Context, cluster *Cluster) error
	Update(ctx context.Context, cluster *Cluster) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*Cluster, error)
	GetByUID(ctx context.Context, uid string) (*Cluster, error)
	List(ctx context.Context) ([]*Cluster, error)
}

// CommandHistoryRepository 指令历史接口
type CommandHistoryRepository interface {
	Create(ctx context.Context, cmd *CommandHistory) error
	Update(ctx context.Context, cmd *CommandHistory) error
	GetByCommandID(ctx context.Context, cmdID string) (*CommandHistory, error)
	ListByCluster(ctx context.Context, clusterID string, limit, offset int) ([]*CommandHistory, error)
	ListByUser(ctx context.Context, userID int64, limit, offset int) ([]*CommandHistory, error)
	// 新增：带筛选条件的列表查询
	List(ctx context.Context, opts CommandQueryOpts) ([]*CommandHistory, error)
	Count(ctx context.Context, opts CommandQueryOpts) (int64, error)
}

// SettingsRepository 设置接口
type SettingsRepository interface {
	Get(ctx context.Context, key string) (*Setting, error)
	GetByPrefix(ctx context.Context, prefix string) ([]*Setting, error)
	Set(ctx context.Context, setting *Setting) error
	Delete(ctx context.Context, key string) error
	List(ctx context.Context) ([]*Setting, error)
}

// AIConversationRepository AI 对话接口
type AIConversationRepository interface {
	Create(ctx context.Context, conv *AIConversation) error
	Update(ctx context.Context, conv *AIConversation) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*AIConversation, error)
	ListByUser(ctx context.Context, userID int64, limit, offset int) ([]*AIConversation, error)
}

// AIMessageRepository AI 消息接口
type AIMessageRepository interface {
	Create(ctx context.Context, msg *AIMessage) error
	ListByConversation(ctx context.Context, convID int64) ([]*AIMessage, error)
	DeleteByConversation(ctx context.Context, convID int64) error
}

// AIProviderRepository AI 提供商配置接口
type AIProviderRepository interface {
	Create(ctx context.Context, p *AIProvider) error
	Update(ctx context.Context, p *AIProvider) error
	Delete(ctx context.Context, id int64) error // 软删除
	GetByID(ctx context.Context, id int64) (*AIProvider, error)
	List(ctx context.Context) ([]*AIProvider, error) // deleted_at IS NULL

	// 角色路由
	UpdateRoles(ctx context.Context, id int64, roles []string) error
	FindByRole(ctx context.Context, role string) (*AIProvider, error)

	// 统计更新
	IncrementUsage(ctx context.Context, id int64, requests, tokens int64, cost float64) error
	UpdateStatus(ctx context.Context, id int64, status, errorMsg string) error
}

// AISettingsRepository AI 全局设置接口
type AISettingsRepository interface {
	Get(ctx context.Context) (*AISettings, error)
	Update(ctx context.Context, cfg *AISettings) error
}

// AIProviderModelRepository 提供商模型管理接口
type AIProviderModelRepository interface {
	Create(ctx context.Context, m *AIProviderModel) error
	Delete(ctx context.Context, id int64) error
	GetByID(ctx context.Context, id int64) (*AIProviderModel, error)
	ListByProvider(ctx context.Context, provider string) ([]*AIProviderModel, error)
	ListAll(ctx context.Context) ([]*AIProviderModel, error)
	GetDefaultModel(ctx context.Context, provider string) (*AIProviderModel, error)
}

// SLORepository SLO 配置与路由映射访问接口
// 时序数据（raw/hourly）已迁移至 OTelSnapshot + ClickHouse
type SLORepository interface {
	// Targets
	GetTargets(ctx context.Context, clusterID string) ([]*SLOTarget, error)
	GetTargetsByHost(ctx context.Context, clusterID, host string) ([]*SLOTarget, error)
	UpsertTarget(ctx context.Context, t *SLOTarget) error
	DeleteTarget(ctx context.Context, clusterID, host, timeRange string) error

	// Route Mapping
	UpsertRouteMapping(ctx context.Context, m *SLORouteMapping) error
	GetRouteMappingByServiceKey(ctx context.Context, clusterID, serviceKey string) (*SLORouteMapping, error)
	GetRouteMappingsByDomain(ctx context.Context, clusterID, domain string) ([]*SLORouteMapping, error)
	GetAllRouteMappings(ctx context.Context, clusterID string) ([]*SLORouteMapping, error)
	GetAllDomains(ctx context.Context, clusterID string) ([]string, error)
	DeleteRouteMapping(ctx context.Context, clusterID, serviceKey string) error
}

// ==================== AI Role Budget Repository 接口 ====================

// AIRoleBudgetRepository 角色预算接口
type AIRoleBudgetRepository interface {
	Get(ctx context.Context, role string) (*AIRoleBudget, error)
	ListAll(ctx context.Context) ([]*AIRoleBudget, error)
	Upsert(ctx context.Context, budget *AIRoleBudget) error
	Delete(ctx context.Context, role string) error
	IncrementUsage(ctx context.Context, role string, inputTokens, outputTokens int) error
	ResetDailyUsage(ctx context.Context, role string) error
	ResetMonthlyUsage(ctx context.Context, role string) error
}

// ==================== AI Report Repository 接口 ====================

// AIReportRepository AI 分析报告接口
type AIReportRepository interface {
	Create(ctx context.Context, report *AIReport) error
	GetByID(ctx context.Context, id int64) (*AIReport, error)
	ListByIncident(ctx context.Context, incidentID string) ([]*AIReport, error)
	ListByCluster(ctx context.Context, clusterID, role string, limit int) ([]*AIReport, error)
	ListRecent(ctx context.Context, role string, limit, offset int) ([]*AIReport, int, error)
	CountByClusterAndRole(ctx context.Context, clusterID, role string, since time.Time) (int, error)
	DeleteBefore(ctx context.Context, before time.Time) (int64, error)
	UpdateInvestigationSteps(ctx context.Context, id int64, steps string) error
	UpdateResult(ctx context.Context, id int64, report *AIReport) error
}

// ==================== AIOps Repository 接口 ====================

// AIOpsBaselineRepository 基线状态数据访问接口
type AIOpsBaselineRepository interface {
	BatchUpsert(ctx context.Context, states []*AIOpsBaselineState) error
	ListAll(ctx context.Context) ([]*AIOpsBaselineState, error)
	ListByEntity(ctx context.Context, entityKey string) ([]*AIOpsBaselineState, error)
	DeleteByEntity(ctx context.Context, entityKey string) error
}

// AIOpsGraphRepository 依赖图快照数据访问接口
type AIOpsGraphRepository interface {
	Save(ctx context.Context, clusterID string, snapshot []byte) error
	Load(ctx context.Context, clusterID string) ([]byte, error)
	ListClusterIDs(ctx context.Context) ([]string, error)
}

// ==================== AIOps Incident Repository 接口 ====================

// AIOpsIncidentRepository 事件数据访问接口
type AIOpsIncidentRepository interface {
	CreateIncident(ctx context.Context, inc *AIOpsIncident) error
	GetByID(ctx context.Context, id string) (*AIOpsIncident, error)
	UpdateState(ctx context.Context, id, state, severity string) error
	Resolve(ctx context.Context, id string, resolvedAt time.Time) error
	UpdateRootCause(ctx context.Context, id, rootCause string) error
	UpdatePeakRisk(ctx context.Context, id string, peakRisk float64) error
	IncrementRecurrence(ctx context.Context, id string) error
	List(ctx context.Context, opts AIOpsIncidentQueryOpts) ([]*AIOpsIncident, error)
	Count(ctx context.Context, opts AIOpsIncidentQueryOpts) (int, error)
	AddEntity(ctx context.Context, entity *AIOpsIncidentEntity) error
	GetEntities(ctx context.Context, incidentID string) ([]*AIOpsIncidentEntity, error)
	AddTimeline(ctx context.Context, entry *AIOpsIncidentTimeline) error
	GetTimeline(ctx context.Context, incidentID string) ([]*AIOpsIncidentTimeline, error)
	GetIncidentStats(ctx context.Context, clusterID string, since time.Time) (*AIOpsIncidentStatsRaw, error)
	TopRootCauses(ctx context.Context, clusterID string, since time.Time, limit int) ([]AIOpsRootCauseCount, error)
	ListByEntity(ctx context.Context, entityKey string, since time.Time) ([]*AIOpsIncident, error)
}

// ==================== GitHub Integration Repository 接口 ====================

// GitHubInstallationRepository GitHub App 安装记录接口
type GitHubInstallationRepository interface {
	Upsert(ctx context.Context, inst *GitHubInstallation) error
	Get(ctx context.Context) (*GitHubInstallation, error)
	Delete(ctx context.Context) error
}

// RepoConfigRepository 仓库映射配置接口
type RepoConfigRepository interface {
	Upsert(ctx context.Context, config *RepoConfig) error
	GetByRepo(ctx context.Context, repo string) (*RepoConfig, error)
	List(ctx context.Context) ([]*RepoConfig, error)
	Delete(ctx context.Context, repo string) error
}

// DeployConfigRepository 部署配置接口
type DeployConfigRepository interface {
	Upsert(ctx context.Context, config *DeployConfig) error
	GetByCluster(ctx context.Context, clusterID string) (*DeployConfig, error)
	List(ctx context.Context) ([]*DeployConfig, error)
	Delete(ctx context.Context, clusterID string) error
}

// DeployHistoryRepository 部署历史接口
type DeployHistoryRepository interface {
	Create(ctx context.Context, record *DeployHistory) error
	GetByID(ctx context.Context, id int64) (*DeployHistory, error)
	List(ctx context.Context, opts DeployHistoryQueryOpts) ([]*DeployHistory, error)
	Count(ctx context.Context, opts DeployHistoryQueryOpts) (int, error)
	GetLatestByPath(ctx context.Context, clusterID, path string) (*DeployHistory, error)
}

// ==================== Dialect 接口 ====================

// Dialect 数据库方言接口
// 每种数据库引擎 (SQLite, MySQL, PostgreSQL) 提供自己的实现
// Dialect 负责 SQL 生成和行扫描（各 DB 的类型映射不同）
type Dialect interface {
	Audit() AuditDialect
	User() UserDialect
	Event() EventDialect
	Notify() NotifyDialect
	Cluster() ClusterDialect
	Command() CommandDialect
	Settings() SettingsDialect
	AIConversation() AIConversationDialect
	AIMessage() AIMessageDialect
	AIProvider() AIProviderDialect
	AISettings() AISettingsDialect
	AIProviderModel() AIProviderModelDialect
	SLO() SLODialect
	AIRoleBudget() AIRoleBudgetDialect
	AIReport() AIReportDialect

	AIOpsBaseline() AIOpsBaselineDialect
	AIOpsGraph() AIOpsGraphDialect
	AIOpsIncident() AIOpsIncidentDialect
	GitHubInstall() GitHubInstallDialect
	RepoConfig() RepoConfigDialect
	DeployConfig() DeployConfigDialect
	DeployHistory() DeployHistoryDialect
	Migrate(db *sql.DB) error
}

// AuditDialect 审计日志 SQL 方言
type AuditDialect interface {
	Insert(log *AuditLog) (query string, args []any)
	List(opts AuditQueryOpts) (query string, args []any)
	Count(opts AuditQueryOpts) (query string, args []any)
	ScanRow(rows *sql.Rows) (*AuditLog, error)
}

// UserDialect 用户 SQL 方言
type UserDialect interface {
	Insert(user *User) (query string, args []any)
	Update(user *User) (query string, args []any)
	Delete(id int64) (query string, args []any)
	SelectByID(id int64) (query string, args []any)
	SelectByUsername(username string) (query string, args []any)
	SelectAll() (query string, args []any)
	UpdateLastLogin(id int64, ip string) (query string, args []any)
	ScanRow(rows *sql.Rows) (*User, error)
}

// EventDialect 事件 SQL 方言
type EventDialect interface {
	Upsert(event *ClusterEvent) (query string, args []any)
	ListByCluster(clusterID string, opts EventQueryOpts) (query string, args []any)
	ListByInvolvedResource(clusterID, kind, namespace, name string) (query string, args []any)
	ListByType(clusterID, eventType string, since time.Time) (query string, args []any)
	DeleteBefore(clusterID string, before time.Time) (query string, args []any)
	CountByCluster(clusterID string) (query string, args []any)
	CountByHour(clusterID string, since time.Time) (query string, args []any)
	CountByHourAndKind(clusterID string, since time.Time) (query string, args []any)
	ScanRow(rows *sql.Rows) (*ClusterEvent, error)
	ScanHourlyCount(rows *sql.Rows) (*HourlyEventCount, error)
	ScanHourlyKindCount(rows *sql.Rows) (*HourlyKindCount, error)
}

// NotifyDialect 通知渠道 SQL 方言
type NotifyDialect interface {
	Insert(ch *NotifyChannel) (query string, args []any)
	Update(ch *NotifyChannel) (query string, args []any)
	Delete(id int64) (query string, args []any)
	SelectByID(id int64) (query string, args []any)
	SelectByType(channelType string) (query string, args []any)
	SelectAll() (query string, args []any)
	SelectEnabled() (query string, args []any)
	ScanRow(rows *sql.Rows) (*NotifyChannel, error)
}

// ClusterDialect 集群 SQL 方言
type ClusterDialect interface {
	Insert(cluster *Cluster) (query string, args []any)
	Update(cluster *Cluster) (query string, args []any)
	Delete(id int64) (query string, args []any)
	SelectByID(id int64) (query string, args []any)
	SelectByUID(uid string) (query string, args []any)
	SelectAll() (query string, args []any)
	ScanRow(rows *sql.Rows) (*Cluster, error)
}

// CommandDialect 指令历史 SQL 方言
type CommandDialect interface {
	Insert(cmd *CommandHistory) (query string, args []any)
	Update(cmd *CommandHistory) (query string, args []any)
	SelectByCommandID(cmdID string) (query string, args []any)
	SelectByCluster(clusterID string, limit, offset int) (query string, args []any)
	SelectByUser(userID int64, limit, offset int) (query string, args []any)
	// 新增：带筛选条件的查询
	SelectWithOpts(opts CommandQueryOpts) (query string, args []any)
	CountWithOpts(opts CommandQueryOpts) (query string, args []any)
	ScanRow(rows *sql.Rows) (*CommandHistory, error)
}

// SettingsDialect 设置 SQL 方言
type SettingsDialect interface {
	SelectByKey(key string) (query string, args []any)
	SelectByPrefix(prefix string) (query string, args []any)
	Upsert(s *Setting) (query string, args []any)
	Delete(key string) (query string, args []any)
	SelectAll() (query string, args []any)
	ScanRow(rows *sql.Rows) (*Setting, error)
}

// AIConversationDialect AI 对话 SQL 方言
type AIConversationDialect interface {
	Insert(conv *AIConversation) (query string, args []any)
	Update(conv *AIConversation) (query string, args []any)
	Delete(id int64) (query string, args []any)
	SelectByID(id int64) (query string, args []any)
	SelectByUser(userID int64, limit, offset int) (query string, args []any)
	ScanRow(rows *sql.Rows) (*AIConversation, error)
}

// AIMessageDialect AI 消息 SQL 方言
type AIMessageDialect interface {
	Insert(msg *AIMessage) (query string, args []any)
	SelectByConversation(convID int64) (query string, args []any)
	DeleteByConversation(convID int64) (query string, args []any)
	ScanRow(rows *sql.Rows) (*AIMessage, error)
}

// AIProviderDialect AI 提供商 SQL 方言
type AIProviderDialect interface {
	Insert(p *AIProvider) (query string, args []any)
	Update(p *AIProvider) (query string, args []any)
	Delete(id int64) (query string, args []any)
	SelectByID(id int64) (query string, args []any)
	SelectAll() (query string, args []any)
	UpdateRoles(id int64, rolesJSON string) (query string, args []any)
	IncrementUsage(id int64, requests, tokens int64, cost float64) (query string, args []any)
	UpdateStatus(id int64, status, errorMsg string) (query string, args []any)
	ScanRow(rows *sql.Rows) (*AIProvider, error)
}

// AISettingsDialect AI 全局设置 SQL 方言
type AISettingsDialect interface {
	Select() (query string, args []any)
	Update(cfg *AISettings) (query string, args []any)
	ScanRow(rows *sql.Rows) (*AISettings, error)
}

// AIProviderModelDialect 提供商模型 SQL 方言
type AIProviderModelDialect interface {
	Insert(m *AIProviderModel) (query string, args []any)
	Delete(id int64) (query string, args []any)
	SelectByID(id int64) (query string, args []any)
	SelectByProvider(provider string) (query string, args []any)
	SelectAll() (query string, args []any)
	SelectDefault(provider string) (query string, args []any)
	ScanRow(rows *sql.Rows) (*AIProviderModel, error)
}

// AIRoleBudgetDialect 角色预算 SQL 方言
type AIRoleBudgetDialect interface {
	Upsert(b *AIRoleBudget) (query string, args []any)
	SelectByRole(role string) (query string, args []any)
	SelectAll() (query string, args []any)
	Delete(role string) (query string, args []any)
	IncrementUsage(role string, inputTokens, outputTokens int) (query string, args []any)
	ResetDailyUsage(role string) (query string, args []any)
	ResetMonthlyUsage(role string) (query string, args []any)
	ScanRow(rows *sql.Rows) (*AIRoleBudget, error)
}

// AIReportDialect AI 分析报告 SQL 方言
type AIReportDialect interface {
	Insert(r *AIReport) (query string, args []any)
	SelectByID(id int64) (query string, args []any)
	SelectByIncident(incidentID string) (query string, args []any)
	SelectByCluster(clusterID, role string, limit int) (query string, args []any)
	SelectRecent(role string, limit, offset int) (query string, args []any)
	CountRecent(role string) (query string, args []any)
	CountByClusterAndRole(clusterID, role string, since time.Time) (query string, args []any)
	DeleteBefore(before time.Time) (query string, args []any)
	UpdateInvestigationSteps(id int64, steps string) (query string, args []any)
	UpdateResult(id int64, r *AIReport) (query string, args []any)
	ScanRow(rows *sql.Rows) (*AIReport, error)
}

// SLODialect SLO 配置与路由映射 SQL 方言
// 时序数据（raw/hourly）已迁移至 OTelSnapshot + ClickHouse
type SLODialect interface {
	// Targets
	SelectTargets(clusterID string) (query string, args []any)
	SelectTargetsByHost(clusterID, host string) (query string, args []any)
	UpsertTarget(t *SLOTarget) (query string, args []any)
	DeleteTarget(clusterID, host, timeRange string) (query string, args []any)
	ScanTarget(rows *sql.Rows) (*SLOTarget, error)

	// Route Mapping
	UpsertRouteMapping(m *SLORouteMapping) (query string, args []any)
	SelectRouteMappingByServiceKey(clusterID, serviceKey string) (query string, args []any)
	SelectRouteMappingsByDomain(clusterID, domain string) (query string, args []any)
	SelectAllRouteMappings(clusterID string) (query string, args []any)
	SelectAllDomains(clusterID string) (query string, args []any)
	DeleteRouteMapping(clusterID, serviceKey string) (query string, args []any)
	ScanRouteMapping(rows *sql.Rows) (*SLORouteMapping, error)
}

// ==================== AIOps Dialect 接口 ====================

// AIOpsBaselineDialect 基线 SQL 方言
type AIOpsBaselineDialect interface {
	Upsert(state *AIOpsBaselineState) (query string, args []any)
	SelectAll() (query string, args []any)
	SelectByEntity(entityKey string) (query string, args []any)
	DeleteByEntity(entityKey string) (query string, args []any)
	ScanRow(rows *sql.Rows) (*AIOpsBaselineState, error)
}

// AIOpsGraphDialect 依赖图 SQL 方言
type AIOpsGraphDialect interface {
	Upsert(clusterID string, snapshot []byte) (query string, args []any)
	SelectByCluster(clusterID string) (query string, args []any)
	SelectAllClusterIDs() (query string, args []any)
	ScanSnapshot(rows *sql.Rows) (clusterID string, data []byte, err error)
}

// ==================== AIOps Incident Dialect 接口 ====================

// AIOpsIncidentDialect 事件 SQL 方言
type AIOpsIncidentDialect interface {
	InsertIncident(inc *AIOpsIncident) (string, []any)
	SelectByID(id string) (string, []any)
	UpdateState(id, state, severity string) (string, []any)
	Resolve(id string, resolvedAt time.Time) (string, []any)
	UpdateRootCause(id, rootCause string) (string, []any)
	UpdatePeakRisk(id string, peakRisk float64) (string, []any)
	IncrementRecurrence(id string) (string, []any)
	InsertEntity(entity *AIOpsIncidentEntity) (string, []any)
	SelectEntities(incidentID string) (string, []any)
	InsertTimeline(entry *AIOpsIncidentTimeline) (string, []any)
	SelectTimeline(incidentID string) (string, []any)
	ScanIncident(rows *sql.Rows) (*AIOpsIncident, error)
	ScanEntity(rows *sql.Rows) (*AIOpsIncidentEntity, error)
	ScanTimeline(rows *sql.Rows) (*AIOpsIncidentTimeline, error)
}

// ==================== GitHub Integration Dialect 接口 ====================

// GitHubInstallDialect GitHub 安装 SQL 方言
type GitHubInstallDialect interface {
	Upsert(inst *GitHubInstallation) (query string, args []any)
	Select() (query string, args []any)
	Delete() (query string, args []any)
	ScanRow(rows *sql.Rows) (*GitHubInstallation, error)
}

// RepoConfigDialect 仓库配置 SQL 方言
type RepoConfigDialect interface {
	Upsert(config *RepoConfig) (query string, args []any)
	SelectByRepo(repo string) (query string, args []any)
	SelectAll() (query string, args []any)
	Delete(repo string) (query string, args []any)
	ScanRow(rows *sql.Rows) (*RepoConfig, error)
}

// DeployConfigDialect 部署配置 SQL 方言
type DeployConfigDialect interface {
	Upsert(config *DeployConfig) (query string, args []any)
	SelectByCluster(clusterID string) (query string, args []any)
	SelectAll() (query string, args []any)
	Delete(clusterID string) (query string, args []any)
	ScanRow(rows *sql.Rows) (*DeployConfig, error)
}

// DeployHistoryDialect 部署历史 SQL 方言
type DeployHistoryDialect interface {
	Insert(record *DeployHistory) (query string, args []any)
	SelectByID(id int64) (query string, args []any)
	SelectWithOpts(opts DeployHistoryQueryOpts) (query string, args []any)
	CountWithOpts(opts DeployHistoryQueryOpts) (query string, args []any)
	SelectLatestByPath(clusterID, path string) (query string, args []any)
	ScanRow(rows *sql.Rows) (*DeployHistory, error)
}
