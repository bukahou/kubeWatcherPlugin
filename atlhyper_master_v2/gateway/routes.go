// atlhyper_master_v2/gateway/routes.go
// API 路由统一注册
// 所有 API 在此集中管理，便于查看和维护
//
// 认证策略（开源项目，展示优先）：
//   - Public: 所有只读查询，无需登录
//   - Operator (2): 敏感信息查看、指令下发
//   - Admin (3): 用户管理、系统配置
//   - Viewer (1): 等同于游客，无额外权限
package gateway

import (
	"net/http"

	"AtlHyper/atlhyper_master_v2/ai"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/deployer"
	"AtlHyper/atlhyper_master_v2/gateway/handler"
	adminHandler "AtlHyper/atlhyper_master_v2/gateway/handler/admin"
	aiopsHandler "AtlHyper/atlhyper_master_v2/gateway/handler/aiops"
	k8sHandler "AtlHyper/atlhyper_master_v2/gateway/handler/k8s"
	observeHandler "AtlHyper/atlhyper_master_v2/gateway/handler/observe"
	settingsHandler "AtlHyper/atlhyper_master_v2/gateway/handler/settings"
	sloHandler "AtlHyper/atlhyper_master_v2/gateway/handler/slo"
	"AtlHyper/atlhyper_master_v2/gateway/middleware"
	"AtlHyper/atlhyper_master_v2/github"
	"AtlHyper/atlhyper_master_v2/service"
)

// Router 路由管理器
type Router struct {
	mux            *http.ServeMux
	publicMux      *http.ServeMux // 公开路由（不需要认证）
	service        service.Service
	database       *database.DB
	aiService      ai.AIService
	analyzeTrigger aiopsHandler.AnalyzeTrigger
	ghClient       github.Client
	deployer       deployer.Deployer
}

// NewRouter 创建路由管理器
func NewRouter(svc service.Service, db *database.DB, aiSvc ai.AIService, trigger aiopsHandler.AnalyzeTrigger, ghClient github.Client, dep deployer.Deployer) *Router {
	return &Router{
		mux:            http.NewServeMux(),
		publicMux:      http.NewServeMux(),
		service:        svc,
		database:       db,
		aiService:      aiSvc,
		analyzeTrigger: trigger,
		ghClient:       ghClient,
		deployer:       dep,
	}
}

// Handler 返回包装了中间件的 Handler
func (r *Router) Handler() http.Handler {
	r.registerRoutes()

	// 组合路由器：先检查公开路由，再检查需要认证的路由
	return middleware.Logging(middleware.CORS(http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// 尝试公开路由
		if h, pattern := r.publicMux.Handler(req); pattern != "" {
			h.ServeHTTP(w, req)
			return
		}
		// 需要认证的路由
		middleware.AuthRequired(r.mux).ServeHTTP(w, req)
	})))
}

// registerRoutes 注册所有路由
func (r *Router) registerRoutes() {
	// 创建 Handlers — 顶层 (package handler)
	clusterH := handler.NewClusterHandler(r.service)
	overviewH := handler.NewOverviewHandler(r.service)
	eventH := handler.NewEventHandler(r.service)
	opsH := handler.NewOpsHandler(r.service)

	// 创建 Handlers — K8s 资源 (package k8s)
	podH := k8sHandler.NewPodHandler(r.service)
	nodeH := k8sHandler.NewNodeHandler(r.service)
	deploymentH := k8sHandler.NewDeploymentHandler(r.service)
	daemonsetH := k8sHandler.NewDaemonSetHandler(r.service)
	statefulsetH := k8sHandler.NewStatefulSetHandler(r.service)
	serviceH := k8sHandler.NewServiceHandler(r.service)
	ingressH := k8sHandler.NewIngressHandler(r.service)
	configmapH := k8sHandler.NewConfigMapHandler(r.service)
	secretH := k8sHandler.NewSecretHandler(r.service)
	namespaceH := k8sHandler.NewNamespaceHandler(r.service)
	jobH := k8sHandler.NewJobHandler(r.service)
	cronjobH := k8sHandler.NewCronJobHandler(r.service)
	pvH := k8sHandler.NewPVHandler(r.service)
	pvcH := k8sHandler.NewPVCHandler(r.service)
	networkPolicyH := k8sHandler.NewNetworkPolicyHandler(r.service)
	resourceQuotaH := k8sHandler.NewResourceQuotaHandler(r.service)
	limitRangeH := k8sHandler.NewLimitRangeHandler(r.service)
	serviceAccountH := k8sHandler.NewServiceAccountHandler(r.service)

	// 创建 Handlers — 可观测性 (package observe)
	nodeMetricsH := observeHandler.NewNodeMetricsHandler(r.service, r.service)
	observeH := observeHandler.NewObserveHandler(r.service, r.service)

	// 创建 Handlers — SLO (package slo)
	sloH := sloHandler.NewSLOHandler(r.service, r.service)

	// 创建 Handlers — AIOps (package aiops)
	aiopsGraphH := aiopsHandler.NewAIOpsGraphHandler(r.service)
	aiopsBaselineH := aiopsHandler.NewAIOpsBaselineHandler(r.service)
	aiopsRiskH := aiopsHandler.NewAIOpsRiskHandler(r.service)
	aiopsIncidentH := aiopsHandler.NewAIOpsIncidentHandler(r.service)
	aiopsAIH := aiopsHandler.NewAIOpsAIHandler(r.service)
	if r.analyzeTrigger != nil {
		aiopsAIH.SetAnalyzeTrigger(r.analyzeTrigger)
	}

	// 创建 Handlers — 管理 (package admin)
	userH := adminHandler.NewUserHandler(r.database.User)
	commandH := adminHandler.NewCommandHandler(r.service)
	notifyH := adminHandler.NewNotifyHandler(r.service)
	settingsH := adminHandler.NewSettingsHandler(r.service)
	aiProviderH := adminHandler.NewAIProviderHandler(r.service)
	auditH := adminHandler.NewAuditHandler(r.service)

	// ================================================================
	// 公开路由（无需认证）
	// 开源项目：所有只读查询对外开放
	// ================================================================

	// 登录需要审计（记录成功/失败的登录尝试）
	r.publicAudited("/api/v2/user/login", "login", "user", userH.Login)

	r.public(func(register func(pattern string, h http.HandlerFunc)) {
		// 健康检查
		register("/health", healthCheck)

		// ---------- 集群概览 ----------
		register("/api/v2/overview", overviewH.Get)

		// ---------- 集群查询 ----------
		register("/api/v2/clusters", clusterH.List)
		register("/api/v2/clusters/", clusterH.Get)

		// ---------- 工作负载查询 ----------
		// Pod
		register("/api/v2/pods", podH.List)
		register("/api/v2/pods/", podH.Get)

		// Node
		register("/api/v2/nodes", nodeH.List)
		register("/api/v2/nodes/", nodeH.Get)

		// Deployment
		register("/api/v2/deployments", deploymentH.List)
		register("/api/v2/deployments/", deploymentH.Get)

		// DaemonSet
		register("/api/v2/daemonsets", daemonsetH.List)
		register("/api/v2/daemonsets/", daemonsetH.Get)

		// StatefulSet
		register("/api/v2/statefulsets", statefulsetH.List)
		register("/api/v2/statefulsets/", statefulsetH.Get)

		// ---------- 网络查询 ----------
		// Service
		register("/api/v2/services", serviceH.List)
		register("/api/v2/services/", serviceH.Get)

		// Ingress
		register("/api/v2/ingresses", ingressH.List)
		register("/api/v2/ingresses/", ingressH.Get)

		// ---------- 批处理工作负载查询 ----------
		// Job
		register("/api/v2/jobs", jobH.List)
		register("/api/v2/jobs/", jobH.Get)

		// CronJob
		register("/api/v2/cronjobs", cronjobH.List)
		register("/api/v2/cronjobs/", cronjobH.Get)

		// ---------- 存储查询 ----------
		// PersistentVolume
		register("/api/v2/pvs", pvH.List)
		register("/api/v2/pvs/", pvH.Get)

		// PersistentVolumeClaim
		register("/api/v2/pvcs", pvcH.List)
		register("/api/v2/pvcs/", pvcH.Get)

		// ---------- 策略与配额查询 ----------
		// NetworkPolicy
		register("/api/v2/network-policies", networkPolicyH.List)
		register("/api/v2/network-policies/", networkPolicyH.Get)

		// ResourceQuota
		register("/api/v2/resource-quotas", resourceQuotaH.List)
		register("/api/v2/resource-quotas/", resourceQuotaH.Get)

		// LimitRange
		register("/api/v2/limit-ranges", limitRangeH.List)
		register("/api/v2/limit-ranges/", limitRangeH.Get)

		// ServiceAccount
		register("/api/v2/service-accounts", serviceAccountH.List)
		register("/api/v2/service-accounts/", serviceAccountH.Get)

		// ---------- 配置查询（仅列表，详情需要权限） ----------
		register("/api/v2/configmaps", configmapH.List)

		// ---------- 命名空间查询 ----------
		register("/api/v2/namespaces", namespaceH.List)
		register("/api/v2/namespaces/", namespaceH.Get)

		// ---------- 事件查询 ----------
		register("/api/v2/events", eventH.List)
		register("/api/v2/events/by-resource", eventH.ListByResource)

		// ---------- 指令查询 ----------
		register("/api/v2/commands/history", commandH.ListHistory)
		register("/api/v2/commands/", commandH.GetStatus)

		// ---------- SLO 监控查询（只读） ----------
		register("/api/v2/slo/domains", sloH.Domains)      // V1: 按 service key
		register("/api/v2/slo/domains/v2", sloH.DomainsV2) // V2: 按真实域名
		register("/api/v2/slo/domains/detail", sloH.DomainDetail)
		register("/api/v2/slo/domains/history", sloH.DomainHistory)
		register("/api/v2/slo/domains/latency", sloH.LatencyDistribution)
		register("/api/v2/slo/targets", sloH.Targets)
		register("/api/v2/slo/status-history", sloH.StatusHistory)

		// ---------- 节点指标查询（只读） ----------
		register("/api/v2/node-metrics", nodeMetricsH.Route)
		register("/api/v2/node-metrics/", nodeMetricsH.Route)

		// ---------- 可观测性查询（ClickHouse 按需） ----------
		register("/api/v2/observe/metrics/summary", observeH.MetricsSummary)
		register("/api/v2/observe/metrics/nodes", observeH.MetricsNodes)
		register("/api/v2/observe/metrics/nodes/", observeH.MetricsNodeRoute)
		register("/api/v2/observe/logs/summary", observeH.LogsSummary)
		register("/api/v2/observe/logs/query", observeH.LogsQuery)
		register("/api/v2/observe/logs/histogram", observeH.LogsHistogram)
		register("/api/v2/observe/traces/services", observeH.TracesServices)
		register("/api/v2/observe/traces/services/", observeH.APMServiceSeries)
		register("/api/v2/observe/traces/stats", observeH.TracesStats)
		register("/api/v2/observe/traces/topology", observeH.TracesTopology)
		register("/api/v2/observe/traces/operations", observeH.TracesOperations)
		register("/api/v2/observe/traces", observeH.TracesList)
		register("/api/v2/observe/traces/", observeH.TracesDetail)
		register("/api/v2/observe/slo/summary", observeH.SLOSummary)
		register("/api/v2/observe/slo/ingress", observeH.SLOIngress)
		register("/api/v2/observe/slo/timeseries", observeH.SLOTimeSeries)

		// ---------- AIOps 查询（只读） ----------
		register("/api/v2/aiops/graph", aiopsGraphH.Graph)
		register("/api/v2/aiops/graph/trace", aiopsGraphH.Trace)
		register("/api/v2/aiops/baseline", aiopsBaselineH.Baseline)
		register("/api/v2/aiops/risk/cluster", aiopsRiskH.ClusterRisk)
		register("/api/v2/aiops/risk/entities", aiopsRiskH.EntityRisks)
		register("/api/v2/aiops/risk/entity", aiopsRiskH.EntityRisk)
		register("/api/v2/aiops/incidents", aiopsIncidentH.List)
		register("/api/v2/aiops/incidents/stats", aiopsIncidentH.Stats)
		register("/api/v2/aiops/incidents/patterns", aiopsIncidentH.Patterns)
		register("/api/v2/aiops/incidents/", aiopsIncidentH.Detail)
	})

	// ================================================================
	// Operator 权限（Role >= 2）
	// 敏感信息查看、操作执行
	// 所有敏感操作都需要审计（包括权限不足的失败尝试）
	// ================================================================

	// AIOps AI 增强 (Operator 权限，有 LLM API 调用成本)
	r.operator(func(register func(pattern string, h http.HandlerFunc)) {
		register("/api/v2/aiops/ai/summarize", aiopsAIH.Summarize)
		register("/api/v2/aiops/ai/recommend", aiopsAIH.Recommend)
		register("/api/v2/aiops/ai/reports", aiopsAIH.ReportsHandler)
	})

	// ConfigMap 详情、通知渠道、审计日志、AI 配置查询（不审计，只是查看）
	r.operator(func(register func(pattern string, h http.HandlerFunc)) {
		register("/api/v2/configmaps/", configmapH.Get)
		register("/api/v2/secrets", secretH.List)
		register("/api/v2/notify/channels", notifyH.ListChannels)
		register("/api/v2/audit/logs", auditH.List)
		register("/api/v2/settings/ai", settingsH.AIConfigHandler)
		register("/api/v2/ai/providers", aiProviderH.ProvidersHandler)
		register("/api/v2/ai/settings", aiProviderH.SettingsHandler)
		register("/api/v2/ai/roles", aiProviderH.RolesOverviewHandler)
		register("/api/v2/ai/budgets", aiProviderH.BudgetsHandler)
		register("/api/v2/ai/reports", aiProviderH.AIReportsHandler)
	})

	// ---------- 需要审计的敏感操作 ----------
	// 指令下发
	r.operatorAudited("/api/v2/commands", "execute", "command", commandH.Create)

	// Pod 操作
	r.operatorAudited("/api/v2/ops/pods/logs", "read", "pod", opsH.PodLogs)
	r.operatorAudited("/api/v2/ops/pods/restart", "execute", "pod", opsH.PodRestart)

	// Deployment 操作
	r.operatorAudited("/api/v2/ops/deployments/scale", "execute", "deployment", opsH.DeploymentScale)
	r.operatorAudited("/api/v2/ops/deployments/restart", "execute", "deployment", opsH.DeploymentRestart)
	r.operatorAudited("/api/v2/ops/deployments/image", "execute", "deployment", opsH.DeploymentImage)

	// Node 操作
	r.operatorAudited("/api/v2/ops/nodes/cordon", "execute", "node", opsH.NodeCordon)
	r.operatorAudited("/api/v2/ops/nodes/uncordon", "execute", "node", opsH.NodeUncordon)

	// ConfigMap/Secret 数据获取（敏感数据读取需要审计）
	r.operatorAudited("/api/v2/ops/configmaps/data", "read", "configmap", opsH.ConfigMapData)
	r.operatorAudited("/api/v2/ops/secrets/data", "read", "secret", opsH.SecretData)

	// ================================================================
	// AI 对话（需要认证，Viewer+ 即可使用）
	// SSE 流式响应需要较长的 WriteTimeout（在 handler 内处理）
	// ================================================================

	if r.aiService != nil {
		aiH := aiopsHandler.NewAIHandler(r.aiService)
		r.mux.HandleFunc("/api/v2/ai/conversations", aiH.Conversations)
		r.mux.HandleFunc("/api/v2/ai/conversations/", aiH.ConversationByID)
		r.mux.HandleFunc("/api/v2/ai/chat", aiH.Chat)
	}

	// ================================================================
	// Admin 权限（Role >= 3）
	// 用户管理、系统配置
	// 所有管理操作都需要审计
	// ================================================================

	// ================================================================
	// GitHub 集成路由（需要 Admin 权限）
	// ================================================================
	if r.ghClient != nil {
		githubH := settingsHandler.NewGitHubHandler(r.ghClient, r.database.GitHubInstall, r.database.RepoConfig)
		deployH := adminHandler.NewDeployHandler(r.ghClient, r.database.DeployConfig, r.database.DeployHistory, r.database.GitHubInstall)
		if r.deployer != nil {
			deployH.SetDeployer(r.deployer)
		}

		// GitHub 连接状态（公开读取）
		r.public(func(register func(pattern string, h http.HandlerFunc)) {
			register("/api/github/connection", githubH.Connection)
		})

		// GitHub OAuth 回调（公开 + 审计，code 由 GitHub 签发，一次性使用）
		r.publicAudited("/api/github/callback", "create", "github", githubH.Callback)

		// GitHub 连接管理（Admin 权限，审计）
		r.adminAudited("/api/github/connect", "create", "github", githubH.Connect)
		r.adminAudited("/api/github/disconnect", "delete", "github", githubH.Disconnect)

		// 仓库管理（Admin 权限）
		r.admin(func(register func(pattern string, h http.HandlerFunc)) {
			register("/api/github/repos", githubH.Repos)
			register("/api/github/repos/", githubH.RepoSubRoute)
		})

		// 部署配置（Admin 权限）
		r.admin(func(register func(pattern string, h http.HandlerFunc)) {
			register("/api/deploy/config", deployH.Config)
			register("/api/deploy/kustomize-paths", deployH.KustomizePaths)
			register("/api/deploy/test-connection", deployH.TestConnection)
			register("/api/deploy/status", deployH.Status)
			register("/api/deploy/sync", deployH.SyncNow)
			register("/api/deploy/rollback", deployH.Rollback)
		})

		// 部署历史（公开读取）
		r.public(func(register func(pattern string, h http.HandlerFunc)) {
			register("/api/deploy/history", deployH.History)
		})
	}

	// 用户列表查询（不审计，只是查看）
	r.admin(func(register func(pattern string, h http.HandlerFunc)) {
		register("/api/v2/user/list", userH.List)
	})

	// ---------- 需要审计的管理操作 ----------
	// 用户管理
	r.adminAudited("/api/v2/user/register", "create", "user", userH.Register)
	r.adminAudited("/api/v2/user/update-role", "update", "user", userH.UpdateRole)
	r.adminAudited("/api/v2/user/update-status", "update", "user", userH.UpdateStatus)
	r.adminAudited("/api/v2/user/delete", "delete", "user", userH.Delete)

	// 通知渠道管理（Operator 可管理）
	r.operatorAudited("/api/v2/notify/channels/", "update", "notify", notifyH.ChannelHandler)

	// AI 配置管理（需要 Admin 权限）
	r.adminAudited("/api/v2/settings/ai/", "update", "ai_config", settingsH.AIConfigHandler)

	// AI Provider 管理（需要 Admin 权限）
	r.adminAudited("/api/v2/ai/providers/", "update", "ai_provider", aiProviderH.ProviderHandler)
	r.adminAudited("/api/v2/ai/settings/", "update", "ai_settings", aiProviderH.SettingsHandler)
	r.adminAudited("/api/v2/ai/budgets/", "update", "ai_budget", aiProviderH.BudgetHandler)

	// AI 报告详情 + 深度分析触发（Operator 权限，审计）
	r.operatorAudited("/api/v2/aiops/ai/reports/", "read", "ai_report", aiopsAIH.ReportDetailHandler)
	r.operatorAudited("/api/v2/aiops/ai/analyze", "execute", "ai_analysis", aiopsAIH.AnalyzeHandler)
}

// ================================================================
// 路由注册辅助函数
// ================================================================

// public 注册公开路由（无需认证）
func (r *Router) public(fn func(register func(pattern string, h http.HandlerFunc))) {
	fn(func(pattern string, h http.HandlerFunc) {
		r.publicMux.HandleFunc(pattern, h)
	})
}

// operator 注册 Operator 权限路由
func (r *Router) operator(fn func(register func(pattern string, h http.HandlerFunc))) {
	fn(func(pattern string, h http.HandlerFunc) {
		r.mux.HandleFunc(pattern, middleware.RequireMinRole(middleware.RoleOperator, h))
	})
}

// admin 注册 Admin 权限路由
func (r *Router) admin(fn func(register func(pattern string, h http.HandlerFunc))) {
	fn(func(pattern string, h http.HandlerFunc) {
		r.mux.HandleFunc(pattern, middleware.RequireMinRole(middleware.RoleAdmin, h))
	})
}

// ================================================================
// 审计路由辅助函数
// 审计中间件包装在权限检查之外，无论成功失败都会记录
// ================================================================

// audit 创建审计中间件
func (r *Router) audit(action, resource string) func(http.HandlerFunc) http.HandlerFunc {
	return middleware.Audit(r.database.Audit, middleware.AuditConfig{
		Action:   action,
		Resource: resource,
	})
}

// operatorAudited 注册带审计的 Operator 权限路由
// 顺序: Audit -> RequireMinRole(Operator) -> Handler
// 无论认证成功/失败，都会记录审计日志
func (r *Router) operatorAudited(pattern, action, resource string, h http.HandlerFunc) {
	wrapped := r.audit(action, resource)(middleware.RequireMinRole(middleware.RoleOperator, h))
	r.mux.HandleFunc(pattern, wrapped)
}

// adminAudited 注册带审计的 Admin 权限路由
// 顺序: Audit -> RequireMinRole(Admin) -> Handler
func (r *Router) adminAudited(pattern, action, resource string, h http.HandlerFunc) {
	wrapped := r.audit(action, resource)(middleware.RequireMinRole(middleware.RoleAdmin, h))
	r.mux.HandleFunc(pattern, wrapped)
}

// publicAudited 注册带审计的公开路由（如登录）
func (r *Router) publicAudited(pattern, action, resource string, h http.HandlerFunc) {
	wrapped := r.audit(action, resource)(h)
	r.publicMux.HandleFunc(pattern, wrapped)
}

// healthCheck 健康检查
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok"))
}
