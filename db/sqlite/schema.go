package sqlite

import (
	"NeuroController/db/utils"
	"log"
)

// ============================================================
// ✅ CreateTables：初始化 SQLite 表结构
// - 使用全局 utils.DB 数据库连接
// - 如表已存在则不会重复创建（IF NOT EXISTS）
// ============================================================
func CreateTables() error {

	// 1️⃣ 创建 event_logs 表（用于记录告警/事件日志）
	_, err := utils.DB.Exec(`
		CREATE TABLE IF NOT EXISTS event_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			category TEXT,         -- 事件来源类别，如 kube-event / apm / custom
			eventTime TEXT,        -- 实际事件发生时间（ISO 8601 字符串）
			kind TEXT,             -- 资源类型，如 Pod / Node / Deployment
			message TEXT,          -- 告警/事件的详细消息
			name TEXT,             -- 对应的资源名称
			namespace TEXT,        -- 所属命名空间
			node TEXT,             -- 所属节点名称（可为空）
			reason TEXT,           -- 原因（如 K8s 事件中的 reason）
			severity TEXT,         -- 严重程度：info / warning / critical
			time TEXT              -- 入库时间戳（记录时间，区别于 eventTime）
		)
	`)
	if err != nil {
		log.Printf("❌ 创建 event_logs 表失败: %v", err)
		return err
	}

	// 2️⃣ 创建 users 表（用于 Web 登录认证）
	_, err = utils.DB.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			username TEXT NOT NULL UNIQUE,      -- 用户名，唯一
			password_hash TEXT NOT NULL,        -- 加密后的密码（bcrypt）
			display_name TEXT,                  -- 展示用名称
			email TEXT,                         -- 邮箱地址（可选）
			role INTEGER NOT NULL,              -- 角色标识（如 1=普通用户，3=管理员）
			created_at TEXT NOT NULL,           -- 创建时间（ISO 字符串）
			last_login TEXT                     -- 最近一次登录时间（可为空）
		)
	`)
	if err != nil {
		log.Printf("❌ 创建 users 表失败: %v", err)
	}
	

	// 3️⃣ 创建用户审计表格。主要记录对集群进行的操作细节
	// username	TEXT	执行操作的用户名
	// role	INTEGER	用户角色（如 1=普通用户，2=运维，3=管理员）
	// action	TEXT	操作内容（如 "cordon_node", "delete_pod" 等）
	// success	BOOLEAN	操作是否成功（true / false）
	// timestamp	TEXT	操作发生时间（ISO8601 格式字符串）
	_, err = utils.DB.Exec(`
		CREATE TABLE IF NOT EXISTS user_audit_logs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			user_id INTEGER NOT NULL,		   -- 操作用户 ID
			username TEXT NOT NULL,            -- 执行操作的用户名
			role INTEGER NOT NULL,             -- 用户角色（如 1=普通用户，2=运维，3=管理员）
			action TEXT NOT NULL,              -- 操作内容（如 "cordon_node", "delete_pod" 等）
			success BOOLEAN NOT NULL,          -- 操作是否成功（true / false）
			timestamp TEXT NOT NULL DEFAULT  (datetime('now', 'localtime'))            -- 操作发生时间（ISO8601 格式字符串）
		)
	`)
	if err != nil {
		log.Printf("❌ 创建 user_audit_logs 表失败: %v", err)
		return err
	}

	return nil
}
