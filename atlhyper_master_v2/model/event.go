// atlhyper_master_v2/model/event.go
// Event Web API 响应类型（camelCase JSON tag）
package model

// EventLog 事件日志项
type EventLog struct {
	ClusterID string `json:"clusterId"`
	Category  string `json:"category"`
	EventTime string `json:"eventTime"`
	Kind      string `json:"kind"`
	Message   string `json:"message"`
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Node      string `json:"node"`
	Reason    string `json:"reason"`
	Severity  string `json:"severity"`
	Time      string `json:"time"`
}

// EventCards 事件统计卡片
type EventCards struct {
	TotalAlerts     int `json:"totalAlerts"`
	TotalEvents     int `json:"totalEvents"`
	Warning         int `json:"warning"`
	Info            int `json:"info"`
	Error           int `json:"error"`
	CategoriesCount int `json:"categoriesCount"`
	KindsCount      int `json:"kindsCount"`
}

// EventOverview 事件概览
type EventOverview struct {
	Cards EventCards `json:"cards"`
	Rows  []EventLog `json:"rows"`
}

// EventListResponse 事件列表响应
type EventListResponse struct {
	Events []EventLog `json:"events"`
	Total  int        `json:"total"`
	Source string     `json:"source"`
	Limit  int        `json:"limit,omitempty"`
	Offset int        `json:"offset,omitempty"`
}
