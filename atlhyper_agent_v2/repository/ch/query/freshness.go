// freshness.go 各信号最近一条数据的时间
//
// 页面上「没有流量」和「采集挂了」都表现为空白。三张表各取一次 max(时间列)
// 就能区分：metrics 是 Collector 主动拉取的，只要节点活着就有数据；
// traces / logs 由请求触发。metrics 还在流动而 traces 停了 = 没人访问；
// metrics 也停了 = 采集链路出问题。
package query

import (
	"context"
	"time"

	"AtlHyper/atlhyper_agent_v2/repository"
	"AtlHyper/atlhyper_agent_v2/sdk"
	"AtlHyper/model_v3/cluster"
)

type freshnessRepository struct {
	client sdk.ClickHouseClient
}

// NewFreshnessQueryRepository 创建信号新鲜度查询仓库
func NewFreshnessQueryRepository(client sdk.ClickHouseClient) repository.FreshnessQueryRepository {
	return &freshnessRepository{client: client}
}

// 各表的时间列名不同：metrics 用 TimeUnix，traces / logs 用 Timestamp
var freshnessSources = []struct {
	table  string
	column string
}{
	{"otel_metrics_gauge", "TimeUnix"},
	{"otel_traces", "Timestamp"},
	{"otel_logs", "Timestamp"},
}

// GetSignalFreshness 取三个信号最近一条数据的时间。
//
// 单个信号查询失败时保持零值 —— Master 会把零值判为 absent，
// 这比编一个假时间戳好：查不到本身就是要显示给用户看的信息。
func (r *freshnessRepository) GetSignalFreshness(ctx context.Context) (*cluster.SignalFreshness, error) {
	out := &cluster.SignalFreshness{}
	targets := []*time.Time{&out.MetricsAt, &out.TracesAt, &out.LogsAt}

	for i, src := range freshnessSources {
		// 只扫最近 24 小时：全表 max() 在千万行级别上代价过高，
		// 而超过 24 小时没数据与「刚好 24 小时」在判定上是同一个结论（早就该报警了）。
		query := "SELECT max(" + src.column + ") FROM " + src.table +
			" WHERE " + src.column + " > now() - INTERVAL 24 HOUR"
		var ts time.Time
		if err := r.client.QueryRow(ctx, query).Scan(&ts); err != nil {
			continue
		}
		*targets[i] = ts
	}
	return out, nil
}
