// correlation.go 相关性分析模型
//
// 回答「慢/错的请求，和正常请求相比，什么属性显著超标」。
// 思路对齐 Elastic APM 的 Correlations（significant terms：找出在
// 异常集合中 "uncommonly common" 的属性值），实现用 ClickHouse 条件聚合。
package apm

// Correlation 分析模式
const (
	CorrelationModeLatency = "latency" // 前景 = 耗时 > 该操作 P95 的入口 span
	CorrelationModeFailure = "failure" // 前景 = StatusCode = Error 的入口 span
)

// CorrelationItem 单个属性值的相关性
type CorrelationItem struct {
	Field   string  `json:"field"`   // 归一化字段名（如 user_agent.family）
	Value   string  `json:"value"`   // 属性值；"(none)" = 字段缺失（缺失本身可能相关）
	FgCount int64   `json:"fgCount"` // 前景集中出现次数
	BgCount int64   `json:"bgCount"` // 背景集（全部）中出现次数
	FgRatio float64 `json:"fgRatio"` // 前景占比 0-1
	BgRatio float64 `json:"bgRatio"` // 背景占比 0-1
	Lift    float64 `json:"lift"`    // fgRatio / bgRatio（提升度）
	Score   float64 `json:"score"`   // fgRatio × ln(lift)：覆盖率 × 提升度，防低频值刷分
	Impact  string  `json:"impact"`  // high / medium / low
}

// Impact 分级
const (
	CorrelationImpactHigh   = "high"
	CorrelationImpactMedium = "medium"
	CorrelationImpactLow    = "low"
)

// CorrelationResult 一次相关性分析的结果
type CorrelationResult struct {
	Mode            string            `json:"mode"`
	ForegroundCount int64             `json:"foregroundCount"`
	BackgroundCount int64             `json:"backgroundCount"`
	// LowSample 前景样本 < 5：结果仅供参考，前端必须标注 —— 2 条失败请求
	// 里 100% 是 iPhone 不构成统计显著，但作为线索仍有价值
	LowSample   bool              `json:"lowSample"`
	ThresholdMs float64           `json:"thresholdMs,omitempty"` // latency 模式的 P95 阈值
	Items       []CorrelationItem `json:"items"`
}
