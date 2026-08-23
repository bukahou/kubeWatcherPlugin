// atlhyper_master_v2/config/defaults.go
// 默认值定义
// 所有可用的环境变量都在此处列出，便于快速查阅
package config

// ============================================================
// 时间类型默认值
// ============================================================
var defaultDurations = map[string]string{
	// -------------------- DataHub 配置 --------------------
	"MASTER_DATAHUB_EVENT_RETENTION":    "30m", // Event 保留时间
	"MASTER_DATAHUB_HEARTBEAT_EXPIRE":   "45s", // 心跳过期时间
	"MASTER_DATAHUB_SNAPSHOT_RETENTION": "15m", // OTel 快照时间线保留时间

	// -------------------- 超时配置 --------------------
	"MASTER_TIMEOUT_COMMAND_POLL": "60s", // 长轮询超时
	"MASTER_TIMEOUT_HEARTBEAT":    "45s", // 心跳超时阈值

	// -------------------- Event 持久化 --------------------
	"MASTER_EVENT_CLEANUP_INTERVAL": "1h", // 清理检查间隔

	// -------------------- Event 告警 --------------------
	"MASTER_EVENT_ALERT_INTERVAL": "30s", // 告警检测间隔

	// -------------------- JWT 配置 --------------------
	"MASTER_JWT_TOKEN_EXPIRY": "24h", // Token 有效期

	// -------------------- AI 配置 --------------------
	"MASTER_AI_TOOL_TIMEOUT": "30s", // Tool 执行超时

	// -------------------- SLO 配置 --------------------
	"MASTER_SLO_AGGREGATE_INTERVAL": "1h",    // 聚合间隔
	"MASTER_SLO_CLEANUP_INTERVAL":   "1h",    // 清理间隔
	"MASTER_SLO_RAW_RETENTION":      "48h",   // raw 数据保留时间
	"MASTER_SLO_HOURLY_RETENTION":   "2160h", // hourly 数据保留时间 (90 天)
	"MASTER_SLO_STATUS_RETENTION":   "4320h", // 状态历史保留时间 (180 天)

	// -------------------- 节点指标持久化配置 --------------------
	"MASTER_METRICS_SAMPLE_INTERVAL":  "30s", // 历史数据采样间隔
	"MASTER_METRICS_CLEANUP_INTERVAL": "1h",  // 清理检查间隔
}

// ============================================================
// 整数类型默认值
// ============================================================
var defaultInts = map[string]int{
	// -------------------- 服务器配置 --------------------
	"MASTER_GATEWAY_PORT":  8080, // Gateway 端口（Web/API）
	"MASTER_AGENTSDK_PORT": 8081, // AgentSDK 端口（Agent 数据上报）
	"MASTER_TESTER_PORT":   9080, // Tester 端口（测试服务）

	// -------------------- DataHub 配置 --------------------
	"MASTER_DATAHUB_SNAPSHOT_RETAIN": 10, // 快照保留数量

	// -------------------- Redis 配置 --------------------
	"MASTER_REDIS_DB": 0, // Redis 数据库编号

	// -------------------- 数据库配置 --------------------
	"MASTER_DB_MAX_CONNS": 10, // 最大连接数

	// -------------------- Event 持久化 --------------------
	"MASTER_EVENT_RETENTION_DAYS": 30,     // 保留天数
	"MASTER_EVENT_MAX_COUNT":      100000, // 单集群最大事件数

	// -------------------- 节点指标持久化 --------------------
	"MASTER_METRICS_RETENTION_DAYS": 30, // 历史数据保留天数

	// -------------------- GitHub 配置 --------------------
	"GITHUB_APP_ID": 0, // GitHub App ID
}

// ============================================================
// 字符串类型默认值
// ============================================================
var defaultStrings = map[string]string{
	// -------------------- 日志配置 --------------------
	"MASTER_LOG_LEVEL":  "info", // 日志级别: debug / info / warn / error
	"MASTER_LOG_FORMAT": "text", // 日志格式: text / json

	// -------------------- DataHub 配置 --------------------
	"MASTER_DATAHUB_TYPE": "memory", // DataHub 类型

	// -------------------- Redis 配置 --------------------
	"MASTER_REDIS_ADDR":     "localhost:6379", // Redis 地址
	"MASTER_REDIS_PASSWORD": "",               // Redis 密码

	// -------------------- 数据库配置 --------------------
	"MASTER_DB_TYPE": "sqlite",                                            // 数据库类型
	"MASTER_DB_PATH": "atlhyper_master_v2/database/sqlite/data/master.db", // SQLite 路径
	"MASTER_DB_DSN":  "",                                                  // MySQL/PG 连接串

	// -------------------- JWT 配置 --------------------
	"MASTER_JWT_SECRET": "", // JWT 密钥（必须通过环境变量配置）

	// -------------------- 默认管理员配置 --------------------
	"MASTER_ADMIN_USERNAME":     "", // 管理员用户名（必须通过环境变量配置）
	"MASTER_ADMIN_PASSWORD":     "", // 管理员密码（必须通过环境变量配置）
	"MASTER_ADMIN_DISPLAY_NAME": "", // 管理员显示名称

	// -------------------- AI Seed 配置 --------------------
	"MASTER_AI_SEED_PROVIDER": "", // 种子 Provider 类型（为空则不初始化）
	"MASTER_AI_SEED_NAME":     "", // 种子 Provider 显示名称
	"MASTER_AI_SEED_API_KEY":  "", // 种子 API Key
	"MASTER_AI_SEED_MODEL":    "", // 种子模型名称
	"MASTER_AI_SEED_BASE_URL": "", // 种子自定义 API 地址

	// -------------------- GitHub 配置 --------------------
	"GITHUB_APP_SLUG":         "",                                           // GitHub App URL slug
	"GITHUB_PRIVATE_KEY_PATH": "",                                           // GitHub Private Key PEM 文件路径
	"GITHUB_CALLBACK_URL":     "http://localhost:3000/auth/github/callback", // GitHub App 安装回调 URL
}

// ============================================================
// 布尔类型默认值
// ============================================================
var defaultBools = map[string]bool{
	// -------------------- AI 配置 --------------------
	"MASTER_AI_ENABLED": false, // 是否启用 AI 功能（Web UI 配置）

	// -------------------- Event 告警 --------------------
	"MASTER_EVENT_ALERT_ENABLED": true, // 是否启用事件告警
}
