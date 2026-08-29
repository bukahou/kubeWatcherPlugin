package query

import (
	"context"
	"database/sql"
	"errors"
	"slices"
	"testing"

	"AtlHyper/model_v3/metrics"

	_ "modernc.org/sqlite"
)

// ──────────────────────────────────────────────────────────────
// 采集失败必须显式声明，不得静默留下零值
// ──────────────────────────────────────────────────────────────
//
// 背景（2026-08-29 压测缺陷 ①）：9 路 fill* 查询里只有 fillSystemInfo
// 上抛错误，其余 8 路遇错静默 return —— 结构体保持零值，前端把 0 当
// 真实测量渲染成「CPU 0.0%」，用户据此误判「树莓派宕机了」。
// 同类事故 2026-08-24 已发生过一次（见 metrics.go 内注释）。
//
// 契约：查询失败的 section 必须出现在 NodeMetrics.Unavailable 里；
// 部分失败不得丢弃整个节点。

// failingCHClient 所有查询都失败的 ClickHouse 客户端。
// QueryRow 无法直接构造带错误的 *sql.Row，用内存 SQLite 返回一个
// Scan 时必然报错的 Row（空结果集 → ErrNoRows）。
type failingCHClient struct{ db *sql.DB }

func newFailingCHClient(t *testing.T) *failingCHClient {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return &failingCHClient{db: db}
}

func (f *failingCHClient) Query(ctx context.Context, q string, args ...any) (*sql.Rows, error) {
	return nil, errors.New("simulated: context deadline exceeded")
}
func (f *failingCHClient) QueryRow(ctx context.Context, q string, args ...any) *sql.Row {
	return f.db.QueryRow("SELECT 1 WHERE 1=0") // Scan → ErrNoRows
}
func (f *failingCHClient) Ping(ctx context.Context) error { return nil }
func (f *failingCHClient) Close() error                   { return nil }

func TestBuildNodeMetrics_AllQueriesFail_DeclaresUnavailable(t *testing.T) {
	r := &metricsRepository{client: newFailingCHClient(t)}

	nm, err := r.buildNodeMetrics(context.Background(), "10.0.0.1", "test-node")
	// 部分/全部查询失败不是致命错误：节点必须返回，缺失通过 Unavailable 声明
	if err != nil {
		t.Fatalf("查询失败不应使整个节点构建报错（会导致节点被丢弃）: %v", err)
	}
	if nm == nil {
		t.Fatal("nm 不应为 nil")
	}

	// 核心 section 必须全部声明为不可用
	for _, sec := range []string{
		metrics.SectionCPU, metrics.SectionMemory,
		metrics.SectionDisks, metrics.SectionNetworks,
		metrics.SectionTemperature,
	} {
		if !slices.Contains(nm.Unavailable, sec) {
			t.Errorf("section %q 查询已失败，但未出现在 Unavailable=%v —— 零值会被当成真实测量", sec, nm.Unavailable)
		}
	}
}

func TestBuildNodeMetrics_Unavailable_Deterministic(t *testing.T) {
	r := &metricsRepository{client: newFailingCHClient(t)}
	a, _ := r.buildNodeMetrics(context.Background(), "10.0.0.1", "n1")
	b, _ := r.buildNodeMetrics(context.Background(), "10.0.0.1", "n1")
	if !slices.Equal(a.Unavailable, b.Unavailable) {
		t.Errorf("Unavailable 顺序不确定（并发写入未排序）: %v vs %v", a.Unavailable, b.Unavailable)
	}
}
