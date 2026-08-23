// Package metrics 定义基础设施指标数据模型
//
// 数据源: ClickHouse otel_metrics_gauge + otel_metrics_sum (Node Exporter)
package metrics

import (
	"time"

	model_v3 "AtlHyper/model_v3"
)

// ============================================================
// NodeMetrics — 节点指标快照（某时刻）
// ============================================================

// NodeMetrics 单个节点的指标快照（从 ClickHouse 查询聚合）
type NodeMetrics struct {
	NodeName  string    `json:"nodeName"`
	NodeIP    string    `json:"nodeIP"`
	Timestamp time.Time `json:"timestamp"`

	CPU         NodeCPU         `json:"cpu"`
	Memory      NodeMemory      `json:"memory"`
	Disks       []NodeDisk      `json:"disks"`
	Networks    []NodeNetwork   `json:"networks"`
	Temperature NodeTemperature `json:"temperature"`

	PSI     NodePSI     `json:"psi"`
	TCP     NodeTCP     `json:"tcp"`
	System  NodeSystem  `json:"system"`
	VMStat  NodeVMStat  `json:"vmstat"`
	Softnet NodeSoftnet `json:"softnet"`

	Kernel string `json:"kernel,omitempty"`
	Uptime int64  `json:"uptime,omitempty"`

	// 硬件健康（Phase 1a）。Hardware 为 nil 表示上报方（旧版 Agent）不采集，
	// Master 判定时按「无数据」处理，不得视为正常。
	HardwareProfile HardwareProfile `json:"hardwareProfile,omitempty"`
	Hardware        *NodeHardware   `json:"hardware,omitempty"`
}

type NodeCPU struct {
	UsagePct  float64   `json:"usagePct"`
	UserPct   float64   `json:"userPct"`
	SystemPct float64   `json:"systemPct"`
	IOWaitPct float64   `json:"iowaitPct"`
	Load1     float64   `json:"load1"`
	Load5     float64   `json:"load5"`
	Load15    float64   `json:"load15"`
	Cores     int       `json:"cores"`
	FreqHz    []float64 `json:"freqHz,omitempty"`    // 各核当前频率（node_cpu_scaling_frequency_hertz）
	FreqMaxHz float64   `json:"freqMaxHz,omitempty"` // 标称最高频率（node_cpu_scaling_frequency_max_hertz）
}

type NodeMemory struct {
	TotalBytes     int64   `json:"totalBytes"`
	AvailableBytes int64   `json:"availableBytes"`
	FreeBytes      int64   `json:"freeBytes"`
	CachedBytes    int64   `json:"cachedBytes"`
	BuffersBytes   int64   `json:"buffersBytes"`
	UsagePct       float64 `json:"usagePct"`
	SwapTotalBytes int64   `json:"swapTotalBytes"`
	SwapFreeBytes  int64   `json:"swapFreeBytes"`
	SwapUsagePct   float64 `json:"swapUsagePct"`
}

type NodeDisk struct {
	Device           string  `json:"device"`
	MountPoint       string  `json:"mountPoint"`
	FSType           string  `json:"fsType"`
	TotalBytes       int64   `json:"totalBytes"`
	AvailBytes       int64   `json:"availBytes"`
	UsagePct         float64 `json:"usagePct"`
	ReadBytesPerSec  float64 `json:"readBytesPerSec"`
	WriteBytesPerSec float64 `json:"writeBytesPerSec"`
	ReadIOPS         float64 `json:"readIOPS"`
	WriteIOPS        float64 `json:"writeIOPS"`
	IOUtilPct        float64 `json:"ioUtilPct"`
	AwaitReadMs      float64 `json:"awaitReadMs"`  // 平均读延迟 = read_time 速率 ÷ reads_completed 速率
	AwaitWriteMs     float64 `json:"awaitWriteMs"` // 平均写延迟
	QueueDepth       float64 `json:"queueDepth"`   // 平均在途请求数 = io_time_weighted 速率
}

type NodeNetwork struct {
	Interface     string  `json:"interface"`
	Up            bool    `json:"up"`
	SpeedBps      int64   `json:"speedBps"`
	MTU           int     `json:"mtu"`
	RxBytesPerSec float64 `json:"rxBytesPerSec"`
	TxBytesPerSec float64 `json:"txBytesPerSec"`
	RxPktPerSec   float64 `json:"rxPktPerSec"`
	TxPktPerSec   float64 `json:"txPktPerSec"`
	RxErrPerSec   float64 `json:"rxErrPerSec"`
	TxErrPerSec   float64 `json:"txErrPerSec"`
	RxDropPerSec  float64 `json:"rxDropPerSec"`
	TxDropPerSec  float64 `json:"txDropPerSec"`
}

type NodeTemperature struct {
	CPUTempC float64      `json:"cpuTempC"`
	CPUMaxC  float64      `json:"cpuMaxC"`
	CPUCritC float64      `json:"cpuCritC"`
	Sensors  []TempSensor `json:"sensors"`
}

type TempSensor struct {
	Chip     string  `json:"chip"`
	ChipName string  `json:"chipName,omitempty"` // node_hwmon_chip_names 的可读名（coretemp / nvme / rp1_adc ...）
	Sensor   string  `json:"sensor"`
	CurrentC float64 `json:"currentC"`
	MaxC     float64 `json:"maxC"`
	CritC    float64 `json:"critC"`
}

type NodePSI struct {
	CPUSomePct float64 `json:"cpuSomePct"`
	MemSomePct float64 `json:"memSomePct"`
	MemFullPct float64 `json:"memFullPct"`
	IOSomePct  float64 `json:"ioSomePct"`
	IOFullPct  float64 `json:"ioFullPct"`
}

type NodeTCP struct {
	CurrEstab   int64 `json:"currEstab"`
	Alloc       int64 `json:"alloc"`
	InUse       int64 `json:"inUse"`
	TimeWait    int64 `json:"timeWait"`
	SocketsUsed int64 `json:"socketsUsed"`
}

type NodeSystem struct {
	ConntrackEntries int64 `json:"conntrackEntries"`
	ConntrackLimit   int64 `json:"conntrackLimit"`
	FilefdAllocated  int64 `json:"filefdAllocated"`
	FilefdMax        int64 `json:"filefdMax"`
	EntropyBits      int64 `json:"entropyBits"`
}

type NodeVMStat struct {
	PgFaultPerSec    float64 `json:"pgFaultPerSec"`
	PgMajFaultPerSec float64 `json:"pgMajFaultPerSec"`
	PswpInPerSec     float64 `json:"pswpInPerSec"`
	PswpOutPerSec    float64 `json:"pswpOutPerSec"`
}

type NodeSoftnet struct {
	DroppedPerSec  float64 `json:"droppedPerSec"`
	SqueezedPerSec float64 `json:"squeezedPerSec"`
}

// ============================================================
// 时序数据（趋势图用）
// ============================================================

type Point struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

type Series struct {
	Metric string            `json:"metric"`
	Labels map[string]string `json:"labels,omitempty"`
	Points []Point           `json:"points"`
}

// Summary 集群节点指标概览
type Summary struct {
	TotalNodes  int     `json:"totalNodes"`
	OnlineNodes int     `json:"onlineNodes"`
	AvgCPUPct   float64 `json:"avgCpuPct"`
	AvgMemPct   float64 `json:"avgMemPct"`
	MaxCPUPct   float64 `json:"maxCpuPct"`
	MaxMemPct   float64 `json:"maxMemPct"`
	MaxCPUTemp  float64 `json:"maxCpuTemp"`
}

// ============================================================
// 辅助方法
// ============================================================

func (m *NodeMetrics) IsHealthy() bool {
	if m.CPU.UsagePct > 90 || m.Memory.UsagePct > 90 {
		return false
	}
	for _, d := range m.Disks {
		if d.UsagePct > 90 {
			return false
		}
	}
	return true
}

func (m *NodeMetrics) GetPrimaryDisk() *NodeDisk {
	for i := range m.Disks {
		if m.Disks[i].MountPoint == "/" {
			return &m.Disks[i]
		}
	}
	if len(m.Disks) > 0 {
		return &m.Disks[0]
	}
	return nil
}

func (m *NodeMetrics) GetPrimaryNetwork() *NodeNetwork {
	for i := range m.Networks {
		if m.Networks[i].Up && m.Networks[i].Interface != "lo" {
			return &m.Networks[i]
		}
	}
	if len(m.Networks) > 0 {
		return &m.Networks[0]
	}
	return nil
}

func (m *NodeMetrics) GetHealthStatus() model_v3.HealthStatus {
	if m.CPU.UsagePct > 95 || m.Memory.UsagePct > 95 {
		return model_v3.HealthStatusCritical
	}
	if m.CPU.UsagePct > 80 || m.Memory.UsagePct > 80 {
		return model_v3.HealthStatusWarning
	}
	for _, d := range m.Disks {
		if d.UsagePct > 95 {
			return model_v3.HealthStatusCritical
		}
		if d.UsagePct > 80 {
			return model_v3.HealthStatusWarning
		}
	}
	return model_v3.HealthStatusHealthy
}
