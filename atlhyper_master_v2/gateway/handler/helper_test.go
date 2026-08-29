package handler

import (
	"net/http/httptest"
	"testing"
)

// ──────────────────────────────────────────────────────────────
// 集群参数提取契约
// ──────────────────────────────────────────────────────────────
//
// 背景（2026-08-29 能力盘点 R1）：aiops 系列端点读 `cluster`，其余端点读
// `cluster_id`。盘点时按惯例用 cluster_id 调 aiops/baseline，得到
// "missing cluster parameter" 而误判为「端点无数据」。
//
// ClusterIDFromQuery 让两种写法都能工作，消除接线时的踩坑点。
// cluster_id 为规范名，cluster 保留向后兼容（既有前端调用仍在用）。

func TestClusterIDFromQuery(t *testing.T) {
	tests := []struct {
		name  string
		query string
		want  string
	}{
		{"规范名 cluster_id", "?cluster_id=requiem", "requiem"},
		{"兼容名 cluster", "?cluster=requiem", "requiem"},
		{"两者都在时以规范名优先", "?cluster_id=aaa&cluster=bbb", "aaa"},
		{"都没有返回空", "?foo=bar", ""},
		{"空值等同缺失，回退到另一个", "?cluster_id=&cluster=requiem", "requiem"},
		{"无查询串", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest("GET", "/api/v2/aiops/baseline"+tt.query, nil)
			if got := ClusterIDFromQuery(r); got != tt.want {
				t.Errorf("ClusterIDFromQuery(%q) = %q, want %q", tt.query, got, tt.want)
			}
		})
	}
}
