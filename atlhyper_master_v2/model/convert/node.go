// atlhyper_master_v2/model/convert/node.go
// cluster.Node → model.NodeItem / model.NodeDetail 转换函数
// 包含 K8s CPU/Memory 单位转换
package convert

import (
	"strconv"

	"AtlHyper/atlhyper_master_v2/model"
	model_v3 "AtlHyper/model_v3"
	"AtlHyper/model_v3/cluster"
)

// NodeItem 转换为列表项（扁平，单位已转换）
func NodeItem(src *cluster.Node) model.NodeItem {
	return model.NodeItem{
		Name:         src.Summary.Name,
		Ready:        src.Summary.Ready == "True",
		InternalIP:   src.Addresses.InternalIP,
		OSImage:      src.Info.OSImage,
		Architecture: src.Info.Architecture,
		CPUCores:     cpuToFloat(src.Capacity.CPU),
		MemoryGiB:    memToGiB(src.Capacity.Memory),
		Schedulable:  src.Summary.Schedulable,
	}
}

// NodeItems 转换多个 Node 为列表项
func NodeItems(src []cluster.Node) []model.NodeItem {
	if src == nil {
		return []model.NodeItem{}
	}
	result := make([]model.NodeItem, len(src))
	for i := range src {
		result[i] = NodeItem(&src[i])
	}
	return result
}

// NodeDetail 转换为详情（扁平，单位已转换）
func NodeDetail(src *cluster.Node) model.NodeDetail {
	d := model.NodeDetail{
		Name:        src.Summary.Name,
		Roles:       src.Summary.Roles,
		Ready:       src.Summary.Ready == "True",
		Schedulable: src.Summary.Schedulable,
		Age:         src.Summary.Age,
		CreatedAt:   formatTime(src.Summary.CreationTime),

		Hostname:     src.Addresses.Hostname,
		InternalIP:   src.Addresses.InternalIP,
		ExternalIP:   src.Addresses.ExternalIP,
		OSImage:      src.Info.OSImage,
		OS:           src.Info.OperatingSystem,
		Architecture: src.Info.Architecture,
		Kernel:       src.Info.KernelVersion,
		CRI:          src.Info.ContainerRuntimeVersion,
		Kubelet:      src.Info.KubeletVersion,
		KubeProxy:    src.Info.KubeProxyVersion,

		CPUCapacityCores:    cpuToFloat(src.Capacity.CPU),
		CPUAllocatableCores: cpuToFloat(src.Allocatable.CPU),
		MemCapacityGiB:      memToGiB(src.Capacity.Memory),
		MemAllocatableGiB:   memToGiB(src.Allocatable.Memory),
		PodsCapacity:        parseInt(src.Capacity.Pods),
		PodsAllocatable:     parseInt(src.Allocatable.Pods),
		EphemeralStorageGiB: memToGiB(src.Capacity.EphemeralStorage),

		PodCIDRs:   src.Spec.PodCIDRs,
		ProviderID: src.Spec.ProviderID,

		Labels:  src.Labels,
		Badges:  src.Summary.Badges,
		Reason:  src.Summary.Reason,
		Message: src.Summary.Message,
	}

	// Metrics
	if m := src.Metrics; m != nil {
		d.CPUUsageCores = cpuToFloat(m.CPU.Usage)
		d.CPUUtilPct = m.CPU.UtilPct
		d.MemUsageGiB = memToGiB(m.Memory.Usage)
		d.MemUtilPct = m.Memory.UtilPct
		d.PodsUsed = m.Pods.Used
		d.PodsUtilPct = m.Pods.UtilPct
		d.PressureMemory = m.Pressure.MemoryPressure
		d.PressureDisk = m.Pressure.DiskPressure
		d.PressurePID = m.Pressure.PIDPressure
		d.NetworkUnavailable = m.Pressure.NetworkUnavailable
	}

	// Conditions
	for _, c := range src.Conditions {
		d.Conditions = append(d.Conditions, model.NodeConditionResponse{
			Type:      c.Type,
			Status:    c.Status,
			Reason:    c.Reason,
			Message:   c.Message,
			Heartbeat: formatTime(c.LastHeartbeatTime),
			ChangedAt: formatTime(c.LastTransitionTime),
		})
	}

	// Taints
	for _, t := range src.Taints {
		d.Taints = append(d.Taints, model.NodeTaintResponse{
			Key:    t.Key,
			Value:  t.Value,
			Effect: t.Effect,
		})
	}

	return d
}

// cpuToFloat 将 K8s CPU 字符串转换为核心数（float64）
// "4" → 4.0, "500m" → 0.5, "4000m" → 4.0, "123456789n" → 0.123
func cpuToFloat(s string) float64 {
	return float64(model_v3.ParseCPU(s)) / 1000.0
}

// memToGiB 将 K8s Memory 字符串转换为 GiB（float64）
// "8Gi" → 8.0, "16384Mi" → 16.0, "1073741824" → 1.0
func memToGiB(s string) float64 {
	bytes := model_v3.ParseMemory(s)
	if bytes == 0 {
		return 0
	}
	return float64(bytes) / (1024 * 1024 * 1024)
}

// parseInt 解析整数字符串
func parseInt(s string) int {
	v, _ := strconv.Atoi(s)
	return v
}
