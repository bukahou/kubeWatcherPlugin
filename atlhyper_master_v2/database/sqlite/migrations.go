// atlhyper_master_v2/database/sqlite/migrations.go
// 数据库迁移
package sqlite

import (
	"database/sql"
	"time"

	"AtlHyper/atlhyper_master_v2/config"
	"AtlHyper/common/logger"

	"golang.org/x/crypto/bcrypt"
)

var log = logger.Module("Migrations")

// migrate 执行数据库迁移
func migrate(db *sql.DB) error {
	migrations := []string{
		// ==================== 集群事件表（只存 Warning，业务去重）====================
		// dedup_key = MD5(cluster_id + involved_kind + involved_namespace + involved_name + reason)
		// 同一资源的同一 Reason 只存一条，更新 count/last_timestamp/message
		`CREATE TABLE IF NOT EXISTS cluster_events (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			dedup_key TEXT UNIQUE NOT NULL,
			cluster_id TEXT NOT NULL,
			namespace TEXT NOT NULL,
			name TEXT NOT NULL,
			type TEXT NOT NULL DEFAULT 'Warning',
			reason TEXT NOT NULL,
			message TEXT,
			source_component TEXT,
			source_host TEXT,
			involved_kind TEXT,
			involved_name TEXT,
			involved_namespace TEXT,
			first_timestamp TEXT NOT NULL,
			last_timestamp TEXT NOT NULL,
			count INTEGER DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_events_cluster_time ON cluster_events(cluster_id, last_timestamp DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_events_involved ON cluster_events(cluster_id, involved_kind, involved_namespace, involved_name)`,
		`CREATE INDEX IF NOT EXISTS idx_events_reason ON cluster_events(cluster_id, reason, last_timestamp DESC)`,

		// ==================== 通知渠道表 ====================
		`CREATE TABLE IF NOT EXISTS notify_channels (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			type TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			enabled INTEGER DEFAULT 0,
			config TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,

		// ==================== 用户表 ====================
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT UNIQUE NOT NULL,
			password_hash TEXT NOT NULL,
			display_name TEXT,
			email TEXT,
			role INTEGER DEFAULT 3,
			status INTEGER DEFAULT 1,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			last_login_at TEXT,
			last_login_ip TEXT
		)`,

		// ==================== 集群表 ====================
		`CREATE TABLE IF NOT EXISTS clusters (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cluster_uid TEXT UNIQUE NOT NULL,
			name TEXT NOT NULL,
			description TEXT,
			environment TEXT,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,

		// ==================== 审计日志表 ====================
		`CREATE TABLE IF NOT EXISTS audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			timestamp TEXT NOT NULL,
			user_id INTEGER,
			username TEXT,
			role INTEGER,
			source TEXT,
			action TEXT,
			resource TEXT,
			method TEXT,
			request_body TEXT,
			status_code INTEGER,
			success INTEGER,
			error_message TEXT,
			ip TEXT,
			user_agent TEXT,
			duration_ms INTEGER
		)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_timestamp ON audit_logs(timestamp DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_user ON audit_logs(user_id, timestamp DESC)`,

		// ==================== 指令历史表 ====================
		`CREATE TABLE IF NOT EXISTS command_history (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			command_id TEXT UNIQUE NOT NULL,
			cluster_id TEXT NOT NULL,
			source TEXT,
			user_id INTEGER,
			action TEXT NOT NULL,
			target_kind TEXT,
			target_namespace TEXT,
			target_name TEXT,
			params TEXT,
			status TEXT NOT NULL,
			result TEXT,
			error_message TEXT,
			created_at TEXT NOT NULL,
			started_at TEXT,
			finished_at TEXT,
			duration_ms INTEGER
		)`,
		`CREATE INDEX IF NOT EXISTS idx_cmd_cluster ON command_history(cluster_id, created_at DESC)`,

		// ==================== 系统设置表 ====================
		`CREATE TABLE IF NOT EXISTS settings (
			key TEXT PRIMARY KEY,
			value TEXT,
			description TEXT,
			updated_at TEXT NOT NULL,
			updated_by INTEGER
		)`,

		// ==================== AI 对话表 ====================
		`CREATE TABLE IF NOT EXISTS ai_conversations (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,
			cluster_id TEXT NOT NULL,
			title TEXT,
			message_count INTEGER DEFAULT 0,
			total_input_tokens INTEGER DEFAULT 0,
			total_output_tokens INTEGER DEFAULT 0,
			total_tool_calls INTEGER DEFAULT 0,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_conv_user ON ai_conversations(user_id, updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_conv_cluster ON ai_conversations(cluster_id)`,

		// ==================== AI 消息表 ====================
		`CREATE TABLE IF NOT EXISTS ai_messages (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			conversation_id INTEGER NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			tool_calls TEXT,
			created_at TEXT NOT NULL,
			FOREIGN KEY (conversation_id) REFERENCES ai_conversations(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_msg_conv ON ai_messages(conversation_id, created_at ASC)`,

		// ==================== AI 提供商配置表 ====================
		`CREATE TABLE IF NOT EXISTS ai_providers (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			name TEXT NOT NULL,
			provider TEXT NOT NULL,
			api_key TEXT NOT NULL,
			model TEXT NOT NULL,
			base_url TEXT DEFAULT '',
			description TEXT,
			roles TEXT DEFAULT '[]',
			context_window_override INTEGER DEFAULT 0,
			total_requests INTEGER DEFAULT 0,
			total_tokens INTEGER DEFAULT 0,
			total_cost REAL DEFAULT 0,
			last_used_at TEXT,
			last_error TEXT,
			last_error_at TEXT,
			status TEXT DEFAULT 'unknown',
			status_checked_at TEXT,
			created_at TEXT NOT NULL,
			created_by INTEGER,
			updated_at TEXT NOT NULL,
			updated_by INTEGER,
			deleted_at TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_provider_deleted ON ai_providers(deleted_at)`,

		// ==================== AI 全局设置表 (单行) ====================
		`CREATE TABLE IF NOT EXISTS ai_settings (
			id INTEGER PRIMARY KEY CHECK (id = 1),
			tool_timeout INTEGER DEFAULT 30,
			updated_at TEXT NOT NULL,
			updated_by INTEGER
		)`,

		// ==================== AI 提供商模型表 ====================
		// 各提供商支持的模型列表 (DB管理)
		`CREATE TABLE IF NOT EXISTS ai_provider_models (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			provider TEXT NOT NULL,
			model TEXT NOT NULL,
			display_name TEXT,
			is_default INTEGER DEFAULT 0,
			sort_order INTEGER DEFAULT 0,
			context_window INTEGER DEFAULT 0,
			created_at TEXT NOT NULL,
			UNIQUE(provider, model)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_models_provider ON ai_provider_models(provider, sort_order)`,

		// ==================== AI 角色预算表 ====================
		`CREATE TABLE IF NOT EXISTS ai_role_budget (
			role TEXT PRIMARY KEY,
			daily_input_token_limit INTEGER DEFAULT 0,
			daily_output_token_limit INTEGER DEFAULT 0,
			daily_call_limit INTEGER DEFAULT 0,
			daily_input_tokens_used INTEGER DEFAULT 0,
			daily_output_tokens_used INTEGER DEFAULT 0,
			daily_calls_used INTEGER DEFAULT 0,
			daily_reset_at TEXT,
			monthly_input_token_limit INTEGER DEFAULT 0,
			monthly_output_token_limit INTEGER DEFAULT 0,
			monthly_call_limit INTEGER DEFAULT 0,
			monthly_input_tokens_used INTEGER DEFAULT 0,
			monthly_output_tokens_used INTEGER DEFAULT 0,
			monthly_calls_used INTEGER DEFAULT 0,
			monthly_reset_at TEXT,
			fallback_provider_id INTEGER,
			auto_trigger_min_severity TEXT DEFAULT 'critical',
			auto_trigger_mode TEXT DEFAULT 'auto',
			schedule_start_time TEXT DEFAULT '',
			schedule_end_time TEXT DEFAULT '',
			updated_at TEXT NOT NULL
		)`,

		// ==================== AI 分析报告表 ====================
		`CREATE TABLE IF NOT EXISTS ai_reports (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			incident_id TEXT,
			cluster_id TEXT NOT NULL,
			role TEXT NOT NULL,
			trigger TEXT NOT NULL,
			summary TEXT,
			root_cause_analysis TEXT,
			recommendations TEXT,
			similar_incidents TEXT,
			investigation_steps TEXT,
			evidence_chain TEXT,
			provider_name TEXT,
			model TEXT,
			input_tokens INTEGER DEFAULT 0,
			output_tokens INTEGER DEFAULT 0,
			duration_ms INTEGER DEFAULT 0,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_reports_incident ON ai_reports(incident_id)`,
		`CREATE INDEX IF NOT EXISTS idx_ai_reports_cluster ON ai_reports(cluster_id, role, created_at DESC)`,

		// ==================== SLO: 目标配置表 ====================
		// SLO 目标配置，用户可配置不同周期的可用性和延迟目标
		`CREATE TABLE IF NOT EXISTS slo_targets (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cluster_id TEXT NOT NULL,
			host TEXT NOT NULL,
			ingress_name TEXT NOT NULL,
			ingress_class TEXT NOT NULL DEFAULT 'nginx',
			namespace TEXT NOT NULL,
			tls INTEGER NOT NULL DEFAULT 1,
			time_range TEXT NOT NULL,
			availability_target REAL NOT NULL DEFAULT 95.00,
			p95_latency_target INTEGER NOT NULL DEFAULT 300,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE(cluster_id, host, time_range)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_slo_targets_cluster ON slo_targets(cluster_id)`,

		// ==================== SLO: 路由映射表 ====================
		// 存储 Traefik service 名称到域名/路径的映射关系
		// Agent 采集 IngressRoute CRD 后上报，用于将 service 维度的指标转换为 domain/path 维度
		// 注意：同一个 service 可能服务于多个路径，所以唯一约束基于 domain + path
		`CREATE TABLE IF NOT EXISTS slo_route_mapping (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			cluster_id TEXT NOT NULL,
			domain TEXT NOT NULL,
			path_prefix TEXT NOT NULL DEFAULT '/',
			ingress_name TEXT NOT NULL,
			namespace TEXT NOT NULL,
			tls INTEGER NOT NULL DEFAULT 1,
			service_key TEXT NOT NULL,
			service_name TEXT NOT NULL,
			service_port INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			UNIQUE(cluster_id, domain, path_prefix)
		)`,
		`CREATE INDEX IF NOT EXISTS idx_route_mapping_domain ON slo_route_mapping(cluster_id, domain)`,
		`CREATE INDEX IF NOT EXISTS idx_route_mapping_service ON slo_route_mapping(cluster_id, service_key)`,

		// ==================== AIOps: 基线状态表 ====================
		// 持久化 EMA 状态，用于重启恢复
		`CREATE TABLE IF NOT EXISTS aiops_baseline_states (
			entity_key  TEXT NOT NULL,
			metric_name TEXT NOT NULL,
			ema         REAL NOT NULL,
			variance    REAL NOT NULL,
			count       INTEGER NOT NULL,
			updated_at  INTEGER NOT NULL,
			PRIMARY KEY (entity_key, metric_name)
		)`,

		// ==================== AIOps: 依赖图快照表 ====================
		// 定期持久化图快照，用于重启恢复（每集群一条，覆盖式更新）
		`CREATE TABLE IF NOT EXISTS aiops_dependency_graph_snapshots (
			cluster_id TEXT PRIMARY KEY,
			snapshot   BLOB NOT NULL,
			created_at TEXT DEFAULT CURRENT_TIMESTAMP
		)`,

		// ==================== AIOps: 事件表 ====================
		// 异常事件记录，关联集群和受影响实体
		`CREATE TABLE IF NOT EXISTS aiops_incidents (
			id TEXT PRIMARY KEY,
			cluster_id TEXT NOT NULL,
			state TEXT NOT NULL DEFAULT 'firing',
			severity TEXT NOT NULL DEFAULT 'warning',
			root_cause TEXT,
			peak_risk REAL NOT NULL DEFAULT 0,
			started_at TEXT NOT NULL,
			resolved_at TEXT,
			duration_s INTEGER NOT NULL DEFAULT 0,
			recurrence INTEGER NOT NULL DEFAULT 0,
			summary TEXT,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_aiops_incidents_cluster ON aiops_incidents(cluster_id, started_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_aiops_incidents_state ON aiops_incidents(state)`,

		// ==================== AIOps: 事件受影响实体表 ====================
		// 事件关联的实体信息（风险分数、角色等）
		`CREATE TABLE IF NOT EXISTS aiops_incident_entities (
			incident_id TEXT NOT NULL,
			entity_key TEXT NOT NULL,
			entity_type TEXT NOT NULL,
			r_local REAL NOT NULL DEFAULT 0,
			r_final REAL NOT NULL DEFAULT 0,
			role TEXT NOT NULL DEFAULT 'affected',
			PRIMARY KEY (incident_id, entity_key),
			FOREIGN KEY (incident_id) REFERENCES aiops_incidents(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_aiops_incident_entities_entity ON aiops_incident_entities(entity_key)`,

		// ==================== AIOps: 事件时间线表 ====================
		// 事件生命周期中的关键事件记录
		`CREATE TABLE IF NOT EXISTS aiops_incident_timeline (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			incident_id TEXT NOT NULL,
			timestamp TEXT NOT NULL,
			event_type TEXT NOT NULL,
			entity_key TEXT,
			detail TEXT,
			FOREIGN KEY (incident_id) REFERENCES aiops_incidents(id) ON DELETE CASCADE
		)`,
		`CREATE INDEX IF NOT EXISTS idx_aiops_incident_timeline_inc ON aiops_incident_timeline(incident_id, timestamp ASC)`,

		// ==================== GitHub App 安装记录（单行）====================
		`CREATE TABLE IF NOT EXISTS github_installations (
			id              INTEGER PRIMARY KEY,
			installation_id INTEGER NOT NULL UNIQUE,
			account_login   TEXT NOT NULL,
			created_at      TEXT NOT NULL
		)`,

		// ==================== 仓库映射配置 ====================
		`CREATE TABLE IF NOT EXISTS repo_config (
			id              INTEGER PRIMARY KEY AUTOINCREMENT,
			repo            TEXT NOT NULL UNIQUE,
			mapping_enabled INTEGER DEFAULT 0,
			created_at      TEXT NOT NULL,
			updated_at      TEXT NOT NULL
		)`,

		// ==================== 部署配置（每集群一条）====================
		`CREATE TABLE IF NOT EXISTS deploy_config (
			id            INTEGER PRIMARY KEY,
			cluster_id    TEXT NOT NULL UNIQUE,
			repo_url      TEXT NOT NULL,
			paths         TEXT NOT NULL DEFAULT '[]',
			interval_sec  INTEGER DEFAULT 60,
			auto_deploy   INTEGER DEFAULT 1,
			created_at    TEXT NOT NULL,
			updated_at    TEXT NOT NULL
		)`,

		// ==================== 部署历史 ====================
		`CREATE TABLE IF NOT EXISTS deploy_history (
			id                INTEGER PRIMARY KEY AUTOINCREMENT,
			cluster_id        TEXT NOT NULL,
			path              TEXT NOT NULL,
			namespace         TEXT NOT NULL,
			commit_sha        TEXT NOT NULL,
			commit_message    TEXT,
			commit_author     TEXT DEFAULT '',
			commit_avatar_url TEXT DEFAULT '',
			pr_number         INTEGER DEFAULT 0,
			pr_title          TEXT DEFAULT '',
			pr_url            TEXT DEFAULT '',
			changed_files     TEXT DEFAULT '[]',
			compare_url       TEXT DEFAULT '',
			source_repo       TEXT DEFAULT '',
			source_commit_sha TEXT DEFAULT '',
			deployed_at       TEXT NOT NULL,
			trigger           TEXT NOT NULL DEFAULT 'auto',
			status            TEXT DEFAULT 'pending',
			duration_ms       INTEGER DEFAULT 0,
			resource_total    INTEGER DEFAULT 0,
			resource_changed  INTEGER DEFAULT 0,
			error_message     TEXT
		)`,
		`CREATE INDEX IF NOT EXISTS idx_deploy_history_lookup ON deploy_history(cluster_id, path, deployed_at DESC)`,
	}

	for _, m := range migrations {
		if _, err := db.Exec(m); err != nil {
			log.Error("数据库迁移失败", "err", err, "sql", m)
			return err
		}
	}

	// 初始化默认管理员（从配置读取）
	if err := initDefaultAdmin(db); err != nil {
		return err
	}

	// 初始化默认 AI 模型列表
	if err := initDefaultAIModels(db); err != nil {
		return err
	}

	// 初始化默认角色预算
	if err := initDefaultRoleBudgets(db); err != nil {
		return err
	}

	// 清理无效的 entrypoint 级别数据
	if err := cleanupEntrypointData(db); err != nil {
		log.Warn("清理 entrypoint 数据失败", "err", err)
		// 不中断启动，只记录警告
	}

	log.Info("数据库迁移完成")
	return nil
}

// initDefaultAdmin 初始化默认管理员用户
// 从 config.GlobalConfig.Admin 读取配置
// 如果用户已存在则跳过
func initDefaultAdmin(db *sql.DB) error {
	adminCfg := config.GlobalConfig.Admin
	if adminCfg.Username == "" || adminCfg.Password == "" {
		log.Info("未配置默认管理员，跳过创建")
		return nil
	}

	// 检查用户是否已存在
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users WHERE username = ?", adminCfg.Username).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		log.Info("管理员用户已存在，跳过创建", "username", adminCfg.Username)
		return nil
	}

	// 加密密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(adminCfg.Password), bcrypt.DefaultCost)
	if err != nil {
		log.Error("密码加密失败", "err", err)
		return err
	}

	// 插入默认管理员
	now := time.Now().Format(time.RFC3339)
	_, err = db.Exec(`
		INSERT INTO users (username, password_hash, display_name, role, status, created_at, updated_at)
		VALUES (?, ?, ?, 3, 1, ?, ?)`,
		adminCfg.Username, string(hashedPassword), adminCfg.DisplayName, now, now)
	if err != nil {
		log.Error("创建默认管理员失败", "err", err)
		return err
	}

	log.Info("已创建默认管理员", "username", adminCfg.Username)
	return nil
}

// initDefaultAIModels 初始化默认 AI 模型列表
func initDefaultAIModels(db *sql.DB) error {
	// 检查是否已有数据
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM ai_provider_models").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil // 已有数据，跳过
	}

	now := time.Now().Format(time.RFC3339)
	models := []struct {
		provider      string
		model         string
		displayName   string
		isDefault     int
		sortOrder     int
		contextWindow int // tokens
	}{
		// Gemini 2.5 系列
		{"gemini", "gemini-2.5-flash", "Gemini 2.5 Flash", 1, 1, 1048576},
		{"gemini", "gemini-2.5-flash-lite", "Gemini 2.5 Flash Lite", 0, 2, 1048576},
		{"gemini", "gemini-2.5-pro", "Gemini 2.5 Pro", 0, 3, 1048576},
		// OpenAI
		{"openai", "gpt-4o", "GPT-4o", 1, 1, 128000},
		{"openai", "gpt-4o-mini", "GPT-4o Mini", 0, 2, 128000},
		{"openai", "gpt-4-turbo", "GPT-4 Turbo", 0, 3, 128000},
		{"openai", "gpt-4", "GPT-4", 0, 4, 8192},
		{"openai", "o1", "o1", 0, 5, 200000},
		{"openai", "o1-mini", "o1 Mini", 0, 6, 128000},
		// Anthropic
		{"anthropic", "claude-sonnet-4-20250514", "Claude Sonnet 4", 1, 1, 200000},
		{"anthropic", "claude-opus-4-5-20251101", "Claude Opus 4.5", 0, 2, 200000},
		{"anthropic", "claude-3-5-sonnet-20241022", "Claude 3.5 Sonnet", 0, 3, 200000},
		{"anthropic", "claude-3-5-haiku-20241022", "Claude 3.5 Haiku", 0, 4, 200000},
		{"anthropic", "claude-3-opus-20240229", "Claude 3 Opus", 0, 5, 200000},
		{"anthropic", "claude-3-haiku-20240307", "Claude 3 Haiku", 0, 6, 200000},
		// Ollama (本地部署)
		{"ollama", "qwen2.5:14b", "Qwen 2.5 14B", 1, 1, 32768},
		{"ollama", "qwen2.5:7b", "Qwen 2.5 7B", 0, 2, 32768},
		{"ollama", "qwen2.5:32b", "Qwen 2.5 32B", 0, 3, 32768},
		{"ollama", "llama3.1:8b", "Llama 3.1 8B", 0, 4, 131072},
		{"ollama", "deepseek-r1:14b", "DeepSeek R1 14B", 0, 5, 65536},
	}

	stmt, err := db.Prepare(`INSERT INTO ai_provider_models (provider, model, display_name, is_default, sort_order, context_window, created_at) VALUES (?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, m := range models {
		if _, err := stmt.Exec(m.provider, m.model, m.displayName, m.isDefault, m.sortOrder, m.contextWindow, now); err != nil {
			log.Warn("插入默认模型失败", "err", err)
		}
	}

	log.Info("已初始化默认 AI 模型列表")
	return nil
}

// initDefaultRoleBudgets 初始化默认角色预算（种子数据）
func initDefaultRoleBudgets(db *sql.DB) error {
	var count int
	if err := db.QueryRow("SELECT COUNT(*) FROM ai_role_budget").Scan(&count); err != nil {
		return err
	}
	if count > 0 {
		return nil
	}

	now := time.Now().Format(time.RFC3339)
	budgets := []struct {
		role                    string
		dailyInputTokenLimit    int
		dailyOutputTokenLimit   int
		dailyCallLimit          int
		monthlyInputTokenLimit  int
		monthlyOutputTokenLimit int
		monthlyCallLimit        int
		autoTriggerMinSeverity  string
	}{
		{"background", 400000, 100000, 50, 4000000, 1000000, 500, "low"},
		{"chat", 800000, 200000, 100, 8000000, 2000000, 1000, "off"},
		{"analysis", 1600000, 400000, 20, 8000000, 2000000, 100, "high"},
	}

	stmt, err := db.Prepare(`INSERT INTO ai_role_budget
		(role, daily_input_token_limit, daily_output_token_limit, daily_call_limit,
		 monthly_input_token_limit, monthly_output_token_limit, monthly_call_limit,
		 auto_trigger_min_severity, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, b := range budgets {
		if _, err := stmt.Exec(b.role,
			b.dailyInputTokenLimit, b.dailyOutputTokenLimit, b.dailyCallLimit,
			b.monthlyInputTokenLimit, b.monthlyOutputTokenLimit, b.monthlyCallLimit,
			b.autoTriggerMinSeverity, now); err != nil {
			log.Warn("插入默认角色预算失败", "role", b.role, "err", err)
		}
	}

	log.Info("已初始化默认角色预算")
	return nil
}

// cleanupEntrypointData 清理无效的 entrypoint 级别 SLO 数据
func cleanupEntrypointData(db *sql.DB) error {
	result, err := db.Exec(`DELETE FROM slo_targets WHERE host LIKE ?`, "%@entrypoint%")
	if err != nil {
		log.Error("清理 slo_targets entrypoint 数据失败", "err", err)
		return err
	}
	if rowsAffected, _ := result.RowsAffected(); rowsAffected > 0 {
		log.Info("已清理 slo_targets 表 entrypoint 数据", "rows", rowsAffected)
	}
	return nil
}
