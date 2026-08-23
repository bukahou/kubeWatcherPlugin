// atlhyper_master_v2/aiops/types.go
// AIOps 引擎共用类型定义
package aiops

import "time"

// ==================== 依赖图类型 ====================

// GraphNode 图节点
type GraphNode struct {
	Key       string            `json:"key"`  // "default/service/api-server"
	Type      string            `json:"type"` // "ingress" | "service" | "pod" | "node"
	Namespace string            `json:"namespace"`
	Name      string            `json:"name"`
	Metadata  map[string]string `json:"metadata,omitempty"`
}

// GraphEdge 图边
type GraphEdge struct {
	From   string  `json:"from"`   // source node key
	To     string  `json:"to"`     // target node key
	Type   string  `json:"type"`   // "routes_to" | "calls" | "runs_on" | "selects"
	Weight float64 `json:"weight"` // 边权重 (默认 1.0)
}

// DependencyGraph 依赖图
type DependencyGraph struct {
	ClusterID string                `json:"clusterId"`
	Nodes     map[string]*GraphNode `json:"nodes"`
	Edges     []*GraphEdge          `json:"edges"`
	UpdatedAt time.Time             `json:"updatedAt"`

	// 内部索引（不序列化）
	adjacency map[string][]string `json:"-"` // 正向邻接表: from -> [to...]
	reverse   map[string][]string `json:"-"` // 反向邻接表: to -> [from...]
}

// NewDependencyGraph 创建空依赖图
func NewDependencyGraph(clusterID string) *DependencyGraph {
	return &DependencyGraph{
		ClusterID: clusterID,
		Nodes:     make(map[string]*GraphNode),
		Edges:     make([]*GraphEdge, 0),
		UpdatedAt: time.Now(),
		adjacency: make(map[string][]string),
		reverse:   make(map[string][]string),
	}
}

// AddNode 添加节点
func (g *DependencyGraph) AddNode(key, typ, namespace, name string, metadata map[string]string) {
	if _, exists := g.Nodes[key]; exists {
		return
	}
	g.Nodes[key] = &GraphNode{
		Key:       key,
		Type:      typ,
		Namespace: namespace,
		Name:      name,
		Metadata:  metadata,
	}
}

// AddEdge 添加边
func (g *DependencyGraph) AddEdge(from, to, typ string, weight float64) {
	g.Edges = append(g.Edges, &GraphEdge{
		From:   from,
		To:     to,
		Type:   typ,
		Weight: weight,
	})
}

// RebuildIndex 重建邻接表索引
func (g *DependencyGraph) RebuildIndex() {
	g.adjacency = make(map[string][]string)
	g.reverse = make(map[string][]string)
	for _, edge := range g.Edges {
		g.adjacency[edge.From] = append(g.adjacency[edge.From], edge.To)
		g.reverse[edge.To] = append(g.reverse[edge.To], edge.From)
	}
}

// Adjacency 返回正向邻接表（供 risk propagation 使用）
func (g *DependencyGraph) Adjacency() map[string][]string {
	return g.adjacency
}

// Reverse 返回反向邻接表（供 risk propagation 使用）
func (g *DependencyGraph) Reverse() map[string][]string {
	return g.reverse
}

// TraceResult 链路追踪结果
type TraceResult struct {
	Nodes []*GraphNode `json:"nodes"`
	Edges []*GraphEdge `json:"edges"`
	Depth int          `json:"depth"`
}

// DiffResult 图变更结果
type DiffResult struct {
	AddedNodes   []string
	RemovedNodes []string
	AddedEdges   []*GraphEdge
	RemovedEdges []*GraphEdge
}

// ==================== 基线类型 ====================

// BaselineState 基线状态（每个实体-指标对）
type BaselineState struct {
	EntityKey       string  `json:"entityKey"`
	MetricName      string  `json:"metricName"`
	EMA             float64 `json:"ema"`
	Variance        float64 `json:"variance"`
	Count           int64   `json:"count"`
	ConsecutiveZero int64   `json:"consecutiveZero"` // 连续零值计数（快速冷启动用）
	UpdatedAt       int64   `json:"updatedAt"`
}

// AnomalyResult 异常检测结果
type AnomalyResult struct {
	EntityKey    string  `json:"entityKey"`
	MetricName   string  `json:"metricName"`
	CurrentValue float64 `json:"currentValue"`
	Baseline     float64 `json:"baseline"`
	Deviation    float64 `json:"deviation"`
	Score        float64 `json:"score"`
	IsAnomaly    bool    `json:"isAnomaly"`
	DetectedAt   int64   `json:"detectedAt"`
}

// EntityBaseline 实体基线汇总（API 响应）
type EntityBaseline struct {
	EntityKey string           `json:"entityKey"`
	States    []*BaselineState `json:"states"`
	Anomalies []*AnomalyResult `json:"anomalies"`
}

// MetricDataPoint 指标数据点
type MetricDataPoint struct {
	EntityKey  string
	MetricName string
	Value      float64
}

// ==================== 风险评分类型 ====================

// EntityRisk 实体风险评分
type EntityRisk struct {
	EntityKey    string  `json:"entityKey"`
	EntityType   string  `json:"entityType"` // "service" | "pod" | "node" | "ingress"
	Namespace    string  `json:"namespace"`
	Name         string  `json:"name"`
	RLocal       float64 `json:"rLocal"`       // Stage 1: 局部风险 [0, 1]
	WTime        float64 `json:"wTime"`        // Stage 2: 时序权重 [0, 1]
	RWeighted    float64 `json:"rWeighted"`    // R_local × W_time
	RFinal       float64 `json:"rFinal"`       // Stage 3: 传播后最终风险 [0, 1]
	RiskLevel    string  `json:"riskLevel"`    // "healthy" | "low" | "medium" | "high" | "critical"
	FirstAnomaly int64   `json:"firstAnomaly"` // 首次异常时间 (Unix, 0 = 无异常)
}

// ClusterRisk 集群整体风险
type ClusterRisk struct {
	ClusterID     string        `json:"clusterId"`
	Risk          float64       `json:"risk"`          // [0, 100]
	Level         string        `json:"level"`         // "healthy" | "low" | "warning" | "critical"
	TopEntities   []*EntityRisk `json:"topEntities"`   // 风险最高的 Top 5 实体
	TotalEntities int           `json:"totalEntities"` // 图中总实体数
	AnomalyCount  int           `json:"anomalyCount"`  // 当前异常实体数
	UpdatedAt     int64         `json:"updatedAt"`
}

// EntityRiskDetail 实体风险详情
type EntityRiskDetail struct {
	EntityRisk
	Metrics     []*AnomalyResult   `json:"metrics"`              // 各指标异常详情
	Propagation []*PropagationPath `json:"propagation"`          // 传播路径
	CausalChain []*CausalEntry     `json:"causalChain"`          // 因果链（按时间排序，向后兼容）
	CausalTree  []*CausalTreeNode  `json:"causalTree,omitempty"` // 因果树（依赖图驱动）
}

// CausalTreeNode 因果树节点（以查询实体为中心，按依赖图展开）
type CausalTreeNode struct {
	EntityKey  string            `json:"entityKey"`
	EntityType string            `json:"entityType"`
	RFinal     float64           `json:"rFinal"`
	EdgeType   string            `json:"edgeType,omitempty"`
	Direction  string            `json:"direction,omitempty"` // "upstream" | "downstream"
	Metrics    []*AnomalyResult  `json:"metrics,omitempty"`
	Children   []*CausalTreeNode `json:"children,omitempty"`
}

// PropagationPath 风险传播路径
type PropagationPath struct {
	From         string  `json:"from"`
	To           string  `json:"to"`
	EdgeType     string  `json:"edgeType"`
	Contribution float64 `json:"contribution"`
}

// CausalEntry 因果链条目
type CausalEntry struct {
	EntityKey  string  `json:"entityKey"`
	MetricName string  `json:"metricName"`
	Deviation  float64 `json:"deviation"`
	DetectedAt int64   `json:"detectedAt"`
}

// RiskLevel 从 R_final 映射到风险等级
func RiskLevel(rFinal float64) string {
	switch {
	case rFinal >= 0.8:
		return "critical"
	case rFinal >= 0.6:
		return "high"
	case rFinal >= 0.4:
		return "medium"
	case rFinal >= 0.2:
		return "low"
	default:
		return "healthy"
	}
}

// ClusterRiskLevel 从 ClusterRisk 映射到等级
func ClusterRiskLevel(risk float64) string {
	switch {
	case risk >= 80:
		return "critical"
	case risk >= 50:
		return "warning"
	case risk >= 20:
		return "low"
	default:
		return "healthy"
	}
}

// ExtractEntityType 从 entityKey 提取实体类型
// key 格式: "namespace/type/name"
func ExtractEntityType(entityKey string) string {
	parts := splitEntityKey(entityKey)
	if len(parts) >= 2 {
		return parts[1]
	}
	return ""
}

// splitEntityKey 分割 entityKey
func splitEntityKey(key string) []string {
	var parts []string
	start := 0
	for i, c := range key {
		if c == '/' {
			parts = append(parts, key[start:i])
			start = i + 1
		}
	}
	parts = append(parts, key[start:])
	return parts
}

// ==================== 常量 ====================

const (
	ColdStartMinCount      = 100   // 前 100 个数据点只学习不告警
	ColdStartZeroFastTrack = 10    // 零值计数器快速通道：连续 10 个零值即可结束冷启动
	DefaultAlpha           = 0.033 // α = 2/(60+1), 窗口 60 个采样点
	AnomalyThreshold       = 3.0   // 3σ 规则
	SigmoidK               = 2.0   // sigmoid 斜率
)

// ==================== 状态机类型 ====================

// EntityState 实体当前状态
type EntityState string

const (
	StateHealthy  EntityState = "healthy"
	StateWarning  EntityState = "warning"
	StateIncident EntityState = "incident"
	StateRecovery EntityState = "recovery"
	StateStable   EntityState = "stable"
)

// StateMachineEntry 状态机条目（每个实体一个）
type StateMachineEntry struct {
	EntityKey         string      `json:"entityKey"`
	CurrentState      EntityState `json:"currentState"`
	IncidentID        string      `json:"incidentId"`
	ConditionMetSince int64       `json:"conditionMetSince"`
	LastRFinal        float64     `json:"lastRFinal"`
	LastEvaluatedAt   int64       `json:"lastEvaluatedAt"`
}

// ==================== 事件类型 ====================

// Incident 事件
type Incident struct {
	ID         string      `json:"id"`
	ClusterID  string      `json:"clusterId"`
	State      EntityState `json:"state"`
	Severity   string      `json:"severity"`
	RootCause  string      `json:"rootCause"`
	PeakRisk   float64     `json:"peakRisk"`
	StartedAt  time.Time   `json:"startedAt"`
	ResolvedAt *time.Time  `json:"resolvedAt"`
	DurationS  int64       `json:"durationS"`
	Recurrence int         `json:"recurrence"`
	Summary    string      `json:"summary"`
	CreatedAt  time.Time   `json:"createdAt"`
}

// IncidentEntity 受影响实体
type IncidentEntity struct {
	IncidentID string  `json:"incidentId"`
	EntityKey  string  `json:"entityKey"`
	EntityType string  `json:"entityType"`
	RLocal     float64 `json:"rLocal"`
	RFinal     float64 `json:"rFinal"`
	Role       string  `json:"role"`
}

// IncidentTimeline 事件时间线条目
type IncidentTimeline struct {
	ID         int64     `json:"id"`
	IncidentID string    `json:"incidentId"`
	Timestamp  time.Time `json:"timestamp"`
	EventType  string    `json:"eventType"`
	EntityKey  string    `json:"entityKey"`
	Detail     string    `json:"detail"`
}

// 时间线事件类型常量
const (
	TimelineAnomalyDetected     = "anomaly_detected"
	TimelineStateChange         = "state_change"
	TimelineMetricSpike         = "metric_spike"
	TimelineRootCauseIdentified = "root_cause_identified"
	TimelineRecoveryStarted     = "recovery_started"
	TimelineRecurrence          = "recurrence"
)

// IncidentDetail 事件详情（API 响应）
type IncidentDetail struct {
	Incident
	Entities []*IncidentEntity   `json:"entities"`
	Timeline []*IncidentTimeline `json:"timeline"`
}

// IncidentStats 事件统计
type IncidentStats struct {
	TotalIncidents  int              `json:"totalIncidents"`
	ActiveIncidents int              `json:"activeIncidents"`
	MTTR            float64          `json:"mttr"`
	RecurrenceRate  float64          `json:"recurrenceRate"`
	BySeverity      map[string]int   `json:"bySeverity"`
	ByState         map[string]int   `json:"byState"`
	TopRootCauses   []RootCauseCount `json:"topRootCauses"`
}

// RootCauseCount 根因统计
type RootCauseCount struct {
	EntityKey string `json:"entityKey"`
	Count     int    `json:"count"`
}

// IncidentPattern 历史事件模式
type IncidentPattern struct {
	EntityKey      string      `json:"entityKey"`
	PatternCount   int         `json:"patternCount"`
	AvgDuration    float64     `json:"avgDuration"`
	LastOccurrence time.Time   `json:"lastOccurrence"`
	CommonMetrics  []string    `json:"commonMetrics"`
	Incidents      []*Incident `json:"incidents"`
}

// IncidentQueryOpts 事件查询选项
type IncidentQueryOpts struct {
	ClusterID string
	State     string
	Severity  string
	From      time.Time
	To        time.Time
	Limit     int
	Offset    int
}

// SeverityFromRisk 从 R_final 映射严重度
func SeverityFromRisk(rFinal float64) string {
	switch {
	case rFinal >= 0.9:
		return "critical"
	case rFinal >= 0.7:
		return "high"
	case rFinal >= 0.5:
		return "medium"
	default:
		return "low"
	}
}
