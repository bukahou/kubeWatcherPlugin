// observe_freshness.go 信号新鲜度判定
//
// 判定放在 Master 而不是前端：「多久算陈旧」「是没流量还是采集挂了」都是业务判断，
// 前端只该渲染文案和颜色。
package query

import (
	"context"
	"time"

	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/model_v3/cluster"
)

// 各信号的陈旧阈值。
//
// metrics 是 Collector 主动拉取的（15 秒一轮），超过 5 分钟没数据只能是链路出问题；
// traces / logs 由请求触发，凌晨安静一小时完全正常，阈值放宽到 15 分钟，
// 而且超了也只是 idle 而非故障。
const (
	metricsStaleAfter = 5 * time.Minute
	eventStaleAfter   = 15 * time.Minute
)

// GetSignalFreshness 汇报三个信号的数据新鲜度。
//
// 快照缺失或旧版 Agent 未上报时，返回三个 absent 占位而不是报错 ——
// 页面需要知道「查不到」，这本身就是有用的信息。
func (q *QueryService) GetSignalFreshness(ctx context.Context, clusterID string) (*model.FreshnessResponse, error) {
	otel, err := q.GetOTelSnapshot(ctx, clusterID)
	if err != nil {
		return nil, err
	}

	var f *cluster.SignalFreshness
	if otel != nil {
		f = otel.Freshness
	}
	if f == nil {
		return &model.FreshnessResponse{
			Signals: []model.SignalFreshnessItem{
				{Signal: "metrics", Status: model.FreshnessAbsent},
				{Signal: "traces", Status: model.FreshnessAbsent},
				{Signal: "logs", Status: model.FreshnessAbsent},
			},
		}, nil
	}

	now := time.Now()
	// 先判 metrics：它决定了另外两个信号的「超时」该解释为没流量还是采集异常。
	// 它自己传 collectorHealthy=false —— metrics 是拉取式的，它超时没有「没人访问」
	// 这种解释，只能是链路出了问题。
	metrics := buildFreshnessItem("metrics", f.MetricsAt, now, metricsStaleAfter, false)
	collectorHealthy := metrics.Status == model.FreshnessLive

	return &model.FreshnessResponse{
		Signals: []model.SignalFreshnessItem{
			metrics,
			buildFreshnessItem("traces", f.TracesAt, now, eventStaleAfter, collectorHealthy),
			buildFreshnessItem("logs", f.LogsAt, now, eventStaleAfter, collectorHealthy),
		},
		CollectorHealthy: collectorHealthy,
	}, nil
}

// buildFreshnessItem 判定单个信号。
//
// collectorHealthy 决定超时后的措辞：采集链路正常时超时是 idle（没人访问），
// 链路本身有问题时是 stale（该救火了）。
func buildFreshnessItem(signal string, at, now time.Time, staleAfter time.Duration, collectorHealthy bool) model.SignalFreshnessItem {
	if at.IsZero() {
		return model.SignalFreshnessItem{Signal: signal, Status: model.FreshnessAbsent}
	}

	lag := now.Sub(at)
	item := model.SignalFreshnessItem{
		Signal:     signal,
		LastDataAt: at.UTC().Format(time.RFC3339),
		LagSeconds: int64(lag.Seconds()),
		Status:     model.FreshnessLive,
	}
	if lag <= staleAfter {
		return item
	}
	if collectorHealthy {
		item.Status = model.FreshnessIdle
	} else {
		item.Status = model.FreshnessStale
	}
	return item
}
