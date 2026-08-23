package model

// ListOptions 列表查询选项
//
// 用于 Repository 和 Service 层的资源列表查询
type ListOptions struct {
	// LabelSelector 标签选择器
	// 格式: "key=value,key2=value2" 或 "key in (v1,v2)"
	// 示例: "app=nginx,env=prod"
	LabelSelector string

	// FieldSelector 字段选择器
	// 格式: "field.path=value"
	// 示例: "status.phase=Running", "spec.nodeName=node1"
	FieldSelector string

	// Limit 限制返回数量
	// 0 表示不限制
	Limit int64

	// Namespace 命名空间过滤
	// 空表示所有命名空间
	Namespace string
}

// LogOptions 日志查询选项
//
// 用于获取 Pod 日志
type LogOptions struct {
	// Container 容器名称
	// Pod 有多个容器时必须指定
	Container string

	// TailLines 返回最后 N 行
	// 0 表示返回全部
	TailLines int64

	// SinceSeconds 返回最近 N 秒的日志
	// 0 表示不限制时间
	SinceSeconds int64

	// Timestamps 是否包含时间戳
	Timestamps bool

	// Follow 是否跟踪日志 (流式)
	// 目前不支持
	Follow bool

	// Previous 是否获取之前容器的日志
	// 用于查看重启前的日志
	Previous bool
}

// DeleteOptions 删除选项
type DeleteOptions struct {
	// GracePeriodSeconds 优雅终止时间 (秒)
	// nil 使用默认值，0 表示立即删除
	GracePeriodSeconds *int64

	// Force 是否强制删除
	// 设为 true 时会设置 GracePeriodSeconds=0
	Force bool

	// PropagationPolicy 级联删除策略
	// Orphan: 不删除依赖资源
	// Background: 后台删除依赖资源
	// Foreground: 前台删除依赖资源
	PropagationPolicy string
}
