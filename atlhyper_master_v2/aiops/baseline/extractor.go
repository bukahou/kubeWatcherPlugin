// atlhyper_master_v2/aiops/baseline/extractor.go
// 从 ClusterSnapshot 和 OTelSnapshot 提取指标数据点
package baseline

import (
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
	"AtlHyper/model_v3/cluster"
)

// ExtractMetrics 从快照和 OTelSnapshot 中提取所有实体指标
func ExtractMetrics(
	clusterID string,
	snap *cluster.ClusterSnapshot,
	otel *cluster.OTelSnapshot,
) []aiops.MetricDataPoint {
	var points []aiops.MetricDataPoint

	// 1. Node 指标（从 K8s Metrics Server: snap.Nodes[].Metrics）
	points = append(points, extractNodeMetrics(snap)...)

	// 2. Pod 指标（从 K8s 快照）
	points = append(points, extractPodMetrics(snap)...)

	// 3. OTel 信号（SLO + APM + Log + Enhanced Node）
	if otel != nil {
		// Basic SLO
		points = append(points, extractServiceMetrics(otel)...)
		points = append(points, extractIngressMetrics(otel)...)
		// Enhanced: APM / Log / 深度 Node（函数实现在 extractor_enhanced.go）
		points = append(points, extractAPMMetrics(otel)...)
		points = append(points, extractLogMetrics(otel)...)
		points = append(points, extractEnhancedNodeMetrics(otel)...)
	}

	return points
}

func extractNodeMetrics(snap *cluster.ClusterSnapshot) []aiops.MetricDataPoint {
	var points []aiops.MetricDataPoint

	// 从 K8s Metrics Server 数据读取（snap.Nodes[].Metrics）
	// 不依赖 OTel/ClickHouse，Agent + Master 即可工作
	for i := range snap.Nodes {
		node := &snap.Nodes[i]
		if node.Metrics == nil {
			continue
		}
		key := aiops.EntityKey("_cluster", "node", node.GetName())

		// CPU/Memory 使用率（来自 K8s Metrics Server）
		if node.Metrics.CPU.UtilPct > 0 {
			points = append(points,
				aiops.MetricDataPoint{EntityKey: key, MetricName: "cpu_usage", Value: node.Metrics.CPU.UtilPct},
			)
		}
		if node.Metrics.Memory.UtilPct > 0 {
			points = append(points,
				aiops.MetricDataPoint{EntityKey: key, MetricName: "memory_usage", Value: node.Metrics.Memory.UtilPct},
			)
		}
	}
	return points
}

func extractPodMetrics(snap *cluster.ClusterSnapshot) []aiops.MetricDataPoint {
	var points []aiops.MetricDataPoint
	for i := range snap.Pods {
		pod := &snap.Pods[i]
		key := aiops.EntityKey(pod.Summary.Namespace, "pod", pod.Summary.Name)
		restarts := float64(pod.Status.Restarts)
		isRunning := 0.0
		if pod.Status.Phase == "Running" {
			isRunning = 1.0
		}

		// 容器级指标
		var notReady float64
		var maxRestarts int32
		for j := range pod.Containers {
			c := &pod.Containers[j]
			if !c.Ready {
				notReady++
			}
			if c.RestartCount > maxRestarts {
				maxRestarts = c.RestartCount
			}
		}

		points = append(points,
			aiops.MetricDataPoint{EntityKey: key, MetricName: "restart_count", Value: restarts},
			aiops.MetricDataPoint{EntityKey: key, MetricName: "is_running", Value: isRunning},
			aiops.MetricDataPoint{EntityKey: key, MetricName: "not_ready_containers", Value: notReady},
			aiops.MetricDataPoint{EntityKey: key, MetricName: "max_container_restarts", Value: float64(maxRestarts)},
		)
	}
	return points
}

func extractServiceMetrics(otel *cluster.OTelSnapshot) []aiops.MetricDataPoint {
	var points []aiops.MetricDataPoint
	return points
}

// ExtractDeterministicAnomalies 从快照中提取确定性异常（绕过 EMA 冷启动）
// 扫描容器状态和关联 Event，对 CrashLoopBackOff/OOMKilled 等确定性异常直接生成 AnomalyResult
func ExtractDeterministicAnomalies(snap *cluster.ClusterSnapshot) []*aiops.AnomalyResult {
	now := time.Now().Unix()
	var results []*aiops.AnomalyResult

	// 路径 B1: 容器状态异常
	results = append(results, extractContainerAnomalies(snap, now)...)

	// 路径 B2: Event 关联异常
	results = append(results, extractEventAnomalies(snap, now)...)

	// 路径 B3: Deployment 影响比例异常
	results = append(results, extractDeploymentImpact(snap, now)...)

	// 路径 B4: Node 压力确定性异常（来自 K8s Node Conditions，不依赖 OTel）
	results = append(results, extractNodePressure(snap, now)...)

	return results
}

// extractNodePressure 从 Node Conditions 提取确定性压力异常
func extractNodePressure(snap *cluster.ClusterSnapshot, now int64) []*aiops.AnomalyResult {
	var results []*aiops.AnomalyResult
	for i := range snap.Nodes {
		node := &snap.Nodes[i]
		if node.Metrics == nil {
			continue
		}
		key := aiops.EntityKey("_cluster", "node", node.GetName())

		if node.Metrics.Pressure.MemoryPressure {
			results = append(results, &aiops.AnomalyResult{
				EntityKey: key, MetricName: "memory_pressure",
				CurrentValue: 1, Baseline: 0, Deviation: 10,
				Score: 0.85, IsAnomaly: true, DetectedAt: now,
			})
		}
		if node.Metrics.Pressure.DiskPressure {
			results = append(results, &aiops.AnomalyResult{
				EntityKey: key, MetricName: "disk_pressure",
				CurrentValue: 1, Baseline: 0, Deviation: 10,
				Score: 0.80, IsAnomaly: true, DetectedAt: now,
			})
		}
		if node.Metrics.Pressure.PIDPressure {
			results = append(results, &aiops.AnomalyResult{
				EntityKey: key, MetricName: "pid_pressure",
				CurrentValue: 1, Baseline: 0, Deviation: 10,
				Score: 0.75, IsAnomaly: true, DetectedAt: now,
			})
		}
	}
	return results
}

// extractContainerAnomalies 从容器状态提取确定性异常
// 每个 Pod 只报告最严重的一个容器异常
func extractContainerAnomalies(snap *cluster.ClusterSnapshot, now int64) []*aiops.AnomalyResult {
	var results []*aiops.AnomalyResult
	for i := range snap.Pods {
		pod := &snap.Pods[i]
		key := aiops.EntityKey(pod.Summary.Namespace, "pod", pod.Summary.Name)

		var worstReason string
		var worstScore float64
		for j := range pod.Containers {
			reason := classifyContainerAnomaly(&pod.Containers[j])
			if reason == "" {
				continue
			}
			score := deterministicScore(reason)
			if score > worstScore {
				worstScore = score
				worstReason = reason
			}
		}

		if worstReason != "" {
			results = append(results, &aiops.AnomalyResult{
				EntityKey:    key,
				MetricName:   "container_anomaly",
				CurrentValue: worstScore,
				Baseline:     0,
				Deviation:    worstScore * 10, // 高偏离度确保触发
				Score:        worstScore,
				IsAnomaly:    true,
				DetectedAt:   now,
			})
		}
	}
	return results
}

// classifyContainerAnomaly 判断容器异常原因
// 返回空字符串表示无异常
func classifyContainerAnomaly(c *cluster.PodContainerDetail) string {
	// waiting 状态异常（最明确的信号）
	if c.State == "waiting" {
		switch c.StateReason {
		case "CrashLoopBackOff", "OOMKilled",
			"ImagePullBackOff", "ErrImagePull",
			"CreateContainerConfigError":
			return c.StateReason
		}
	}

	// terminated 且因 OOMKilled 终止
	if c.State == "terminated" && c.LastTerminationReason == "OOMKilled" {
		return "OOMKilled"
	}

	// running + 近期崩溃：容器刚重启回来，快照恰好抓到 running 瞬间
	// 检查 LastTerminationTime 在 10 分钟内，避免对历史重启持续告警
	if c.State == "running" && c.RestartCount > 0 && c.LastTerminationReason != "" {
		if isRecentTermination(c.LastTerminationTime) {
			if c.LastTerminationReason == "OOMKilled" {
				return "OOMKilled"
			}
			return "RecentCrash"
		}
	}

	// 就绪探针失败：容器 running 但 Ready=false
	if c.State == "running" && !c.Ready {
		return "NotReady"
	}

	return ""
}

// isRecentTermination 判断上次终止时间是否在 10 分钟内
func isRecentTermination(lastTermTime string) bool {
	if lastTermTime == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339, lastTermTime)
	if err != nil {
		return false
	}
	return time.Since(t) < 10*time.Minute
}

// deterministicScore 异常原因 → 固定分数
func deterministicScore(reason string) float64 {
	switch reason {
	case "OOMKilled":
		return 0.95
	case "CrashLoopBackOff":
		return 0.90
	case "CreateContainerConfigError":
		return 0.80
	case "RecentCrash":
		return 0.75
	case "ImagePullBackOff", "ErrImagePull":
		return 0.70
	case "NotReady":
		return 0.60
	default:
		return 0.50
	}
}

// extractEventAnomalies 从 K8s Event 提取关联异常信号
// 筛选 5 分钟内的 Critical Event，关联到已有 Pod 实体
func extractEventAnomalies(snap *cluster.ClusterSnapshot, now int64) []*aiops.AnomalyResult {
	cutoff := time.Unix(now, 0).Add(-5 * time.Minute)

	// 构建 Pod 存在性索引
	podExists := make(map[string]bool, len(snap.Pods))
	for i := range snap.Pods {
		pod := &snap.Pods[i]
		key := aiops.EntityKey(pod.Summary.Namespace, "pod", pod.Summary.Name)
		podExists[key] = true
	}

	// 每个 Pod 只报一次
	reported := make(map[string]bool)
	var results []*aiops.AnomalyResult

	for i := range snap.Events {
		ev := &snap.Events[i]
		if !ev.IsCritical() {
			continue
		}
		if ev.InvolvedObject.Kind != "Pod" {
			continue
		}
		if ev.LastTimestamp.Before(cutoff) {
			continue
		}

		key := aiops.EntityKey(ev.InvolvedObject.Namespace, "pod", ev.InvolvedObject.Name)
		if !podExists[key] || reported[key] {
			continue
		}
		reported[key] = true

		results = append(results, &aiops.AnomalyResult{
			EntityKey:    key,
			MetricName:   "critical_event",
			CurrentValue: 0.85,
			Baseline:     0,
			Deviation:    8.5,
			Score:        0.85,
			IsAnomaly:    true,
			DetectedAt:   now,
		})
	}

	return results
}

// deploymentInfo Deployment 不可用比例信息
type deploymentInfo struct {
	namespace        string
	name             string
	unavailableRatio float64
}

// extractDeploymentImpact Deployment 影响比例异常
// 当 Deployment 不可用比例 >= 50% 时，为该 Deployment 下的不健康 Pod 注入信号
func extractDeploymentImpact(snap *cluster.ClusterSnapshot, now int64) []*aiops.AnomalyResult {
	rsMap := buildRSToDeploymentMap(snap)
	if len(rsMap) == 0 {
		return nil
	}

	var results []*aiops.AnomalyResult
	for i := range snap.Pods {
		pod := &snap.Pods[i]
		if pod.Summary.OwnerKind != "ReplicaSet" {
			continue
		}
		if !isPodUnhealthy(pod) {
			continue
		}

		lookupKey := pod.Summary.Namespace + "/" + pod.Summary.OwnerName
		info, ok := rsMap[lookupKey]
		if !ok {
			continue
		}

		score := deploymentImpactScore(info.unavailableRatio)
		if score == 0 {
			continue
		}

		key := aiops.EntityKey(pod.Summary.Namespace, "pod", pod.Summary.Name)
		results = append(results, &aiops.AnomalyResult{
			EntityKey:    key,
			MetricName:   "deployment_impact",
			CurrentValue: info.unavailableRatio,
			Baseline:     0,
			Deviation:    info.unavailableRatio * 10,
			Score:        score,
			IsAnomaly:    true,
			DetectedAt:   now,
		})
	}
	return results
}

// buildRSToDeploymentMap 构建 "namespace/rsName" -> deploymentInfo 映射
// 通过 snapshot.ReplicaSets 的 OwnerKind/OwnerName 反向关联 Deployment
func buildRSToDeploymentMap(snap *cluster.ClusterSnapshot) map[string]*deploymentInfo {
	// Step 1: 索引 Deployment → deploymentInfo（按 namespace/name）
	depMap := make(map[string]*deploymentInfo, len(snap.Deployments))
	for i := range snap.Deployments {
		dep := &snap.Deployments[i]
		if dep.Summary.Replicas <= 0 {
			continue
		}
		ratio := float64(dep.Summary.Replicas-dep.Summary.Ready) / float64(dep.Summary.Replicas)
		depKey := dep.Summary.Namespace + "/" + dep.Summary.Name
		depMap[depKey] = &deploymentInfo{
			namespace:        dep.Summary.Namespace,
			name:             dep.Summary.Name,
			unavailableRatio: ratio,
		}
	}

	// Step 2: 从 ReplicaSet 的 OwnerKind/OwnerName 反向关联 Deployment
	rsMap := make(map[string]*deploymentInfo)
	for i := range snap.ReplicaSets {
		rs := &snap.ReplicaSets[i]
		if rs.OwnerKind != "Deployment" {
			continue
		}
		parentKey := rs.Namespace + "/" + rs.OwnerName
		if info, ok := depMap[parentKey]; ok {
			rsKey := rs.Namespace + "/" + rs.Name
			rsMap[rsKey] = info
		}
	}
	return rsMap
}

// isPodUnhealthy 至少一个容器 Ready=false
func isPodUnhealthy(pod *cluster.Pod) bool {
	for j := range pod.Containers {
		if !pod.Containers[j].Ready {
			return true
		}
	}
	return false
}

// deploymentImpactScore 不可用比例 → 风险分数
// ratio >= 0.75 → 0.95, >= 0.50 → 0.80, < 0.50 → 0（不注入）
func deploymentImpactScore(ratio float64) float64 {
	switch {
	case ratio >= 0.75:
		return 0.95
	case ratio >= 0.50:
		return 0.80
	default:
		return 0
	}
}

func extractIngressMetrics(otel *cluster.OTelSnapshot) []aiops.MetricDataPoint {
	var points []aiops.MetricDataPoint
	for _, ing := range otel.SLOIngress {
		key := aiops.EntityKey("_cluster", "ingress", ing.ServiceKey)

		errorRate := ing.ErrorRate // 已是 0-100 范围
		points = append(points,
			aiops.MetricDataPoint{EntityKey: key, MetricName: "error_rate", Value: errorRate},
			aiops.MetricDataPoint{EntityKey: key, MetricName: "avg_latency", Value: ing.AvgMs},
		)
	}
	return points
}
