package database

import (
	"path/filepath"
	"testing"
)

// ──────────────────────────────────────────────────────────────
// SQLite 驱动契约测试
// ──────────────────────────────────────────────────────────────
//
// 背景（2026-08-29）：为消除 CGO 依赖（阻塞 Docker 交叉编译，controller
// 构建因此需 10 分钟全程 QEMU 模拟），驱动由 mattn/go-sqlite3（C 实现）
// 换为 modernc.org/sqlite（纯 Go）。换驱动前该层零测试，本文件即为此补。
//
// 这些用例约束的是**行为契约**而非某个驱动的实现细节 ——
// 将来若再换驱动（或升级版本），它们仍应全绿。

// TestOpenSQLite_CreatesParentDir 打开时自动建父目录（emptyDir 首次启动依赖此行为）
func TestOpenSQLite_CreatesParentDir(t *testing.T) {
	path := filepath.Join(t.TempDir(), "nested", "deep", "atlhyper.db")
	db, err := openSQLite(path)
	if err != nil {
		t.Fatalf("openSQLite: %v", err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		t.Errorf("Ping: %v", err)
	}
}

// TestOpenSQLite_WALEnabled DSN 中的 journal_mode=WAL 必须真正生效。
// WAL 让读写不互相阻塞 —— Master 在写审计的同时要服务查询，这是必需的。
// ⚠️ 两个驱动的 DSN 参数语法不同（mattn 用 _journal_mode，modernc 用
// _pragma=journal_mode(WAL)），此测试确保换驱动时不会静默退回 delete 模式。
func TestOpenSQLite_WALEnabled(t *testing.T) {
	db, err := openSQLite(filepath.Join(t.TempDir(), "wal.db"))
	if err != nil {
		t.Fatalf("openSQLite: %v", err)
	}
	defer db.Close()

	var mode string
	if err := db.QueryRow("PRAGMA journal_mode").Scan(&mode); err != nil {
		t.Fatalf("query journal_mode: %v", err)
	}
	if mode != "wal" {
		t.Errorf("journal_mode = %q, want wal —— DSN 参数语法可能未随驱动更新", mode)
	}
}

// TestOpenSQLite_BusyTimeout busy_timeout 必须生效，否则并发写会立刻返回
// SQLITE_BUSY 而不是等待重试。
func TestOpenSQLite_BusyTimeout(t *testing.T) {
	db, err := openSQLite(filepath.Join(t.TempDir(), "busy.db"))
	if err != nil {
		t.Fatalf("openSQLite: %v", err)
	}
	defer db.Close()

	var timeout int
	if err := db.QueryRow("PRAGMA busy_timeout").Scan(&timeout); err != nil {
		t.Fatalf("query busy_timeout: %v", err)
	}
	if timeout < 5000 {
		t.Errorf("busy_timeout = %d, want >= 5000 —— DSN 参数语法可能未随驱动更新", timeout)
	}
}

// TestOpenSQLite_CRUD 基本读写与事务，覆盖 Master 实际用到的类型
func TestOpenSQLite_CRUD(t *testing.T) {
	db, err := openSQLite(filepath.Join(t.TempDir(), "crud.db"))
	if err != nil {
		t.Fatalf("openSQLite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE t (
		id INTEGER PRIMARY KEY AUTOINCREMENT,
		name TEXT NOT NULL,
		score REAL,
		enabled INTEGER NOT NULL DEFAULT 1,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`); err != nil {
		t.Fatalf("create table: %v", err)
	}

	res, err := db.Exec("INSERT INTO t (name, score) VALUES (?, ?)", "alpha", 1.5)
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	// LastInsertId 是 Master 各 repo 的常用返回值，驱动必须支持
	id, err := res.LastInsertId()
	if err != nil || id != 1 {
		t.Fatalf("LastInsertId = %d, err = %v", id, err)
	}

	var name string
	var score float64
	var enabled bool
	if err := db.QueryRow("SELECT name, score, enabled FROM t WHERE id = ?", id).
		Scan(&name, &score, &enabled); err != nil {
		t.Fatalf("select: %v", err)
	}
	if name != "alpha" || score != 1.5 || !enabled {
		t.Errorf("got (%q, %v, %v)", name, score, enabled)
	}

	// 事务回滚
	tx, err := db.Begin()
	if err != nil {
		t.Fatalf("begin: %v", err)
	}
	if _, err := tx.Exec("INSERT INTO t (name) VALUES (?)", "rollback-me"); err != nil {
		t.Fatalf("tx insert: %v", err)
	}
	if err := tx.Rollback(); err != nil {
		t.Fatalf("rollback: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT count(*) FROM t").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 1 {
		t.Errorf("回滚后行数 = %d, want 1", count)
	}
}

// TestOpenSQLite_ConcurrentWrite WAL + busy_timeout 下并发写不应报 SQLITE_BUSY。
// Master 有多个 goroutine 同时写（审计、事件持久化、AIOps），这是真实场景。
func TestOpenSQLite_ConcurrentWrite(t *testing.T) {
	db, err := openSQLite(filepath.Join(t.TempDir(), "conc.db"))
	if err != nil {
		t.Fatalf("openSQLite: %v", err)
	}
	defer db.Close()

	if _, err := db.Exec("CREATE TABLE c (v INTEGER)"); err != nil {
		t.Fatalf("create: %v", err)
	}

	const writers, perWriter = 4, 25
	errCh := make(chan error, writers)
	for w := 0; w < writers; w++ {
		go func() {
			for i := 0; i < perWriter; i++ {
				if _, err := db.Exec("INSERT INTO c (v) VALUES (?)", i); err != nil {
					errCh <- err
					return
				}
			}
			errCh <- nil
		}()
	}
	for i := 0; i < writers; i++ {
		if err := <-errCh; err != nil {
			t.Fatalf("并发写失败（WAL/busy_timeout 可能未生效）: %v", err)
		}
	}

	var count int
	if err := db.QueryRow("SELECT count(*) FROM c").Scan(&count); err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != writers*perWriter {
		t.Errorf("写入行数 = %d, want %d", count, writers*perWriter)
	}
}
