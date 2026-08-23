// Package atlhyper_agent_v2 Agent V2 核心包
//
// 本包提供 Agent 的启动器，负责:
//   - 初始化所有依赖 (SDK, Gateway, Repository, Service, Scheduler)
//   - 依赖注入和组装
//   - 生命周期管理 (启动、运行、停止)
//
// 使用方式:
//
//	config.LoadConfig()
//	agent, err := atlhyper_agent_v2.New()
//	agent.Run(ctx)
package atlhyper_agent_v2

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"AtlHyper/atlhyper_agent_v2/concentrator"
	"AtlHyper/atlhyper_agent_v2/config"
	"AtlHyper/atlhyper_agent_v2/gateway"
	"AtlHyper/atlhyper_agent_v2/repository"
	chrepo "AtlHyper/atlhyper_agent_v2/repository/ch"
	chquery "AtlHyper/atlhyper_agent_v2/repository/ch/query"
	k8srepo "AtlHyper/atlhyper_agent_v2/repository/k8s"
	"AtlHyper/atlhyper_agent_v2/scheduler"
	sdkpkg "AtlHyper/atlhyper_agent_v2/sdk"
	chpkg "AtlHyper/atlhyper_agent_v2/sdk/impl/clickhouse"
	k8spkg "AtlHyper/atlhyper_agent_v2/sdk/impl/k8s"
	commandsvc "AtlHyper/atlhyper_agent_v2/service/command"
	snapshotsvc "AtlHyper/atlhyper_agent_v2/service/snapshot"
	"AtlHyper/common/logger"
)

var log = logger.Module("Agent")

// Agent 是 Agent V2 的主结构体
// 封装调度器，提供启动/运行/停止接口
type Agent struct {
	scheduler *scheduler.Scheduler
	chClient  sdkpkg.ClickHouseClient // 可选，nil 时不启动
}

// New 创建并初始化 Agent 实例
//
// 使用 config.GlobalConfig 中的配置初始化各层组件。
// 调用前必须先调用 config.LoadConfig()。
//
// 初始化顺序:
//  1. SDK 层 - 连接 K8s API Server
//  2. Gateway 层 - 创建 Master 通信客户端
//  3. Repository 层 - 创建各资源仓库
//  4. Service 层 - 创建业务服务
//  5. Scheduler 层 - 创建调度器
//
// 返回:
//   - *Agent: Agent 实例
//   - error: 初始化错误 (通常是 K8s 连接失败)
func New() (*Agent, error) {
	cfg := &config.GlobalConfig

	// 1. 初始化 SDK (连接 K8s)
	k8sClient, err := k8spkg.NewClient(cfg.Kubernetes.KubeConfig)
	if err != nil {
		return nil, err
	}
	log.Info("K8s 客户端初始化完成")

	// 如果未配置 ClusterID，自动获取集群 UID (kube-system namespace 的 UID)
	if cfg.Agent.ClusterID == "" {
		ns, err := k8sClient.GetNamespace(context.Background(), "kube-system")
		if err != nil {
			return nil, fmt.Errorf("failed to get cluster UID: %w", err)
		}
		cfg.Agent.ClusterID = string(ns.UID)
		log.Info("自动检测 ClusterID", "cluster", cfg.Agent.ClusterID)
	}

	// 2. 初始化 Gateway (Master 通信)
	masterGw := gateway.NewMasterGateway(cfg.Master.URL, cfg.Agent.ClusterID, cfg.Timeout.HTTPClient)

	// 3. 初始化 Repository (数据访问层)
	repos := initRepositories(k8sClient)

	// 3.1 初始化 ClickHouse 客户端（可选）
	var otelSummaryRepo repository.OTelSummaryRepository
	var traceQueryRepo repository.TraceQueryRepository
	var logQueryRepo repository.LogQueryRepository
	var metricsQueryRepo repository.MetricsQueryRepository
	var sloQueryRepo repository.SLOQueryRepository
	var dashboardRepo repository.OTelDashboardRepository
	var chClient sdkpkg.ClickHouseClient
	if cfg.ClickHouse.Endpoint != "" {
		chClient, err = chpkg.NewClient(cfg.ClickHouse.Endpoint, cfg.ClickHouse.Database, cfg.ClickHouse.Timeout)
		if err != nil {
			log.Warn("ClickHouse 客户端初始化失败，OTel 不可用", "err", err)
		} else {
			otelSummaryRepo = chrepo.NewOTelSummaryRepository(chClient)
			traceQueryRepo = chquery.NewTraceQueryRepository(chClient)
			logQueryRepo = chquery.NewLogQueryRepository(chClient)
			metricsQueryRepo = chquery.NewMetricsQueryRepository(chClient, repos.node)
			sloQueryRepo = chquery.NewSLOQueryRepository(chClient)
			dashboardRepo = chrepo.NewDashboardRepository(metricsQueryRepo, traceQueryRepo, sloQueryRepo, logQueryRepo)
			log.Info("ClickHouse 客户端初始化完成", "endpoint", cfg.ClickHouse.Endpoint)
		}
	}

	// 3.2 初始化 Concentrator（预聚合时序）
	var conc concentrator.TimeSeriesAggregator
	if cfg.ClickHouse.Endpoint != "" {
		conc = concentrator.NewConcentrator()
		log.Info("Concentrator 初始化完成")
	}

	// 4. 初始化 Service (业务逻辑层)
	snapshotSvc := snapshotsvc.NewSnapshotService(
		cfg.Agent.ClusterID,
		repos.pod, repos.node, repos.deployment,
		repos.statefulSet, repos.daemonSet, repos.replicaSet,
		repos.service, repos.ingress, repos.configMap,
		repos.secret, repos.namespace, repos.event,
		repos.job, repos.cronJob, repos.pv, repos.pvc,
		repos.resourceQuota, repos.limitRange,
		repos.networkPolicy, repos.serviceAccount,
		k8srepo.NewRouteRepository(k8sClient, repos.ingress),
		otelSummaryRepo,
		dashboardRepo,
		conc,
	)

	commandSvc := commandsvc.NewCommandService(
		k8sClient,
		repos.pod, repos.generic,
		traceQueryRepo, logQueryRepo, metricsQueryRepo, sloQueryRepo,
	)

	// 5. 初始化 Scheduler (调度层)
	schedCfg := scheduler.Config{
		SnapshotInterval:    cfg.Scheduler.SnapshotInterval,
		CommandPollInterval: cfg.Scheduler.CommandPollInterval,
		HeartbeatInterval:   cfg.Scheduler.HeartbeatInterval,
		SnapshotTimeout:     cfg.Timeout.SnapshotCollect,
		SnapshotPushTimeout: cfg.Timeout.HTTPClient,
		CommandPollTimeout:  cfg.Timeout.CommandPoll,
		HeartbeatTimeout:    cfg.Timeout.Heartbeat,
	}
	sched := scheduler.New(schedCfg, snapshotSvc, commandSvc, masterGw)

	return &Agent{
		scheduler: sched,
		chClient:  chClient,
	}, nil
}

// Run 运行 Agent
//
// 启动调度器后阻塞等待退出信号 (SIGINT/SIGTERM)。
// 收到信号后优雅停止调度器。
//
// 参数:
//   - ctx: 上下文，可用于外部取消
//
// 返回:
//   - error: 调度器停止时的错误
func (a *Agent) Run(ctx context.Context) error {
	if err := a.scheduler.Start(ctx); err != nil {
		return err
	}

	// 枚举契约自检: otel_traces / otel_logs 的枚举字符串由 Collector exporter 或
	// 上报方 SDK 决定，取值不符时查询会静默返回空行。启动时查一次并每 10 分钟重跑 ——
	// 被观测应用往往在 Agent 启动之后才开始上报，只查一次会错过。
	if a.chClient != nil {
		go chquery.RunEnumContractChecks(ctx, a.chClient, 10*time.Minute)
	}

	// 启动健康检查 HTTP 服务器（供 K8s liveness/readiness 探针使用）
	healthMux := http.NewServeMux()
	healthMux.HandleFunc("/health", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	})
	healthSrv := &http.Server{Addr: ":8082", Handler: healthMux}
	go func() {
		if err := healthSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Error("健康检查服务器异常", "err", err)
		}
	}()

	log.Info("Agent 启动成功", "healthPort", 8082)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	log.Info("正在关闭...")

	// 关闭健康检查服务器
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := healthSrv.Shutdown(shutdownCtx); err != nil {
		log.Error("关闭健康检查服务器失败", "err", err)
	}

	// 关闭 ClickHouse 连接
	if a.chClient != nil {
		if err := a.chClient.Close(); err != nil {
			log.Error("关闭 ClickHouse 连接失败", "err", err)
		}
	}

	return a.scheduler.Stop()
}

// =============================================================================
// 内部辅助
// =============================================================================

// repositories 聚合所有 Repository 实例
// 用于简化依赖传递
type repositories struct {
	pod            repository.PodRepository
	node           repository.NodeRepository
	deployment     repository.DeploymentRepository
	statefulSet    repository.StatefulSetRepository
	daemonSet      repository.DaemonSetRepository
	replicaSet     repository.ReplicaSetRepository
	service        repository.ServiceRepository
	ingress        repository.IngressRepository
	configMap      repository.ConfigMapRepository
	secret         repository.SecretRepository
	namespace      repository.NamespaceRepository
	event          repository.EventRepository
	job            repository.JobRepository
	cronJob        repository.CronJobRepository
	pv             repository.PersistentVolumeRepository
	pvc            repository.PersistentVolumeClaimRepository
	resourceQuota  repository.ResourceQuotaRepository
	limitRange     repository.LimitRangeRepository
	networkPolicy  repository.NetworkPolicyRepository
	serviceAccount repository.ServiceAccountRepository
	generic        repository.GenericRepository
}

// initRepositories 初始化所有 Repository
// 每个 Repository 封装对应 K8s 资源的 CRUD 操作
func initRepositories(client sdkpkg.K8sClient) *repositories {
	return &repositories{
		pod:            k8srepo.NewPodRepository(client),
		node:           k8srepo.NewNodeRepository(client),
		deployment:     k8srepo.NewDeploymentRepository(client),
		statefulSet:    k8srepo.NewStatefulSetRepository(client),
		daemonSet:      k8srepo.NewDaemonSetRepository(client),
		replicaSet:     k8srepo.NewReplicaSetRepository(client),
		service:        k8srepo.NewServiceRepository(client),
		ingress:        k8srepo.NewIngressRepository(client),
		configMap:      k8srepo.NewConfigMapRepository(client),
		secret:         k8srepo.NewSecretRepository(client),
		namespace:      k8srepo.NewNamespaceRepository(client),
		event:          k8srepo.NewEventRepository(client),
		job:            k8srepo.NewJobRepository(client),
		cronJob:        k8srepo.NewCronJobRepository(client),
		pv:             k8srepo.NewPersistentVolumeRepository(client),
		pvc:            k8srepo.NewPersistentVolumeClaimRepository(client),
		resourceQuota:  k8srepo.NewResourceQuotaRepository(client),
		limitRange:     k8srepo.NewLimitRangeRepository(client),
		networkPolicy:  k8srepo.NewNetworkPolicyRepository(client),
		serviceAccount: k8srepo.NewServiceAccountRepository(client),
		generic:        k8srepo.NewGenericRepository(client),
	}
}
