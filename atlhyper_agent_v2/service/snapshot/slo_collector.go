// Package snapshot SLO 多窗口数据采集
//
// 本文件实现 SLO 多窗口缓存逻辑：
//   - 3 个窗口 (1d/7d/30d) 各有独立的缓存 TTL
//   - 顺序执行查询以避免 ClickHouse 资源竞争
//   - 每个窗口独立 3 分钟超时
package snapshot

import (
	"context"
	"time"

	"AtlHyper/model_v3/slo"
)

// sloWindowConfig 窗口配置
type sloWindowConfig struct {
	key      string
	since    time.Duration
	bucket   time.Duration
	cacheTTL time.Duration
}

// sloWindowConfigs 窗口配置列表
var sloWindowConfigs = []sloWindowConfig{
	{"1d", 24 * time.Hour, time.Hour, 5 * time.Minute},
	{"7d", 7 * 24 * time.Hour, 6 * time.Hour, 30 * time.Minute},
	{"30d", 30 * 24 * time.Hour, 24 * time.Hour, 2 * time.Hour},
}

// collectSLOWindows 采集多窗口 SLO 数据
func (s *snapshotService) collectSLOWindows(ctx context.Context, now time.Time) map[string]*slo.SLOWindowData {
	if s.sloWindowCaches == nil {
		s.sloWindowCaches = make(map[string]*sloWindowCache)
	}

	result := make(map[string]*slo.SLOWindowData, len(sloWindowConfigs))

	// 顺序执行每个窗口查询（不再并发）
	// 原因: Linkerd gauge 查询在大窗口下需要 30-60 秒，
	// 3 个窗口并发会导致 ClickHouse 资源竞争 → 全部超时。
	// 窗口有独立 TTL 缓存，大部分调用直接命中缓存，不会阻塞。
	for _, wc := range sloWindowConfigs {
		// 检查缓存
		if cache, ok := s.sloWindowCaches[wc.key]; ok && now.Sub(cache.fetchedAt) < wc.cacheTTL {
			result[wc.key] = cache.data
			continue
		}

		data := s.fetchSLOWindow(wc)
		if data != nil {
			s.sloWindowCaches[wc.key] = &sloWindowCache{data: data, fetchedAt: now}
			result[wc.key] = data
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// fetchSLOWindow 获取单个窗口的 SLO 数据（独立 3 分钟超时）
func (s *snapshotService) fetchSLOWindow(wc sloWindowConfig) *slo.SLOWindowData {
	windowCtx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()

	current, err := s.dashboardRepo.ListIngressSLO(windowCtx, wc.since)
	if err != nil {
		log.Warn("SLO 窗口 current 查询失败", "window", wc.key, "err", err)
		return nil
	}

	previous, _ := s.dashboardRepo.ListIngressSLOPrevious(windowCtx, wc.since)
	history, _ := s.dashboardRepo.GetIngressSLOHistory(windowCtx, wc.since, wc.bucket)

	// 服务网格 SLO 已移除：SLO 只做 ingress 外部视角，
	// 服务间调用质量由 APM 承担（见 slo-ingress-contract-design.md）。
	return &slo.SLOWindowData{
		Current:  current,
		Previous: previous,
		History:  history,
	}
}
