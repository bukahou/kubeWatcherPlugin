// Package snapshot OTel 数据采集
//
// 本文件实现 OTel 快照的缓存与聚合逻辑：
//   - 标量摘要（TTL=5min）：TotalServices / RPS / CPU / Mem 等慢变化指标
//   - Dashboard 列表（TTL=30s）：Services / Topology / Logs 等需要新鲜度的数据
//   - Concentrator 时序摄入与输出
package snapshot

import (
	"context"
	"sync"
	"time"

	"AtlHyper/atlhyper_agent_v2/config"
	"AtlHyper/model_v3/cluster"
)

// getOTelSnapshot 获取 OTel 快照（分离缓存 TTL）
//
// 标量摘要（变化慢）使用 5min TTL，Dashboard 列表（需要新鲜度）使用 30s TTL。
// Concentrator 在每次 Dashboard 数据刷新时摄入数据并输出预聚合时序。
func (s *snapshotService) getOTelSnapshot(ctx context.Context) *cluster.OTelSnapshot {
	summaryTTL := config.GlobalConfig.Scheduler.OTelCacheTTL
	if summaryTTL <= 0 {
		summaryTTL = 5 * time.Minute
	}
	dashboardTTL := 30 * time.Second

	snapshot := &cluster.OTelSnapshot{}
	now := time.Now()

	var wg sync.WaitGroup
	var mu sync.Mutex
	var hasError bool

	setError := func() {
		mu.Lock()
		hasError = true
		mu.Unlock()
	}

	// ===== 标量摘要（TTL = 5min，变化慢） =====

	summaryFresh := s.otelCache != nil && now.Sub(s.otelCacheTime) < summaryTTL
	if summaryFresh {
		// 复用缓存中的标量
		snapshot.TotalServices = s.otelCache.TotalServices
		snapshot.HealthyServices = s.otelCache.HealthyServices
		snapshot.TotalRPS = s.otelCache.TotalRPS
		snapshot.AvgSuccessRate = s.otelCache.AvgSuccessRate
		snapshot.AvgP99Ms = s.otelCache.AvgP99Ms
		snapshot.IngressServices = s.otelCache.IngressServices
		snapshot.IngressAvgRPS = s.otelCache.IngressAvgRPS
		snapshot.MonitoredNodes = s.otelCache.MonitoredNodes
		snapshot.AvgCPUPct = s.otelCache.AvgCPUPct
		snapshot.AvgMemPct = s.otelCache.AvgMemPct
		snapshot.MaxCPUPct = s.otelCache.MaxCPUPct
		snapshot.MaxMemPct = s.otelCache.MaxMemPct
	} else if s.otelSummaryRepo != nil {
		wg.Add(3)

		go func() {
			defer wg.Done()
			totalSvc, healthySvc, totalRPS, avgSuccRate, avgP99, err := s.otelSummaryRepo.GetAPMSummary(ctx)
			if err != nil {
				log.Warn("OTel APM 概览查询失败", "err", err)
				setError()
				return
			}
			mu.Lock()
			snapshot.TotalServices = totalSvc
			snapshot.HealthyServices = healthySvc
			snapshot.TotalRPS = totalRPS
			snapshot.AvgSuccessRate = avgSuccRate
			snapshot.AvgP99Ms = avgP99
			mu.Unlock()
		}()

		go func() {
			defer wg.Done()
			ingressSvc, ingressRPS, err := s.otelSummaryRepo.GetSLOSummary(ctx)
			if err != nil {
				log.Warn("OTel SLO 概览查询失败", "err", err)
				setError()
				return
			}
			mu.Lock()
			snapshot.IngressServices = ingressSvc
			snapshot.IngressAvgRPS = ingressRPS
			mu.Unlock()
		}()

		go func() {
			defer wg.Done()
			nodes, avgCPU, avgMem, maxCPU, maxMem, err := s.otelSummaryRepo.GetMetricsSummary(ctx)
			if err != nil {
				log.Warn("OTel Metrics 概览查询失败", "err", err)
				setError()
				return
			}
			mu.Lock()
			snapshot.MonitoredNodes = nodes
			snapshot.AvgCPUPct = avgCPU
			snapshot.AvgMemPct = avgMem
			snapshot.MaxCPUPct = maxCPU
			snapshot.MaxMemPct = maxMem
			mu.Unlock()
		}()
	}

	// ===== Dashboard 列表（TTL = 30s，需要新鲜度给 Concentrator） =====

	dashboardFresh := s.otelDashboardCache != nil && now.Sub(s.otelDashboardCacheTime) < dashboardTTL
	if dashboardFresh {
		// 复用缓存中的 Dashboard 列表
		cached := s.otelDashboardCache.snapshot
		snapshot.MetricsSummary = cached.MetricsSummary
		snapshot.MetricsNodes = cached.MetricsNodes
		snapshot.APMServices = cached.APMServices
		snapshot.APMTopology = cached.APMTopology
		snapshot.SLOSummary = cached.SLOSummary
		snapshot.SLOIngress = cached.SLOIngress
		snapshot.APMOperations = cached.APMOperations
		snapshot.RecentTraces = cached.RecentTraces
		// RecentLogs 不再缓存到快照（日志走 Command → ClickHouse 按需查询）
		snapshot.LogsSummary = cached.LogsSummary
	} else if s.dashboardRepo != nil {
		defaultSince := 5 * time.Minute

		wg.Add(9) // RecentLogs 已移除(走 Command 按需查询); SLOServices/SLOEdges 已移除(SLO 去 mesh)

		go func() {
			defer wg.Done()
			result, err := s.dashboardRepo.ListAPMOperations(ctx)
			if err != nil {
				log.Warn("Dashboard APMOperations 查询失败", "err", err)
				return
			}
			mu.Lock()
			snapshot.APMOperations = result
			mu.Unlock()
		}()

		go func() {
			defer wg.Done()
			result, err := s.dashboardRepo.GetMetricsSummary(ctx)
			if err != nil {
				log.Warn("Dashboard MetricsSummary 查询失败", "err", err)
				return
			}
			mu.Lock()
			snapshot.MetricsSummary = result
			mu.Unlock()
		}()

		go func() {
			defer wg.Done()
			result, err := s.dashboardRepo.ListAllNodeMetrics(ctx)
			if err != nil {
				log.Warn("Dashboard MetricsNodes 查询失败", "err", err)
				return
			}
			mu.Lock()
			snapshot.MetricsNodes = result
			mu.Unlock()
		}()

		go func() {
			defer wg.Done()
			result, err := s.dashboardRepo.ListAPMServices(ctx)
			if err != nil {
				log.Warn("Dashboard APMServices 查询失败", "err", err)
				return
			}
			mu.Lock()
			snapshot.APMServices = result
			mu.Unlock()
		}()

		go func() {
			defer wg.Done()
			result, err := s.dashboardRepo.GetAPMTopology(ctx)
			if err != nil {
				log.Warn("Dashboard APMTopology 查询失败", "err", err)
				return
			}
			mu.Lock()
			snapshot.APMTopology = result
			mu.Unlock()
		}()

		go func() {
			defer wg.Done()
			result, err := s.dashboardRepo.GetSLOSummary(ctx)
			if err != nil {
				log.Warn("Dashboard SLOSummary 查询失败", "err", err)
				return
			}
			mu.Lock()
			snapshot.SLOSummary = result
			mu.Unlock()
		}()

		go func() {
			defer wg.Done()
			result, err := s.dashboardRepo.ListIngressSLO(ctx, defaultSince)
			if err != nil {
				log.Warn("Dashboard SLOIngress 查询失败", "err", err)
				return
			}
			mu.Lock()
			s.applyRouteHostnames(ctx, result)
			snapshot.SLOIngress = result
			mu.Unlock()
		}()

		// RecentTraces（500 条，用于 Trace 钻入；聚合统计已由 APMOperations 覆盖）
		go func() {
			defer wg.Done()
			traces, err := s.dashboardRepo.ListRecentTraces(ctx, 500)
			if err != nil {
				log.Warn("Dashboard RecentTraces 查询失败", "err", err)
				return
			}
			mu.Lock()
			snapshot.RecentTraces = traces
			mu.Unlock()
		}()

		// LogsSummary
		go func() {
			defer wg.Done()
			summary, err := s.dashboardRepo.GetLogsSummary(ctx)
			if err != nil {
				log.Warn("Dashboard LogsSummary 查询失败", "err", err)
				return
			}
			mu.Lock()
			snapshot.LogsSummary = summary
			mu.Unlock()
		}()

		// RecentLogs 已移除 — 日志不缓存到快照内存
		// 前端查询走 Command → Agent → ClickHouse 按需查询（Kibana 模式）
	}

	wg.Wait()

	// 各信号最新数据时间：让页面能区分「没有流量」和「采集挂了」。
	//
	// 必须在下面的 early return 之前采集：它是元信息，与其他查询的成败无关，
	// 而且查询失败时更需要它 —— 那种情况下上报的是旧缓存，用户必须知道数据有多旧。
	if s.dashboardRepo != nil {
		// 独立超时：走到这里时主 ctx 已被前面十几个 OTel 查询耗掉，
		// 实测三张表的 max() 合计约 2 秒，但继承来的 ctx 往往只剩不到 1 秒 ——
		// 结果就是新鲜度永远 deadline exceeded。collectSLOWindows 早先也是这么解决的。
		freshCtx, cancelFresh := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelFresh()

		if f, err := s.dashboardRepo.GetSignalFreshness(freshCtx); err != nil {
			log.Warn("采集信号新鲜度失败", "err", err)
		} else if f == nil {
			log.Warn("信号新鲜度为空 —— 检查 ClickHouse 客户端是否注入")
		} else {
			snapshot.Freshness = f
		}
	}

	// 逐信号回退：本轮取不到的信号沿用上一份已知值，成功的照常更新。
	//
	// 2026-08-30 修正：旧实现是「任一概览查询失败 → 丢弃整份快照」
	// （注释写的是「全部失败」，与代码不符），一路超时就作废另外 8 路
	// 的成功结果。更要命的是 9 路 dashboard 查询失败时连这个保护都不触发 ——
	// 它们只 log.Warn 后 return，留下 nil 字段被写进 30s 缓存，
	// Master 见 nil 返 404，面板空白 30 秒后突然恢复
	// （用户报告的「有时候没数据、有时候又出现」）。
	//
	// 原则同 2026-08-29 的 buildNodeMetrics 修复：部分失败不丢弃全部。
	var restored RestoredSignals
	if s.otelCache != nil {
		restored = restoreMissingFromPrev(snapshot, s.otelCache)
	}
	if hasError {
		// 新鲜度必须反映本轮实测 —— 否则页面拿着旧数据却显示「数据正常」，自相矛盾。
		log.Warn("OTel 概览查询有失败，相关信号已沿用上一份",
			"freshnessAttached", snapshot.Freshness != nil)
	}

	// 更新缓存
	if !summaryFresh {
		s.otelCacheTime = now
	}
	if !dashboardFresh {
		s.otelDashboardCache = &dashboardCacheData{snapshot: snapshot}
		s.otelDashboardCacheTime = now
	}
	s.otelCache = snapshot

	// Concentrator: 摄入当前数据 + 输出预聚合时序
	if s.conc != nil {
		// 回退来的信号传 nil：它们不是本轮新采样点，重复摄入会让
		// 预聚合时序把「查询失败」伪装成「值保持不变」。
		ingestNodes := snapshot.MetricsNodes
		if restored.MetricsNodes {
			ingestNodes = nil
		}
		ingestSLO := snapshot.SLOIngress
		if restored.SLOIngress {
			ingestSLO = nil
		}
		ingestAPM := snapshot.APMServices
		if restored.APMServices {
			ingestAPM = nil
		}
		s.conc.Ingest(ingestNodes, ingestSLO, ingestAPM, now)
		snapshot.NodeMetricsSeries = s.conc.FlushNodeSeries()
		snapshot.SLOTimeSeries = s.conc.FlushSLOSeries()
		snapshot.APMTimeSeries = s.conc.FlushAPMSeries()
	}

	// 多窗口 SLO 数据采集（带独立 TTL 缓存）
	if s.dashboardRepo != nil {
		snapshot.SLOWindows = s.collectSLOWindows(ctx, now)
	}

	return snapshot
}
