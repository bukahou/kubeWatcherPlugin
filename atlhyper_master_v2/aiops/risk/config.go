// atlhyper_master_v2/aiops/risk/config.go
// 风险评分权重配置
package risk

// MetricChannel 指标通道类型
type MetricChannel int

const (
	ChannelStatistical   MetricChannel = iota // 统计通道 (EMA 连续指标)
	ChannelDeterministic                      // 确定性通道 (确定性直注指标)
	ChannelBoth                               // 双通道 (同时参与统计和确定性)
)

// MetricConfig 指标配置 (权重 + 通道)
type MetricConfig struct {
	Weight  float64
	Channel MetricChannel
}

// RiskConfig 风险评分配置
type RiskConfig struct {
	// Stage 1: 局部风险 (按实体类型 → 指标名 → 配置)
	MetricConfigs map[string]map[string]MetricConfig

	// Stage 2: 时序参数
	TemporalHalfLife float64 // τ (秒)，默认 300 (5 分钟)

	// Stage 3: 传播参数
	SelfWeight float64 // α，默认 0.6 (自身 60%，传播 40%)

	// ClusterRisk 聚合权重
	ClusterWeightMax    float64 // w1，默认 0.5
	ClusterWeightSLO    float64 // w2，默认 0.3
	ClusterWeightGrowth float64 // w3，默认 0.2
}

// DefaultRiskConfig 返回默认配置
func DefaultRiskConfig() *RiskConfig {
	return &RiskConfig{
		MetricConfigs: map[string]map[string]MetricConfig{
			"service": {
				// Basic: SLO 指标
				"error_rate":   {Weight: 0.10, Channel: ChannelStatistical},
				"avg_latency":  {Weight: 0.05, Channel: ChannelStatistical},
				"request_rate": {Weight: 0.05, Channel: ChannelStatistical},
				// Enhanced: APM 统计指标
				"apm_error_rate":  {Weight: 0.15, Channel: ChannelStatistical},
				"apm_p99_latency": {Weight: 0.10, Channel: ChannelStatistical},
				"apm_rps":         {Weight: 0.05, Channel: ChannelStatistical},
				// Enhanced: Log 统计指标
				"log_error_count": {Weight: 0.05, Channel: ChannelStatistical},
				"log_warn_count":  {Weight: 0.05, Channel: ChannelStatistical},
				// Enhanced: 确定性异常（阈值直注，绕过冷启动）
				"apm_high_error_rate":  {Weight: 0.20, Channel: ChannelDeterministic},
				"apm_high_p99_latency": {Weight: 0.15, Channel: ChannelDeterministic},
				"deployment_impact":    {Weight: 0.05, Channel: ChannelDeterministic},
			},
			"pod": {
				"restart_count":          {Weight: 0.20, Channel: ChannelBoth},
				"is_running":             {Weight: 0.10, Channel: ChannelBoth},
				"not_ready_containers":   {Weight: 0.20, Channel: ChannelBoth},
				"max_container_restarts": {Weight: 0.10, Channel: ChannelBoth},
				"container_anomaly":      {Weight: 0.25, Channel: ChannelDeterministic},
				"critical_event":         {Weight: 0.15, Channel: ChannelDeterministic},
				"deployment_impact":      {Weight: 0.25, Channel: ChannelDeterministic},
			},
			"node": {
				// Basic: K8s Metrics Server
				"cpu_usage":       {Weight: 0.20, Channel: ChannelStatistical},
				"memory_usage":    {Weight: 0.20, Channel: ChannelStatistical},
				"memory_pressure": {Weight: 0.10, Channel: ChannelDeterministic},
				"disk_pressure":   {Weight: 0.05, Channel: ChannelDeterministic},
				"pid_pressure":    {Weight: 0.05, Channel: ChannelDeterministic},
				// Enhanced: OTel 深度指标
				"disk_usage": {Weight: 0.15, Channel: ChannelStatistical},
				"psi_cpu":    {Weight: 0.10, Channel: ChannelStatistical},
				"psi_memory": {Weight: 0.10, Channel: ChannelStatistical},
				"psi_io":     {Weight: 0.05, Channel: ChannelStatistical},
			},
			"ingress": {
				"error_rate":  {Weight: 0.50, Channel: ChannelStatistical},
				"avg_latency": {Weight: 0.50, Channel: ChannelStatistical},
			},
			"logs": {
				"log_error_count": {Weight: 0.40, Channel: ChannelStatistical},
				"log_warn_count":  {Weight: 0.20, Channel: ChannelStatistical},
				"log_error_spike": {Weight: 0.40, Channel: ChannelDeterministic},
			},
		},
		TemporalHalfLife:    300,
		SelfWeight:          0.6,
		ClusterWeightMax:    0.5,
		ClusterWeightSLO:    0.3,
		ClusterWeightGrowth: 0.2,
	}
}

// GetWeights 获取指定实体类型的指标权重 (兼容方法，从 MetricConfig 提取 Weight)
func (c *RiskConfig) GetWeights(entityType string) map[string]float64 {
	configs := c.GetMetricConfigs(entityType)
	weights := make(map[string]float64, len(configs))
	for name, cfg := range configs {
		weights[name] = cfg.Weight
	}
	return weights
}

// GetMetricConfigs 获取指定实体类型的指标配置
func (c *RiskConfig) GetMetricConfigs(entityType string) map[string]MetricConfig {
	if m, ok := c.MetricConfigs[entityType]; ok {
		return m
	}
	return map[string]MetricConfig{}
}
