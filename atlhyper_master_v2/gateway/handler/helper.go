// atlhyper_master_v2/gateway/handler/helper.go
// Handler 公共辅助函数
package handler

import (
	"encoding/json"
	"net/http"
)

// WriteJSON 写入 JSON 响应（导出供子包使用）
func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// WriteError 写入错误响应（导出供子包使用）
func WriteError(w http.ResponseWriter, status int, message string) {
	WriteJSON(w, status, map[string]string{"error": message})
}

// writeJSON 包内便捷别名
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	WriteJSON(w, status, data)
}

// writeError 包内便捷别名
func writeError(w http.ResponseWriter, status int, message string) {
	WriteError(w, status, message)
}

// ClusterIDFromQuery 提取集群 ID，兼容两种参数名。
//
// `cluster_id` 是规范名（绝大多数端点使用）；`cluster` 是 aiops 系列的历史
// 写法，保留以兼容已有前端调用。两者都在时以规范名优先。
//
// 统一走此函数后，调用方不必记忆某个端点用哪种写法 —— 历史上这个不一致
// 导致过误判（按 cluster_id 调 aiops 端点报 missing，被当成端点无数据）。
func ClusterIDFromQuery(r *http.Request) string {
	if v := r.URL.Query().Get("cluster_id"); v != "" {
		return v
	}
	return r.URL.Query().Get("cluster")
}
