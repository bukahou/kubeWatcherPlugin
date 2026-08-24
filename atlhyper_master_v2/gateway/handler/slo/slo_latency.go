// atlhyper_master_v2/gateway/handler/slo_latency.go
// 入口延迟分布 API Handler — OTelSnapshot 直读模式
package slo

import (
	"net/http"
	"sort"

	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/model"
	slomodel "AtlHyper/model_v3/slo"
)

// LatencyDistribution GET /api/v2/slo/domains/latency
// 返回指定域名的延迟分布（优先从 SLOWindows 获取，回退到 SLOIngress）
func (h *SLOHandler) LatencyDistribution(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		clusterID = h.defaultClusterID(r.Context())
	}

	domain := r.URL.Query().Get("domain")
	if domain == "" {
		handler.WriteError(w, http.StatusBadRequest, "domain required")
		return
	}

	timeRange := r.URL.Query().Get("time_range")
	if timeRange == "" {
		timeRange = "1d"
	}

	ctx := r.Context()

	// 获取 OTelSnapshot
	otel, err := h.querySvc.GetOTelSnapshot(ctx, clusterID)
	if err != nil {
		sloLog.Error("获取 OTelSnapshot 失败", "err", err)
		handler.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 获取域名下所有 service_key
	mappings, err := h.querySvc.GetSLORouteMappingsByDomain(ctx, clusterID, domain)
	if err != nil {
		sloLog.Error("获取路由映射失败", "domain", domain, "err", err)
		handler.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 域名 → serviceKey。路由映射表为空时按 DisplayName 从快照反查 ——
	// 直接拿域名当 serviceKey 会查不到任何数据（见 serviceKeysForDomain 的注释）
	var ingressForLookup []slomodel.IngressSLO
	if otel != nil {
		ingressForLookup = otel.SLOIngress
	}
	serviceKeys := serviceKeysForDomain(domain, ingressForLookup, mappings)

	// 优先从 SLOWindows[timeRange] 获取 IngressSLO（含 LatencyBuckets + Methods）
	var ingressList []slomodel.IngressSLO
	if otel != nil && otel.SLOWindows != nil {
		if w, ok := otel.SLOWindows[timeRange]; ok {
			ingressList = w.Current
		}
	}
	// 回退到 5min SLOIngress
	if len(ingressList) == 0 && otel != nil {
		ingressList = otel.SLOIngress
	}

	// 合并所有 service 的 IngressSLO 数据
	var totalRequests int64
	var totalRPS float64
	var weightedP50, weightedP95, weightedP99, weightedAvg float64
	var methods []model.MethodBreakdown
	var statusCodes []model.StatusCodeBreakdown
	var buckets []model.LatencyBucket

	// 聚合 map
	statusMap := make(map[string]int64)
	methodMap := make(map[string]int64)
	bucketMap := make(map[float64]int64) // le → count

	for _, ing := range ingressList {
		if !serviceKeys[ing.ServiceKey] {
			continue
		}

		totalRequests += ing.TotalRequests
		totalRPS += ing.RPS
		weightedP50 += ing.P50Ms * float64(ing.TotalRequests)
		p95 := ing.P95Ms
		if p95 == 0 {
			p95 = ing.P90Ms
		}
		weightedP95 += p95 * float64(ing.TotalRequests)
		weightedP99 += ing.P99Ms * float64(ing.TotalRequests)
		weightedAvg += ing.AvgMs * float64(ing.TotalRequests)

		// 聚合状态码
		for _, sc := range ing.StatusCodes {
			statusMap[sc.Code] += sc.Count
		}

		// 聚合延迟桶
		for _, b := range ing.LatencyBuckets {
			bucketMap[b.LE] += b.Count
		}

		// 聚合方法分布
		for _, m := range ing.Methods {
			methodMap[m.Method] += m.Count
		}
	}

	// 计算加权平均
	var p50, p95, p99, avg int
	if totalRequests > 0 {
		p50 = int(weightedP50 / float64(totalRequests))
		p95 = int(weightedP95 / float64(totalRequests))
		p99 = int(weightedP99 / float64(totalRequests))
		avg = int(weightedAvg / float64(totalRequests))
	}

	// 构建状态码分布（按 code 升序）
	for code, count := range statusMap {
		if count > 0 {
			statusCodes = append(statusCodes, model.StatusCodeBreakdown{Code: code, Count: count})
		}
	}
	sort.Slice(statusCodes, func(i, j int) bool { return statusCodes[i].Code < statusCodes[j].Code })

	// 构建延迟分布桶（按 LE 升序）
	for le, count := range bucketMap {
		buckets = append(buckets, model.LatencyBucket{LE: le, Count: count})
	}
	sort.Slice(buckets, func(i, j int) bool { return buckets[i].LE < buckets[j].LE })

	// 构建方法分布（按方法名排序）
	for method, count := range methodMap {
		methods = append(methods, model.MethodBreakdown{Method: method, Count: count})
	}
	sort.Slice(methods, func(i, j int) bool { return methods[i].Method < methods[j].Method })

	resp := model.LatencyDistributionResponse{
		Domain:        domain,
		TotalRequests: totalRequests,
		P50LatencyMs:  p50,
		P95LatencyMs:  p95,
		P99LatencyMs:  p99,
		AvgLatencyMs:  avg,
		Buckets:       buckets,
		Methods:       methods,
		StatusCodes:   statusCodes,
	}

	handler.WriteJSON(w, http.StatusOK, resp)
}
