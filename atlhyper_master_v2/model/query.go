// atlhyper_master_v2/model/query.go
// 查询选项模型
package model

import "time"

// PodQueryOpts Pod 查询选项
type PodQueryOpts struct {
	Namespace string
	NodeName  string
	Phase     string
	Limit     int
	Offset    int
}

// EventQueryOpts Event 查询选项
type EventQueryOpts struct {
	Type   string // Normal / Warning
	Reason string
	Since  time.Time
	Limit  int
	Offset int
}
