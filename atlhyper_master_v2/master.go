// Package atlhyper_master_v2 Master V2 核心包
//
// 本包提供 Master 的启动器，负责:
//   - 初始化所有依赖 (Store, CommandBus, Database, Processor, Query, CommandService, AgentSDK, Gateway)
//   - 依赖注入和组装
//   - 生命周期管理 (启动、运行、停止)
//
// 架构设计:
//   - 外部访问: Gateway → Query（读取）/ CommandService（写入）→ Store / CommandBus
//   - 内部处理: AgentSDK → Processor → Store; AgentSDK → CommandBus
//   - Gateway 禁止直接访问 Store
//
// 使用方式:
//
//	config.LoadConfig()
//	master, err := atlhyper_master_v2.NewMaster()
//	master.Run(ctx)
package atlhyper_master_v2

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/signal"
	"syscall"

	"AtlHyper/atlhyper_master_v2/agentsdk"
	"AtlHyper/atlhyper_master_v2/ai"
	"AtlHyper/atlhyper_master_v2/aiops"
	aiopscore "AtlHyper/atlhyper_master_v2/aiops/core"
	"AtlHyper/atlhyper_master_v2/aiops/enricher"
	"AtlHyper/atlhyper_master_v2/config"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/database/repo"
	"AtlHyper/atlhyper_master_v2/database/sqlite"
	"AtlHyper/atlhyper_master_v2/datahub"
	"AtlHyper/atlhyper_master_v2/deployer"
	"AtlHyper/atlhyper_master_v2/gateway"
	"AtlHyper/atlhyper_master_v2/github"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/mq"
	"AtlHyper/atlhyper_master_v2/notifier"
	"AtlHyper/atlhyper_master_v2/notifier/trigger"
	"AtlHyper/atlhyper_master_v2/processor"
	"AtlHyper/atlhyper_master_v2/service"
	"AtlHyper/atlhyper_master_v2/service/operations"
	"AtlHyper/atlhyper_master_v2/service/query"
	"AtlHyper/atlhyper_master_v2/service/sync"
	"AtlHyper/atlhyper_master_v2/slo"
	"AtlHyper/atlhyper_master_v2/tester"
	"AtlHyper/common/logger"
	"AtlHyper/model_v3/command"
)

var log = logger.Module("Master")

// Master 是 Master V2 的主结构体
type Master struct {
	store        datahub.Store
	bus          mq.CommandBus
	database     *database.DB
	processor    processor.Processor
	service      service.Service
	agentSDK     *agentsdk.Server
	gateway      *gateway.Server
	testerServer *tester.Server
	eventPersist *sync.EventPersistService
	alertManager notifier.AlertManager
	heartbeat    *trigger.HeartbeatTrigger
	eventTrigger *trigger.EventTrigger
	// AIOps 引擎
	aiopsEngine aiops.Engine
	// Deployer（GitOps CD）
	deployer deployer.Deployer
}

// NewMaster 创建并初始化 Master 实例
func NewMaster() (*Master, error) {
	cfg := &config.GlobalConfig

	// 1. 初始化 Store (数据存储)
	store := datahub.NewStore(datahub.Config{
		Type:              cfg.DataHub.Type,
		EventRetention:    cfg.DataHub.EventRetention,
		HeartbeatExpire:   cfg.DataHub.HeartbeatExpire,
		SnapshotRetention: cfg.DataHub.SnapshotRetention,
		RedisAddr:         cfg.Redis.Addr,
		RedisPassword:     cfg.Redis.Password,
		RedisDB:           cfg.Redis.DB,
	})
	log.Info("Store 初始化完成", "type", cfg.DataHub.Type)

	// 2. 初始化 CommandBus (消息队列)
	bus := mq.NewCommandBus(mq.Config{
		Type:          cfg.DataHub.Type,
		RedisAddr:     cfg.Redis.Addr,
		RedisPassword: cfg.Redis.Password,
		RedisDB:       cfg.Redis.DB,
	})
	log.Info("CommandBus 初始化完成")

	// 3. 初始化 Database
	dialect := sqlite.NewDialect()
	db, err := database.NewDatabase(database.Config{
		Type: cfg.Database.Type,
		Path: cfg.Database.Path,
	}, dialect)
	if err != nil {
		return nil, fmt.Errorf("failed to init database: %w", err)
	}

	// 设置 API Key 加密密钥（使用 JWT Secret）
	if err := repo.SetEncryptionSecret(cfg.JWT.SecretKey); err != nil {
		return nil, fmt.Errorf("failed to set encryption secret: %w", err)
	}

	repo.Init(db, dialect)
	log.Info("数据库初始化完成", "type", cfg.Database.Type)

	// 3.1 初始化 AI Settings（首次启动，从配置文件读取默认值）
	if err := database.InitAISettings(context.Background(), db, &cfg.AI); err != nil {
		log.Warn("AI Settings 初始化失败", "err", err)
	}

	// 3.3 种子 AI Provider（从环境变量自动创建，仅首次无数据时生效）
	if err := database.SeedAIProvider(context.Background(), db, &cfg.AI.Seed); err != nil {
		log.Warn("AI Provider 种子初始化失败", "err", err)
	}

	// 4. 初始化 EventPersistService
	eventPersist := sync.NewEventPersistService(
		store,
		db.Event,
		sync.EventPersistConfig{
			RetentionDays:   cfg.Event.RetentionDays,
			MaxCount:        cfg.Event.MaxCount,
			CleanupInterval: cfg.Event.CleanupInterval,
		},
	)
	log.Info("事件持久化服务初始化完成")

	// 4.1 初始化 SLO 路由映射更新器（仅保留路由映射写入，时序数据已迁移至 OTelSnapshot + ClickHouse）
	sloRouteUpdater := slo.NewRouteUpdater(db.SLO)
	log.Info("SLO 路由映射更新器初始化完成")

	// 4.4 初始化 AIOps 引擎
	var aiopsEngine aiops.Engine
	aiopsEngine = aiopscore.NewEngine(aiopscore.EngineConfig{
		Store:         store,
		GraphRepo:     db.AIOpsGraph,
		BaselineRepo:  db.AIOpsBaseline,
		IncidentRepo:  db.AIOpsIncident,
		SLORepo:       db.SLO,
		FlushInterval: cfg.AIOps.FlushInterval,
	})
	log.Info("AIOps 引擎初始化完成")

	// 5. 初始化 Processor（写入路径）
	proc := processor.NewProcessor(processor.Config{
		Store: store,
		OnSnapshotReceived: func(clusterID string) {
			// 同步事件到数据库
			if err := eventPersist.Sync(clusterID); err != nil {
				log.Error("事件同步失败", "cluster", clusterID, "err", err)
			}
			// 更新 SLO 路由映射（时序数据已迁移，仅保留路由映射更新）
			if err := sloRouteUpdater.Sync(store, clusterID); err != nil {
				log.Error("SLO 路由映射更新失败", "cluster", clusterID, "err", err)
			}
			// AIOps 引擎处理
			aiopsEngine.OnSnapshot(clusterID)
		},
	})
	log.Info("数据处理器初始化完成")

	// 6. 初始化 Operations（写入路径，AI Service 依赖 cmdOps）
	cmdOps := operations.NewCommandService(bus, db.Command)
	adminOps := operations.NewAdminService(db.Notify, db.Settings, db.AIProvider, db.AISettings, db.AIRoleBudget)
	log.Info("操作服务初始化完成")

	// 7. 初始化 AI Service（Enricher 依赖 AIService）
	aiService := ai.NewService(
		ai.ServiceConfig{
			ToolTimeout: cfg.AI.ToolTimeout,
		},
		cmdOps, bus,
		db.AIProvider, db.AISettings, db.AIModel, db.AIRoleBudget,
		db.AIConversation, db.AIMessage,
	)
	log.Info("AI 服务初始化完成 (动态配置)")

	// 7.1 初始化 AIOps Enricher（通过 ai.AIService 接口调用 LLM，不再直接操作 ai/llm）
	aiopsEnricher := enricher.NewEnricher(db.AIOpsIncident, db.AIReport, aiService)
	aiopsEnricher.SetStore(store) // OTel 上下文丰富：读取 OTelSnapshot
	aiopsEnricher.EnableBackgroundTrigger(db.AIRoleBudget)
	aiopsEngine.SetIncidentNotify(aiopsEnricher.NotifyIncidentEvent)
	log.Info("AIOps Enricher 初始化完成（后台自动分析已启用）")

	// 8. 初始化 Query（读取路径）— 全部依赖通过构造函数注入
	q := query.NewQueryService(query.QueryServiceDeps{
		Store:       store,
		Bus:         bus,
		EventRepo:   db.Event,
		SLORepo:     db.SLO,
		AIOpsEngine: aiopsEngine,
		AIOpsAI:     aiopsEnricher,
		AdminRepos: query.AdminRepos{
			Audit:      db.Audit,
			Command:    db.Command,
			Notify:     db.Notify,
			Settings:   db.Settings,
			AIProvider: db.AIProvider,
			AISettings: db.AISettings,
			AIModel:    db.AIModel,
			AIBudget:   db.AIRoleBudget,
			AIReport:   db.AIReport,
		},
		DeployRepos: query.DeployRepos{
			DeployConfig:  db.DeployConfig,
			DeployHistory: db.DeployHistory,
			GitHubInstall: db.GitHubInstall,
			RepoConfig:    db.RepoConfig,
		},
		UserRepo: db.User,
	})
	log.Info("查询层初始化完成")

	// 初始化 SLO 写入服务
	sloOps := operations.NewSLOService(db.SLO)

	// 初始化部署 / GitHub 集成写入服务
	deployOps := operations.NewDeployService(db.DeployConfig, db.GitHubInstall)

	// 初始化用户写入服务
	userOps := operations.NewUserService(db.User)

	// 组合统一 Service
	svc := service.NewService(q, cmdOps, adminOps, sloOps, deployOps, userOps)

	// 8. 初始化 AgentSDK
	agentServer := agentsdk.NewServer(agentsdk.Config{
		Port:           cfg.Server.AgentSDKPort,
		CommandTimeout: cfg.Timeout.CommandPoll,
		Bus:            bus,
		Processor:      proc,
		CmdRepo:        db.Command,
	})
	log.Info("AgentSDK 初始化完成", "port", cfg.Server.AgentSDKPort)

	// 9. 注册 AIOps Tool（AI Chat 中使用）
	aiService.RegisterTool("analyze_incident", func(ctx context.Context, clusterID string, params map[string]interface{}) (string, error) {
		incidentID := getStringParam(params, "incident_id")
		if incidentID == "" {
			return "缺少参数 incident_id", nil
		}
		result, err := aiopsEnricher.Summarize(ctx, incidentID)
		if err != nil {
			return fmt.Sprintf("分析事件失败: %v", err), nil
		}
		data, _ := json.Marshal(result)
		return string(data), nil
	})

	aiService.RegisterTool("get_cluster_risk", func(ctx context.Context, clusterID string, params map[string]interface{}) (string, error) {
		topN := getIntParam(params, "top_n", 10)
		risk := aiopsEngine.GetClusterRisk(clusterID)
		entities := aiopsEngine.GetEntityRisks(clusterID, "r_final", topN)
		data, _ := json.Marshal(map[string]interface{}{
			"clusterRisk": scaleClusterRiskForAI(risk),
			"topEntities": scaleEntityRisksForAI(entities),
			"entityCount": len(entities),
		})
		return string(data), nil
	})

	aiService.RegisterTool("get_recent_incidents", func(ctx context.Context, clusterID string, params map[string]interface{}) (string, error) {
		state := getStringParam(params, "state")
		limit := getIntParam(params, "limit", 10)
		opts := aiops.IncidentQueryOpts{
			ClusterID: clusterID,
			State:     state,
			Limit:     limit,
			Offset:    0,
		}
		incidents, total, err := aiopsEngine.GetIncidents(ctx, opts)
		if err != nil {
			return fmt.Sprintf("获取事件列表失败: %v", err), nil
		}
		data, _ := json.Marshal(map[string]interface{}{
			"incidents": incidents,
			"total":     total,
		})
		return string(data), nil
	})
	// 9.1 注册 OTel 查询 Tool（Command 路径：query_traces / query_logs）
	toolTimeout := cfg.AI.ToolTimeout

	aiService.RegisterTool("query_traces", func(ctx context.Context, clusterID string, params map[string]interface{}) (string, error) {
		cmdParams := map[string]interface{}{
			"sub_action": "list_traces",
			"limit":      10,
		}
		if s := getStringParam(params, "service"); s != "" {
			cmdParams["service"] = s
		}
		if s := getStringParam(params, "operation"); s != "" {
			cmdParams["operation"] = s
		}
		if v, ok := params["min_duration_ms"]; ok {
			if d, ok := v.(float64); ok && d > 0 {
				cmdParams["min_duration_ms"] = d
			}
		}
		if s := getStringParam(params, "status_code"); s != "" {
			cmdParams["status_code"] = s
		}
		since := getStringParam(params, "since")
		if since == "" {
			since = "1h"
		}
		cmdParams["since"] = since

		resp, err := cmdOps.CreateCommand(&model.CreateCommandRequest{
			ClusterID: clusterID,
			Action:    command.ActionQueryTraces,
			Source:    "ai",
			Params:    cmdParams,
		})
		if err != nil {
			return fmt.Sprintf("创建查询命令失败: %v", err), nil
		}
		result, err := bus.WaitCommandResult(ctx, resp.CommandID, toolTimeout)
		if err != nil {
			return fmt.Sprintf("查询超时: %v", err), nil
		}
		if result == nil {
			return "查询超时: 未收到 Agent 响应", nil
		}
		if !result.Success {
			return fmt.Sprintf("查询失败: %s", result.Error), nil
		}
		return ai.TruncateToolResult(result.Output, "traces"), nil
	})

	aiService.RegisterTool("query_logs", func(ctx context.Context, clusterID string, params map[string]interface{}) (string, error) {
		cmdParams := map[string]interface{}{
			"limit": 20,
		}
		if s := getStringParam(params, "query"); s != "" {
			cmdParams["query"] = s
		}
		if s := getStringParam(params, "service"); s != "" {
			cmdParams["service"] = s
		}
		if s := getStringParam(params, "level"); s != "" {
			cmdParams["level"] = s
		}
		if s := getStringParam(params, "trace_id"); s != "" {
			cmdParams["trace_id"] = s
		}
		since := getStringParam(params, "since")
		if since == "" {
			since = "1h"
		}
		cmdParams["since"] = since

		resp, err := cmdOps.CreateCommand(&model.CreateCommandRequest{
			ClusterID: clusterID,
			Action:    command.ActionQueryLogs,
			Source:    "ai",
			Params:    cmdParams,
		})
		if err != nil {
			return fmt.Sprintf("创建查询命令失败: %v", err), nil
		}
		result, err := bus.WaitCommandResult(ctx, resp.CommandID, toolTimeout)
		if err != nil {
			return fmt.Sprintf("查询超时: %v", err), nil
		}
		if result == nil {
			return "查询超时: 未收到 Agent 响应", nil
		}
		if !result.Success {
			return fmt.Sprintf("查询失败: %s", result.Error), nil
		}
		return ai.TruncateToolResult(result.Output, "logs"), nil
	})

	// 9.2 注册内存直读 Tool（query_slo / get_entity_detail）
	aiService.RegisterTool("query_slo", func(ctx context.Context, clusterID string, params map[string]interface{}) (string, error) {
		window := getStringParam(params, "window")
		if window == "" {
			window = "7d"
		}
		serviceName := getStringParam(params, "service")
		domain := getStringParam(params, "domain")

		snapshot, err := store.GetSnapshot(clusterID)
		if err != nil || snapshot == nil || snapshot.OTel == nil {
			return "当前无 OTel 数据", nil
		}
		otel := snapshot.OTel

		var services []map[string]interface{}

		// 从 SLOWindows 获取预聚合数据
		// 过滤规则: 指定 service → 只返回 mesh; 指定 domain → 只返回 ingress; 都不指定 → 返回全部
		if windowData, ok := otel.SLOWindows[window]; ok && windowData != nil {
			if serviceName == "" { // 未指定 service 时才包含 ingress
				for _, s := range windowData.Current {
					if domain != "" && s.ServiceKey != domain {
						continue
					}
					services = append(services, map[string]interface{}{
						"name":        s.ServiceKey,
						"type":        "ingress",
						"rps":         s.RPS,
						"successRate": s.SuccessRate,
						"errorRate":   s.ErrorRate,
						"p50Ms":       s.P50Ms,
						"p90Ms":       s.P90Ms,
						"p99Ms":       s.P99Ms,
					})
				}
			}
		} else {
			// 降级：从实时 SLO 列表获取
			if serviceName == "" {
				for _, s := range otel.SLOIngress {
					if domain != "" && s.ServiceKey != domain {
						continue
					}
					services = append(services, map[string]interface{}{
						"name":        s.ServiceKey,
						"type":        "ingress",
						"rps":         s.RPS,
						"successRate": s.SuccessRate,
						"errorRate":   s.ErrorRate,
						"p50Ms":       s.P50Ms,
						"p90Ms":       s.P90Ms,
						"p99Ms":       s.P99Ms,
					})
				}
			}
		}

		if len(services) > 50 {
			services = services[:50]
		}

		data, _ := json.Marshal(map[string]interface{}{
			"window":   window,
			"services": services,
			"count":    len(services),
		})
		return string(data), nil
	})

	aiService.RegisterTool("get_entity_detail", func(ctx context.Context, clusterID string, params map[string]interface{}) (string, error) {
		entityType := getStringParam(params, "entity_type")
		entityName := getStringParam(params, "entity_name")
		namespace := getStringParam(params, "namespace")

		if entityType == "" || entityName == "" {
			return "缺少参数 entity_type 和 entity_name", nil
		}

		entityKey := ai.BuildEntityKey(entityType, namespace, entityName)
		detail := aiopsEngine.GetEntityRisk(clusterID, entityKey)
		if detail == nil {
			return fmt.Sprintf("未找到实体 %s（可能不在依赖图中或无异常数据）", entityKey), nil
		}

		result := ai.SimplifyEntityDetail(detail)
		data, _ := json.Marshal(result)
		return string(data), nil
	})

	log.Info("AI Tool 注册完成 (8 个基础)")

	// 10. 初始化 AlertManager（告警管理器）
	alertMgr, err := notifier.NewManager(db.Notify)
	if err != nil {
		return nil, fmt.Errorf("failed to init alert manager: %w", err)
	}
	log.Info("告警管理器初始化完成")

	// 11. 初始化 HeartbeatTrigger（心跳检测触发器）
	heartbeat := trigger.NewHeartbeatTrigger(store, alertMgr, trigger.HeartbeatConfig{
		CheckInterval: cfg.DataHub.HeartbeatExpire / 2,
		OfflineAfter:  cfg.DataHub.HeartbeatExpire,
	})
	log.Info("心跳检测触发器初始化完成")

	// 11.1 初始化 EventTrigger（事件告警触发器，可选）
	var eventTrigger *trigger.EventTrigger
	if cfg.EventAlert.Enabled {
		eventTrigger = trigger.NewEventTrigger(
			db.Event,
			q,
			alertMgr,
			trigger.EventConfig{
				CheckInterval: cfg.EventAlert.CheckInterval,
			},
		)
		log.Info("事件告警触发器初始化完成")
	}

	// 11.5 初始化 GitHub Client（可选，未配置则跳过）
	var ghClient github.Client
	if cfg.GitHub.AppID > 0 && cfg.GitHub.PrivateKeyPath != "" {
		var err error
		ghClient, err = github.NewClient(github.Config{
			AppID:          cfg.GitHub.AppID,
			AppSlug:        cfg.GitHub.AppSlug,
			PrivateKeyPath: cfg.GitHub.PrivateKeyPath,
			CallbackURL:    cfg.GitHub.CallbackURL,
		})
		if err != nil {
			log.Warn("GitHub Client 初始化失败，GitHub 集成不可用", "err", err)
		} else {
			// 从数据库恢复 Installation ID
			inst, _ := db.GitHubInstall.Get(context.Background())
			if inst != nil {
				if setter, ok := ghClient.(interface{ SetInstallationID(int64) }); ok {
					setter.SetInstallationID(inst.InstallationID)
				}
				log.Info("GitHub Client 初始化完成", "installationID", inst.InstallationID)
			} else {
				log.Info("GitHub Client 初始化完成（未连接）")
			}
		}
	}

	// 11.6 注册 GitHub + CD Tool（仅在 GitHub Client 可用时注册）
	if ghClient != nil {
		aiService.RegisterTool("get_deploy_history", ai.NewDeployHistoryHandler(db.DeployHistory))
		aiService.RegisterTool("rollback_deployment", ai.NewRollbackHandler(db.DeployHistory))
		aiService.RegisterTool("github_read_file", ai.NewGitHubReadFileHandler(ghClient))
		aiService.RegisterTool("github_search_code", ai.NewGitHubSearchCodeHandler(ghClient))
		aiService.RegisterTool("github_recent_commits", ai.NewGitHubRecentCommitsHandler(ghClient))
		log.Info("GitHub + CD Tool 注册完成 (5 个)")
	}

	// 11.7 初始化 Deployer（GitOps CD，仅在 GitHub Client 可用时启用）
	var deployerService deployer.Deployer
	if ghClient != nil {
		deployerService = deployer.NewService(ghClient, db, bus)
		log.Info("Deployer 初始化完成")
	}

	// 12. 初始化 Gateway
	gw := gateway.NewServer(gateway.Config{
		Port:           cfg.Server.GatewayPort,
		Service:        svc,
		Database:       db,
		AIService:      aiService,
		AnalyzeTrigger: aiopsEnricher,
		GitHubClient:   ghClient,
		Deployer:       deployerService,
	})
	log.Info("Gateway 初始化完成", "port", cfg.Server.GatewayPort)

	// 13. 初始化 Tester
	testerServer := tester.NewServer(tester.Config{
		Port:         cfg.Server.TesterPort,
		AlertManager: alertMgr,
	})
	log.Info("Tester 初始化完成", "port", cfg.Server.TesterPort)

	return &Master{
		store:        store,
		bus:          bus,
		database:     db,
		processor:    proc,
		service:      svc,
		agentSDK:     agentServer,
		gateway:      gw,
		testerServer: testerServer,
		eventPersist: eventPersist,
		alertManager: alertMgr,
		heartbeat:    heartbeat,
		eventTrigger: eventTrigger,
		aiopsEngine:  aiopsEngine,
		deployer:     deployerService,
	}, nil
}

// Run 运行 Master
func (m *Master) Run(ctx context.Context) error {
	// 启动 Store
	if err := m.store.Start(); err != nil {
		return fmt.Errorf("failed to start store: %w", err)
	}

	// 启动 CommandBus
	if err := m.bus.Start(); err != nil {
		return fmt.Errorf("failed to start commandbus: %w", err)
	}

	// 启动 EventPersistService
	if err := m.eventPersist.Start(); err != nil {
		return fmt.Errorf("failed to start event persist: %w", err)
	}

	// 启动 AlertManager
	if err := m.alertManager.Start(); err != nil {
		return fmt.Errorf("failed to start alert manager: %w", err)
	}

	// 启动 HeartbeatTrigger
	if err := m.heartbeat.Start(); err != nil {
		return fmt.Errorf("failed to start heartbeat trigger: %w", err)
	}

	// 启动 EventTrigger
	if m.eventTrigger != nil {
		if err := m.eventTrigger.Start(); err != nil {
			return fmt.Errorf("failed to start event trigger: %w", err)
		}
	}

	// 启动 AIOps 引擎
	if m.aiopsEngine != nil {
		if err := m.aiopsEngine.Start(ctx); err != nil {
			return fmt.Errorf("failed to start aiops engine: %w", err)
		}
	}

	// 启动 Deployer
	if m.deployer != nil {
		if err := m.deployer.Start(ctx); err != nil {
			log.Warn("Deployer 启动失败", "err", err)
		}
	}

	// 启动 AgentSDK
	if err := m.agentSDK.Start(); err != nil {
		return fmt.Errorf("failed to start agentsdk: %w", err)
	}

	// 启动 Gateway
	if err := m.gateway.Start(); err != nil {
		return fmt.Errorf("failed to start gateway: %w", err)
	}

	// 启动 Tester
	if err := m.testerServer.Start(); err != nil {
		return fmt.Errorf("failed to start tester: %w", err)
	}

	log.Info("Master 启动成功",
		"gateway_port", config.GlobalConfig.Server.GatewayPort,
		"agentsdk_port", config.GlobalConfig.Server.AgentSDKPort,
	)

	// 等待退出信号
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh

	// 优雅停止
	log.Info("正在关闭...")
	return m.Stop()
}

// Stop 停止 Master
func (m *Master) Stop() error {
	// 停止 Tester
	if err := m.testerServer.Stop(); err != nil {
		log.Error("停止 Tester 失败", "err", err)
	}

	// 停止 Gateway
	if err := m.gateway.Stop(); err != nil {
		log.Error("停止 Gateway 失败", "err", err)
	}

	// 停止 AgentSDK
	if err := m.agentSDK.Stop(); err != nil {
		log.Error("停止 AgentSDK 失败", "err", err)
	}

	// 停止 HeartbeatTrigger
	if err := m.heartbeat.Stop(); err != nil {
		log.Error("停止心跳检测触发器失败", "err", err)
	}

	// 停止 EventTrigger
	if m.eventTrigger != nil {
		if err := m.eventTrigger.Stop(); err != nil {
			log.Error("停止事件告警触发器失败", "err", err)
		}
	}

	// 停止 AIOps 引擎
	if m.aiopsEngine != nil {
		if err := m.aiopsEngine.Stop(); err != nil {
			log.Error("停止 AIOps 引擎失败", "err", err)
		}
	}

	// 停止 Deployer
	if m.deployer != nil {
		if err := m.deployer.Stop(); err != nil {
			log.Error("停止 Deployer 失败", "err", err)
		}
	}

	// 停止 AlertManager
	m.alertManager.Stop()

	// 停止 EventPersistService
	if err := m.eventPersist.Stop(); err != nil {
		log.Error("停止事件持久化服务失败", "err", err)
	}

	// 停止 CommandBus
	if err := m.bus.Stop(); err != nil {
		log.Error("停止 CommandBus 失败", "err", err)
	}

	// 停止 Store
	if err := m.store.Stop(); err != nil {
		log.Error("停止 Store 失败", "err", err)
	}

	// 关闭数据库
	if err := m.database.Close(); err != nil {
		log.Error("关闭数据库失败", "err", err)
	}

	log.Info("Master 已停止")
	return nil
}

// getStringParam 从 map 中安全获取字符串
func getStringParam(m map[string]interface{}, key string) string {
	if v, ok := m[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

// getIntParam 从 map 中安全获取整数（带默认值）
func getIntParam(m map[string]interface{}, key string, defaultVal int) int {
	if v, ok := m[key]; ok {
		switch n := v.(type) {
		case float64:
			return int(n)
		case int:
			return n
		}
	}
	return defaultVal
}

// scaleEntityRisksForAI 将 EntityRisk 列表的分数从 [0,1] 转为 [0,100]（AI tool 输出用）
func scaleEntityRisksForAI(src []*aiops.EntityRisk) []map[string]interface{} {
	result := make([]map[string]interface{}, len(src))
	for i, r := range src {
		result[i] = map[string]interface{}{
			"entityKey":    r.EntityKey,
			"entityType":   r.EntityType,
			"namespace":    r.Namespace,
			"name":         r.Name,
			"rFinal":       math.Round(r.RFinal * 100),
			"riskLevel":    r.RiskLevel,
			"firstAnomaly": r.FirstAnomaly,
		}
	}
	return result
}

// scaleClusterRiskForAI 转换 ClusterRisk（Risk 已是 0-100，TopEntities 需要转换）
func scaleClusterRiskForAI(src *aiops.ClusterRisk) map[string]interface{} {
	if src == nil {
		return nil
	}
	return map[string]interface{}{
		"clusterId":     src.ClusterID,
		"risk":          src.Risk,
		"level":         src.Level,
		"totalEntities": src.TotalEntities,
		"anomalyCount":  src.AnomalyCount,
	}
}

// Store 获取 Store 实例（供测试使用）
func (m *Master) Store() datahub.Store {
	return m.store
}

// Bus 获取 CommandBus 实例（供测试使用）
func (m *Master) Bus() mq.CommandBus {
	return m.bus
}

// Database 获取 Database 实例（供测试使用）
func (m *Master) Database() *database.DB {
	return m.database
}

// Service 获取 Service 实例（供测试使用）
func (m *Master) Service() service.Service {
	return m.service
}
