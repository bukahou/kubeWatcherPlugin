// atlhyper_agent_v2/config/defaults.go
// 默认值定义
// 所有可用的环境变量都在此处列出，便于快速查阅
package config

// ============================================================
// 时间类型默认值
// ============================================================
var defaultDurations = map[string]string{
	// -------------------- 调度器配置 --------------------
	"AGENT_SNAPSHOT_INTERVAL":     "10s",   // 快照采集间隔
	"AGENT_COMMAND_POLL_INTERVAL": "100ms", // 指令轮询间隔（Dashboard 端点走快照直读后，Command 仅用于 Detail 查询，缩短以降低延迟）
	"AGENT_HEARTBEAT_INTERVAL":    "15s",   // 心跳发送间隔
	"AGENT_OTEL_CACHE_TTL":        "10s",   // OTel 概览缓存 TTL（与快照间隔一致）

	// -------------------- 超时配置 --------------------
	"AGENT_TIMEOUT_HTTP_CLIENT":      "90s", // HTTP 客户端超时 (需 > Master 长轮询超时 60s + 网络开销)
	"AGENT_TIMEOUT_SNAPSHOT_COLLECT": "30s", // 快照采集操作超时
	"AGENT_TIMEOUT_COMMAND_POLL":     "60s", // 指令轮询操作超时 (长轮询)
	"AGENT_TIMEOUT_HEARTBEAT":        "10s", // 心跳操作超时

	// -------------------- ClickHouse 配置 --------------------
	"AGENT_CLICKHOUSE_TIMEOUT": "10s", // ClickHouse 查询超时
}

// ============================================================
// 字符串类型默认值
// ============================================================
var defaultStrings = map[string]string{
	// -------------------- 日志配置 --------------------
	"AGENT_LOG_LEVEL":  "info", // 日志级别: debug / info / warn / error
	"AGENT_LOG_FORMAT": "text", // 日志格式: text / json

	// -------------------- Agent 基础配置 --------------------
	"AGENT_CLUSTER_ID": "", // 集群唯一标识，空则自动获取集群 UID

	// -------------------- Master 通信 --------------------
	"AGENT_MASTER_URL": "http://localhost:8081", // Master AgentSDK 端口（非 Gateway 端口）

	// -------------------- Kubernetes 配置 --------------------
	"AGENT_KUBECONFIG": "", // kubeconfig 文件路径，空则使用 InCluster 模式

	// -------------------- ClickHouse 配置（可选，不配置则纯 K8s 模式） --------------------
	"AGENT_CLICKHOUSE_ENDPOINT": "",         // ClickHouse 地址，为空则不连接（纯 K8s 模式，无 OTel 数据）
	"AGENT_CLICKHOUSE_DATABASE": "atlhyper", // ClickHouse 数据库名
}
