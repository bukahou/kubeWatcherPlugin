// atlhyper_master_v2/database/factory.go
// 数据库工厂函数
package database

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	// modernc.org/sqlite —— 纯 Go 实现的 SQLite，不依赖 CGO。
	// 2026-08-29 由 mattn/go-sqlite3（C 实现）换来：CGO 阻塞 Docker 交叉编译，
	// controller 因此每次构建都要在 QEMU 里模拟 arm64，耗时 10 分钟。
	// 驱动名是 "sqlite"（mattn 是 "sqlite3"），DSN 参数语法也不同 ——
	// 见 openSQLite，改动受 factory_test.go 的契约测试守护。
	_ "modernc.org/sqlite"

	"AtlHyper/common/logger"
)

var log = logger.Module("Database")

// Config 数据库配置
type Config struct {
	Type string // sqlite / mysql / postgres
	Path string // SQLite 文件路径
}

// NewDatabase 创建数据库实例
// 打开连接并执行迁移，Repository 通过 repo.Init() 注入
func NewDatabase(cfg Config, dialect Dialect) (*DB, error) {
	var conn *sql.DB
	var err error

	switch cfg.Type {
	case "sqlite":
		conn, err = openSQLite(cfg.Path)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// 执行迁移
	if err := dialect.Migrate(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to migrate database: %w", err)
	}

	log.Info("已连接数据库", "type", cfg.Type)
	return &DB{Conn: conn}, nil
}

// openSQLite 打开 SQLite 数据库连接
func openSQLite(path string) (*sql.DB, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	// modernc 用 _pragma=xxx(value) 语法，而非 mattn 的 _journal_mode=WAL。
	// 写错不会报错，只会静默退回默认（delete 模式 / 无 busy timeout）——
	// factory_test.go 里 TestOpenSQLite_WALEnabled 与 _BusyTimeout 即为此而设。
	db, err := sql.Open("sqlite",
		path+"?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)")
	if err != nil {
		return nil, err
	}

	if err := db.Ping(); err != nil {
		return nil, err
	}

	return db, nil
}
