// Package concentrator 本地时序聚合器（类 Datadog Concentrator）
//
// 维护最近 1 小时的降采样数据，每次快照时输出预聚合时序。
// 采用 1 分钟粒度环形缓冲，Agent 每 10s 采集一次，
// 同一分钟内多次写入取最新值（GAUGE 语义）。
package concentrator

import (
	"sync"
	"time"

	"AtlHyper/model_v3/apm"
	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/metrics"
	"AtlHyper/model_v3/slo"
)

const ringCapacity = 60 // 60 分钟 = 1 小时

// Concentrator 本地时序聚合器
type Concentrator struct {
	nodeRings map[string]*nodeMetricsRing
	sloRings  map[string]*sloMetricsRing
	apmRings  map[string]*apmMetricsRing
	mu        sync.RWMutex
}

// NewConcentrator 创建 Concentrator
func NewConcentrator() TimeSeriesAggregator {
	return &Concentrator{
		nodeRings: make(map[string]*nodeMetricsRing),
		sloRings:  make(map[string]*sloMetricsRing),
		apmRings:  make(map[string]*apmMetricsRing),
	}
}

// Ingest 摄入当前 OTel 快照数据，更新时序环
func (c *Concentrator) Ingest(
	nodes []metrics.NodeMetrics,
	sloIngress []slo.IngressSLO,
	apmServices []apm.APMService,
	ts time.Time,
) {
	c.mu.Lock()
	defer c.mu.Unlock()

	minute := alignMinute(ts)
	truncated := ts.Truncate(time.Minute)

	// 节点指标（25 字段）
	for i := range nodes {
		node := &nodes[i]
		ring, ok := c.nodeRings[node.NodeName]
		if !ok {
			ring = &nodeMetricsRing{}
			c.nodeRings[node.NodeName] = ring
		}

		pt := cluster.NodeMetricsPoint{
			Timestamp: truncated,
			// CPU
			CPUPct:    node.CPU.UsagePct,
			UserPct:   node.CPU.UserPct,
			SystemPct: node.CPU.SystemPct,
			IOWaitPct: node.CPU.IOWaitPct,
			Load1:     node.CPU.Load1,
			Load5:     node.CPU.Load5,
			Load15:    node.CPU.Load15,
			// Memory
			MemPct:       node.Memory.UsagePct,
			SwapUsagePct: node.Memory.SwapUsagePct,
			// Temperature
			CPUTempC: node.Temperature.CPUTempC,
			// PSI
			CPUSomePct: node.PSI.CPUSomePct,
			MemSomePct: node.PSI.MemSomePct,
			IOSomePct:  node.PSI.IOSomePct,
			// TCP
			TCPEstab:    node.TCP.CurrEstab,
			SocketsUsed: node.TCP.SocketsUsed,
		}

		// Disk（主磁盘）
		if d := node.GetPrimaryDisk(); d != nil {
			pt.DiskPct = d.UsagePct
			pt.DiskReadBps = d.ReadBytesPerSec
			pt.DiskWriteBps = d.WriteBytesPerSec
			pt.DiskIOUtilPct = d.IOUtilPct
		}

		// Network（聚合所有活跃非 lo 接口）
		for _, n := range node.Networks {
			if n.Up && n.Interface != "lo" {
				pt.NetRxBps += n.RxBytesPerSec
				pt.NetTxBps += n.TxBytesPerSec
				pt.NetRxPktSec += n.RxPktPerSec
				pt.NetTxPktSec += n.TxPktPerSec
			}
		}

		ring.put(minute, pt)
	}

	// SLO Ingress（6 字段）
	for i := range sloIngress {
		svc := &sloIngress[i]
		ring, ok := c.sloRings[svc.ServiceKey]
		if !ok {
			ring = &sloMetricsRing{}
			c.sloRings[svc.ServiceKey] = ring
		}
		ring.put(minute, cluster.SLOTimePoint{
			Timestamp:   truncated,
			RPS:         svc.RPS,
			SuccessRate: svc.SuccessRate,
			P50Ms:       svc.P50Ms,
			P99Ms:       svc.P99Ms,
			ErrorRate:   svc.ErrorRate,
		})
	}

	// APM Services（6 字段）
	for i := range apmServices {
		svc := &apmServices[i]
		ring, ok := c.apmRings[svc.Name]
		if !ok {
			ring = &apmMetricsRing{namespace: svc.Namespace}
			c.apmRings[svc.Name] = ring
		}
		ring.put(minute, cluster.APMTimePoint{
			Timestamp:   truncated,
			RPS:         svc.RPS,
			SuccessRate: svc.SuccessRate,
			AvgMs:       svc.AvgDurationMs,
			P99Ms:       svc.P99Ms,
			ErrorCount:  svc.ErrorCount,
		})
	}
}

// FlushNodeSeries 输出所有节点的预聚合时序
func (c *Concentrator) FlushNodeSeries() []cluster.NodeMetricsTimeSeries {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]cluster.NodeMetricsTimeSeries, 0, len(c.nodeRings))
	now := alignMinute(time.Now())
	for name, ring := range c.nodeRings {
		points := ring.flush(now)
		if len(points) > 0 {
			result = append(result, cluster.NodeMetricsTimeSeries{
				NodeName: name,
				Points:   points,
			})
		}
	}
	return result
}

// FlushSLOSeries 输出所有服务的预聚合 SLO 时序
func (c *Concentrator) FlushSLOSeries() []cluster.SLOServiceTimeSeries {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]cluster.SLOServiceTimeSeries, 0, len(c.sloRings))
	now := alignMinute(time.Now())
	for name, ring := range c.sloRings {
		points := ring.flush(now)
		if len(points) > 0 {
			result = append(result, cluster.SLOServiceTimeSeries{
				ServiceName: name,
				Points:      points,
			})
		}
	}
	return result
}

// FlushAPMSeries 输出所有服务的预聚合 APM 时序
func (c *Concentrator) FlushAPMSeries() []cluster.APMServiceTimeSeries {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]cluster.APMServiceTimeSeries, 0, len(c.apmRings))
	now := alignMinute(time.Now())
	for name, ring := range c.apmRings {
		points := ring.flush(now)
		if len(points) > 0 {
			result = append(result, cluster.APMServiceTimeSeries{
				ServiceName: name,
				Namespace:   ring.namespace,
				Points:      points,
			})
		}
	}
	return result
}

// alignMinute 截断到分钟边界（Unix 分钟数）
func alignMinute(ts time.Time) int64 {
	return ts.Unix() / 60
}

// ================================================================
// nodeMetricsRing — 节点指标环形缓冲
// ================================================================

type nodeMetricsRing struct {
	points  [ringCapacity]cluster.NodeMetricsPoint
	minutes [ringCapacity]int64 // 对应分钟时间戳
}

func (r *nodeMetricsRing) put(minute int64, pt cluster.NodeMetricsPoint) {
	idx := int(minute % ringCapacity)
	r.minutes[idx] = minute
	r.points[idx] = pt
}

func (r *nodeMetricsRing) flush(nowMinute int64) []cluster.NodeMetricsPoint {
	result := make([]cluster.NodeMetricsPoint, 0, ringCapacity)
	for i := 0; i < ringCapacity; i++ {
		// 从最旧到最新
		m := nowMinute - int64(ringCapacity-1) + int64(i)
		idx := int(m % ringCapacity)
		if idx < 0 {
			idx += ringCapacity
		}
		if r.minutes[idx] == m {
			result = append(result, r.points[idx])
		}
	}
	return result
}

// ================================================================
// sloMetricsRing — SLO 指标环形缓冲
// ================================================================

type sloMetricsRing struct {
	points  [ringCapacity]cluster.SLOTimePoint
	minutes [ringCapacity]int64
}

func (r *sloMetricsRing) put(minute int64, pt cluster.SLOTimePoint) {
	idx := int(minute % ringCapacity)
	r.minutes[idx] = minute
	r.points[idx] = pt
}

func (r *sloMetricsRing) flush(nowMinute int64) []cluster.SLOTimePoint {
	result := make([]cluster.SLOTimePoint, 0, ringCapacity)
	for i := 0; i < ringCapacity; i++ {
		m := nowMinute - int64(ringCapacity-1) + int64(i)
		idx := int(m % ringCapacity)
		if idx < 0 {
			idx += ringCapacity
		}
		if r.minutes[idx] == m {
			result = append(result, r.points[idx])
		}
	}
	return result
}

// ================================================================
// apmMetricsRing — APM 指标环形缓冲
// ================================================================

type apmMetricsRing struct {
	namespace string
	points    [ringCapacity]cluster.APMTimePoint
	minutes   [ringCapacity]int64
}

func (r *apmMetricsRing) put(minute int64, pt cluster.APMTimePoint) {
	idx := int(minute % ringCapacity)
	r.minutes[idx] = minute
	r.points[idx] = pt
}

func (r *apmMetricsRing) flush(nowMinute int64) []cluster.APMTimePoint {
	result := make([]cluster.APMTimePoint, 0, ringCapacity)
	for i := 0; i < ringCapacity; i++ {
		m := nowMinute - int64(ringCapacity-1) + int64(i)
		idx := int(m % ringCapacity)
		if idx < 0 {
			idx += ringCapacity
		}
		if r.minutes[idx] == m {
			result = append(result, r.points[idx])
		}
	}
	return result
}
