// atlhyper_master_v2/gateway/handler/slo_domains.go
// SLO 域名查询 Handler 方法（Domains / DomainsV2 / DomainDetail / DomainHistory）
package slo

import (
	"context"
	"net/http"
	"time"

	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/slo"
	slomodel "AtlHyper/model_v3/slo"
)

// ==================== 域名列表 ====================

// Domains GET /api/v2/slo/domains
func (h *SLOHandler) Domains(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		clusterID = h.defaultClusterID(r.Context())
	}

	timeRange := r.URL.Query().Get("time_range")
	if timeRange == "" {
		timeRange = "1d"
	}

	ctx := r.Context()

	// 从 OTelSnapshot 获取当前 SLO 数据
	otel, err := h.querySvc.GetOTelSnapshot(ctx, clusterID)
	if err != nil {
		sloLog.Error("获取 OTelSnapshot 失败", "err", err)
		handler.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 获取目标配置
	targets, _ := h.querySvc.GetSLOTargets(ctx, clusterID)
	targetMap := buildTargetMap(targets)

	// 优先从 SLOWindows[timeRange] 获取数据
	var ingressList []slomodel.IngressSLO
	if otel != nil && otel.SLOWindows != nil {
		if w, ok := otel.SLOWindows[timeRange]; ok {
			ingressList = w.Current
		}
	}
	if len(ingressList) == 0 && otel != nil {
		ingressList = otel.SLOIngress
	}

	// 构建响应
	var domains []model.DomainSLO
	var totalAvail, totalBudget, totalRPS float64
	var healthyCount, warningCount, criticalCount int

	for _, ing := range ingressList {
		domain := h.buildDomainFromIngress(ctx, clusterID, ing, timeRange, targetMap)
		domains = append(domains, domain)

		if domain.Current != nil {
			totalAvail += domain.Current.Availability
			totalRPS += domain.Current.RequestsPerSec
		}
		totalBudget += domain.ErrorBudget

		switch domain.Status {
		case statusHealthy:
			healthyCount++
		case statusWarning:
			warningCount++
		case statusCritical:
			criticalCount++
		}
	}

	var avgAvail, avgBudget float64
	if len(domains) > 0 {
		avgAvail = totalAvail / float64(len(domains))
		avgBudget = totalBudget / float64(len(domains))
	}

	if domains == nil {
		domains = []model.DomainSLO{}
	}

	resp := model.SLODomainsResponse{
		Domains: domains,
		Summary: model.SLOSummary{
			TotalDomains:    len(domains),
			HealthyCount:    healthyCount,
			WarningCount:    warningCount,
			CriticalCount:   criticalCount,
			AvgAvailability: avgAvail,
			AvgErrorBudget:  avgBudget,
			TotalRPS:        totalRPS,
		},
	}

	handler.WriteJSON(w, http.StatusOK, resp)
}

// buildDomainFromIngress 从 IngressSLO 构建 DomainSLO
func (h *SLOHandler) buildDomainFromIngress(ctx context.Context, clusterID string, ing slomodel.IngressSLO, timeRange string, targetMap map[string]map[string]model.SLOTargetResponse) model.DomainSLO {
	domain := model.DomainSLO{
		Host:    fallbackDomainName(ing),
		Targets: make(map[string]*model.SLOTargetSpec),
		Status:  statusUnknown,
		Trend:   "stable",
	}

	// 获取路由映射元信息
	mapping, _ := h.querySvc.GetSLORouteMappingByServiceKey(ctx, clusterID, ing.ServiceKey)
	if mapping != nil {
		domain.IngressName = mapping.IngressName
		domain.Namespace = mapping.Namespace
		domain.TLS = mapping.TLS
	}

	// 转换 IngressSLO → SLOMetrics
	domain.Current = ingressToSLOMetrics(ing)

	// 设置目标
	if hostTargets, ok := targetMap[ing.ServiceKey]; ok {
		for tr, t := range hostTargets {
			domain.Targets[tr] = &model.SLOTargetSpec{
				Availability: t.AvailabilityTarget,
				P95Latency:   t.P95LatencyTarget,
			}
		}
	}
	if len(domain.Targets) == 0 {
		domain.Targets["1d"] = &model.SLOTargetSpec{Availability: 95.0, P95Latency: 300}
		domain.Targets["7d"] = &model.SLOTargetSpec{Availability: 96.0, P95Latency: 280}
		domain.Targets["30d"] = &model.SLOTargetSpec{Availability: 97.0, P95Latency: 250}
	}

	// 计算状态（无请求时视为 unknown，避免 availability=0 误判为 critical）
	if domain.Current != nil && ing.TotalRequests > 0 {
		target := domain.Targets[timeRange]
		if target == nil {
			target = domain.Targets["1d"]
		}
		if target != nil {
			domain.Status = slo.DetermineStatus(domain.Current.Availability, target.Availability, domain.Current.P95Latency, target.P95Latency)
			domain.ErrorBudget = slo.CalculateErrorBudgetRemaining(domain.Current.Availability, target.Availability)
		}
	}

	return domain
}

// ==================== V2: 按真实域名分组 ====================

// DomainsV2 GET /api/v2/slo/domains/v2
func (h *SLOHandler) DomainsV2(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		clusterID = h.defaultClusterID(r.Context())
	}

	timeRange := r.URL.Query().Get("time_range")
	if timeRange == "" {
		timeRange = "1d"
	}

	ctx := r.Context()

	// 从 OTelSnapshot 获取当前 SLO 数据
	otel, err := h.querySvc.GetOTelSnapshot(ctx, clusterID)
	if err != nil {
		sloLog.Error("获取 OTelSnapshot 失败", "err", err)
		handler.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	// 优先从 SLOWindows[timeRange] 获取数据
	var currentIngress, previousIngress []slomodel.IngressSLO
	if otel != nil && otel.SLOWindows != nil {
		if w, ok := otel.SLOWindows[timeRange]; ok {
			currentIngress = w.Current
			previousIngress = w.Previous
		}
	}
	// 回退: 无窗口数据时使用 5min SLOIngress
	if len(currentIngress) == 0 && otel != nil {
		currentIngress = otel.SLOIngress
	}

	// 构建 ServiceKey → IngressSLO 映射
	ingressMap := make(map[string]slomodel.IngressSLO)
	for _, ing := range currentIngress {
		ingressMap[ing.ServiceKey] = ing
	}

	// 构建上期映射
	previousMap := make(map[string]slomodel.IngressSLO)
	for _, ing := range previousIngress {
		previousMap[ing.ServiceKey] = ing
	}

	// 获取所有真实域名
	domainNames, err := h.querySvc.GetSLOAllDomains(ctx, clusterID)
	if err != nil {
		sloLog.Error("获取域名列表失败", "err", err)
		handler.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	targets, _ := h.querySvc.GetSLOTargets(ctx, clusterID)
	targetMap := buildTargetMap(targets)

	var domainResponses []model.DomainSLOResponseV2

	if len(domainNames) == 0 && len(currentIngress) > 0 {
		// 路由映射为空，回退到按 ServiceKey 分组（与 V1 Domains 一致）
		for _, ing := range currentIngress {
			prev := previousMap[ing.ServiceKey]
			domainResponses = append(domainResponses, h.buildDomainSLOV2Fallback(ing, prev, timeRange, targetMap))
		}
	} else {
		for _, domainName := range domainNames {
			domainResponses = append(domainResponses, h.buildDomainSLOV2(ctx, clusterID, domainName, timeRange, targetMap, ingressMap, previousMap))
		}
	}

	var totalAvail, totalBudget, totalRPS float64
	var healthyCount, warningCount, criticalCount int
	for _, domainResp := range domainResponses {
		if domainResp.Summary != nil {
			totalAvail += domainResp.Summary.Availability
			totalRPS += domainResp.Summary.RequestsPerSec
		}
		totalBudget += domainResp.ErrorBudgetRemaining

		switch domainResp.Status {
		case statusHealthy:
			healthyCount++
		case statusWarning:
			warningCount++
		case statusCritical:
			criticalCount++
		}
	}

	var avgAvail, avgBudget float64
	if len(domainResponses) > 0 {
		avgAvail = totalAvail / float64(len(domainResponses))
		avgBudget = totalBudget / float64(len(domainResponses))
	}

	// Ingress 后端服务总数（SLO 只做入口视角，服务网格已移除）
	var totalServices int
	if otel != nil {
		totalServices = len(otel.SLOIngress)
	}

	if domainResponses == nil {
		domainResponses = []model.DomainSLOResponseV2{}
	}

	resp := model.SLODomainsResponseV2{
		Domains: domainResponses,
		Summary: model.SLOSummary{
			TotalServices:   totalServices,
			TotalDomains:    len(domainResponses),
			HealthyCount:    healthyCount,
			WarningCount:    warningCount,
			CriticalCount:   criticalCount,
			AvgAvailability: avgAvail,
			AvgErrorBudget:  avgBudget,
			TotalRPS:        totalRPS,
		},
	}

	handler.WriteJSON(w, http.StatusOK, resp)
}

// buildDomainSLOV2 构建单个域名的 V2 SLO 信息
func (h *SLOHandler) buildDomainSLOV2(ctx context.Context, clusterID, domain, timeRange string, targetMap map[string]map[string]model.SLOTargetResponse, ingressMap, previousMap map[string]slomodel.IngressSLO) model.DomainSLOResponseV2 {
	resp := model.DomainSLOResponseV2{
		Domain: domain,
		TLS:    true,
		Status: statusUnknown,
	}

	// 获取该域名下的所有路由映射
	mappings, err := h.querySvc.GetSLORouteMappingsByDomain(ctx, clusterID, domain)
	if err != nil {
		sloLog.Error("获取路由映射失败", "domain", domain, "err", err)
		return resp
	}

	if len(mappings) > 0 {
		resp.TLS = mappings[0].TLS
	}

	// 按 service_key 分组
	serviceKeyGroups := make(map[string][]*model.SLORouteMapping)
	for _, mapping := range mappings {
		serviceKeyGroups[mapping.ServiceKey] = append(serviceKeyGroups[mapping.ServiceKey], mapping)
	}

	// 汇总变量
	var totalRequests, errorRequests int64
	var totalRPS float64
	var weightedP95Sum, weightedP99Sum float64
	serviceCount := 0

	for serviceKey, groupMappings := range serviceKeyGroups {
		primaryMapping := groupMappings[0]

		var paths []string
		for _, m := range groupMappings {
			paths = append(paths, m.PathPrefix)
		}

		// 从 OTelSnapshot 获取该 service 的指标
		ing, hasData := ingressMap[serviceKey]

		serviceSLO := model.ServiceSLO{
			ServiceKey:  serviceKey,
			ServiceName: primaryMapping.ServiceName,
			ServicePort: primaryMapping.ServicePort,
			Namespace:   primaryMapping.Namespace,
			Paths:       paths,
			IngressName: primaryMapping.IngressName,
			Targets:     make(map[string]*model.SLOTargetSpec),
			Status:      statusUnknown,
		}

		if hasData {
			serviceSLO.Current = ingressToSLOMetrics(ing)

			// 上期数据
			if prevIng, hasPrev := previousMap[serviceKey]; hasPrev {
				serviceSLO.Previous = ingressToSLOMetrics(prevIng)
			}

			// 设置目标
			if hostTargets, ok := targetMap[serviceKey]; ok {
				for tr, t := range hostTargets {
					serviceSLO.Targets[tr] = &model.SLOTargetSpec{
						Availability: t.AvailabilityTarget,
						P95Latency:   t.P95LatencyTarget,
					}
				}
			}
			if len(serviceSLO.Targets) == 0 {
				serviceSLO.Targets["1d"] = &model.SLOTargetSpec{Availability: 95.0, P95Latency: 300}
			}

			// 计算状态（无请求时视为 unknown，避免 availability=0 误判为 critical）
			target := serviceSLO.Targets[timeRange]
			if target == nil {
				target = serviceSLO.Targets["1d"]
			}
			if target != nil && ing.TotalRequests > 0 {
				serviceSLO.Status = slo.DetermineStatus(serviceSLO.Current.Availability, target.Availability, serviceSLO.Current.P95Latency, target.P95Latency)
				serviceSLO.ErrorBudget = slo.CalculateErrorBudgetRemaining(serviceSLO.Current.Availability, target.Availability)
			}

			// 汇总
			totalRequests += ing.TotalRequests
			errorRequests += ing.TotalErrors
			totalRPS += ing.RPS
			p95 := ing.P95Ms
			if p95 == 0 {
				p95 = ing.P90Ms
			}
			weightedP95Sum += p95 * float64(ing.TotalRequests)
			weightedP99Sum += ing.P99Ms * float64(ing.TotalRequests)
			serviceCount++
		}

		resp.Services = append(resp.Services, serviceSLO)

		// 统计最差状态
		if serviceSLO.Status == statusCritical && resp.Status != statusCritical {
			resp.Status = statusCritical
		} else if serviceSLO.Status == statusWarning && resp.Status != statusCritical && resp.Status != statusWarning {
			resp.Status = statusWarning
		} else if serviceSLO.Status == statusHealthy && resp.Status == statusUnknown {
			resp.Status = statusHealthy
		}
	}

	// 域名级别目标
	if domainTargets, ok := targetMap[domain]; ok {
		resp.Targets = make(map[string]*model.SLOTargetSpec)
		for tr, t := range domainTargets {
			resp.Targets[tr] = &model.SLOTargetSpec{
				Availability: t.AvailabilityTarget,
				P95Latency:   t.P95LatencyTarget,
			}
		}
	}
	if len(resp.Targets) == 0 {
		resp.Targets = map[string]*model.SLOTargetSpec{
			"1d": {Availability: 95.0, P95Latency: 300},
		}
	}

	// 域名级别汇总
	if serviceCount > 0 {
		var p95, p99 int
		if totalRequests > 0 {
			p95 = int(weightedP95Sum / float64(totalRequests))
			p99 = int(weightedP99Sum / float64(totalRequests))
		}
		resp.Summary = &model.SLOMetrics{
			Availability:   slo.CalculateAvailability(totalRequests, errorRequests),
			P95Latency:     p95,
			P99Latency:     p99,
			ErrorRate:      slo.CalculateErrorRate(totalRequests, errorRequests),
			RequestsPerSec: totalRPS,
			TotalRequests:  totalRequests,
		}

		target := resp.Targets[timeRange]
		if target == nil {
			target = resp.Targets["1d"]
		}
		availTarget := 95.0
		if target != nil {
			availTarget = target.Availability
		}
		resp.ErrorBudgetRemaining = slo.CalculateErrorBudgetRemaining(resp.Summary.Availability, availTarget)
	}

	// 域名级别上期汇总
	if len(previousMap) > 0 {
		var prevTotal, prevErrors int64
		var prevRPS, prevWP95, prevWP99 float64
		for serviceKey := range serviceKeyGroups {
			if prevIng, ok := previousMap[serviceKey]; ok {
				prevTotal += prevIng.TotalRequests
				prevErrors += prevIng.TotalErrors
				prevRPS += prevIng.RPS
				pp95 := prevIng.P95Ms
				if pp95 == 0 {
					pp95 = prevIng.P90Ms
				}
				prevWP95 += pp95 * float64(prevIng.TotalRequests)
				prevWP99 += prevIng.P99Ms * float64(prevIng.TotalRequests)
			}
		}
		if prevTotal > 0 {
			var pp95, pp99 int
			pp95 = int(prevWP95 / float64(prevTotal))
			pp99 = int(prevWP99 / float64(prevTotal))
			resp.Previous = &model.SLOMetrics{
				Availability:   slo.CalculateAvailability(prevTotal, prevErrors),
				P95Latency:     pp95,
				P99Latency:     pp99,
				ErrorRate:      slo.CalculateErrorRate(prevTotal, prevErrors),
				RequestsPerSec: prevRPS,
				TotalRequests:  prevTotal,
			}
		}
	}

	return resp
}

// fallbackDomainName 路由映射表为空时的域名来源。
//
// Agent 侧的 RouteRepository 已经从 HTTPRoute 解析出真实域名填进 DisplayName，
// 这里直接用即可；只有它缺失（或未解析、等于 ServiceKey）时才退回 ServiceKey。
// 否则页面会显示 "geass-v3/geass-gateway" 而不是 "geass-api.bukahou.com"。
func fallbackDomainName(ing slomodel.IngressSLO) string {
	if ing.DisplayName != "" && ing.DisplayName != ing.ServiceKey {
		return ing.DisplayName
	}
	return ing.ServiceKey
}

// buildDomainSLOV2Fallback 当路由映射为空时，从单个 IngressSLO 直接构建 V2 响应
// 一个服务一个条目，域名取 DisplayName
func (h *SLOHandler) buildDomainSLOV2Fallback(ing slomodel.IngressSLO, prevIng slomodel.IngressSLO, timeRange string, targetMap map[string]map[string]model.SLOTargetResponse) model.DomainSLOResponseV2 {
	resp := model.DomainSLOResponseV2{
		Domain: fallbackDomainName(ing),
		TLS:    true,
		Status: statusUnknown,
	}

	serviceSLO := model.ServiceSLO{
		ServiceKey:  ing.ServiceKey,
		ServiceName: ing.DisplayName,
		Targets:     make(map[string]*model.SLOTargetSpec),
		Status:      statusUnknown,
	}

	serviceSLO.Current = ingressToSLOMetrics(ing)

	// 上期数据
	if prevIng.TotalRequests > 0 {
		serviceSLO.Previous = ingressToSLOMetrics(prevIng)
	}

	// 设置目标
	if hostTargets, ok := targetMap[ing.ServiceKey]; ok {
		for tr, t := range hostTargets {
			serviceSLO.Targets[tr] = &model.SLOTargetSpec{
				Availability: t.AvailabilityTarget,
				P95Latency:   t.P95LatencyTarget,
			}
		}
	}
	if len(serviceSLO.Targets) == 0 {
		serviceSLO.Targets["1d"] = &model.SLOTargetSpec{Availability: 95.0, P95Latency: 300}
	}

	// 计算状态（无请求时视为 unknown，避免 availability=0 误判为 critical）
	target := serviceSLO.Targets[timeRange]
	if target == nil {
		target = serviceSLO.Targets["1d"]
	}
	if target != nil && serviceSLO.Current != nil && ing.TotalRequests > 0 {
		serviceSLO.Status = slo.DetermineStatus(serviceSLO.Current.Availability, target.Availability, serviceSLO.Current.P95Latency, target.P95Latency)
		serviceSLO.ErrorBudget = slo.CalculateErrorBudgetRemaining(serviceSLO.Current.Availability, target.Availability)
	}

	resp.Services = []model.ServiceSLO{serviceSLO}
	resp.Status = serviceSLO.Status
	resp.Summary = serviceSLO.Current
	resp.Previous = serviceSLO.Previous
	resp.Targets = serviceSLO.Targets

	if serviceSLO.Current != nil {
		availTarget := 95.0
		if target != nil {
			availTarget = target.Availability
		}
		resp.ErrorBudgetRemaining = slo.CalculateErrorBudgetRemaining(serviceSLO.Current.Availability, availTarget)
	}

	return resp
}

// ==================== 域名详情 ====================

// DomainDetail GET /api/v2/slo/domains/detail
func (h *SLOHandler) DomainDetail(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	host := r.URL.Query().Get("host")
	if host == "" {
		handler.WriteError(w, http.StatusBadRequest, "host required")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		clusterID = h.defaultClusterID(r.Context())
	}

	timeRange := r.URL.Query().Get("time_range")
	if timeRange == "" {
		timeRange = "1d"
	}

	ctx := r.Context()

	otel, err := h.querySvc.GetOTelSnapshot(ctx, clusterID)
	if err != nil {
		sloLog.Error("获取 OTelSnapshot 失败", "err", err)
		handler.WriteError(w, http.StatusInternalServerError, err.Error())
		return
	}

	targets, _ := h.querySvc.GetSLOTargets(ctx, clusterID)
	targetMap := buildTargetMap(targets)

	// 查找匹配的 IngressSLO
	if otel != nil {
		for _, ing := range otel.SLOIngress {
			if ing.ServiceKey == host {
				domain := h.buildDomainFromIngress(ctx, clusterID, ing, timeRange, targetMap)
				handler.WriteJSON(w, http.StatusOK, domain)
				return
			}
		}
	}

	// 未找到，返回空数据
	domain := model.DomainSLO{
		Host:    host,
		Targets: make(map[string]*model.SLOTargetSpec),
		Status:  statusUnknown,
		Trend:   "stable",
	}
	handler.WriteJSON(w, http.StatusOK, domain)
}

// ==================== 域名历史 ====================

// DomainHistory GET /api/v2/slo/domains/history
func (h *SLOHandler) DomainHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		handler.WriteError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	host := r.URL.Query().Get("host")
	if host == "" {
		handler.WriteError(w, http.StatusBadRequest, "host required")
		return
	}

	clusterID := r.URL.Query().Get("cluster_id")
	if clusterID == "" {
		clusterID = h.defaultClusterID(r.Context())
	}

	ctx := r.Context()

	// 加载 targets 用于计算 error budget
	targets, _ := h.querySvc.GetSLOTargets(ctx, clusterID)
	targetMap := buildTargetMap(targets)

	timeRange := r.URL.Query().Get("time_range")
	if timeRange == "" {
		timeRange = "1d"
	}

	availTarget := 95.0
	if hostTargets, ok := targetMap[host]; ok {
		if t, ok := hostTargets[timeRange]; ok {
			availTarget = t.AvailabilityTarget
		} else if t, ok := hostTargets["1d"]; ok {
			availTarget = t.AvailabilityTarget
		}
	}

	// 域名 → ServiceKey 映射（与 LatencyDistribution 一致）
	mappings, _ := h.querySvc.GetSLORouteMappingsByDomain(ctx, clusterID, host)
	serviceKeys := make(map[string]bool)
	for _, m := range mappings {
		serviceKeys[m.ServiceKey] = true
	}
	if len(serviceKeys) == 0 {
		serviceKeys[host] = true
	}

	// 优先从 SLOWindows[timeRange].History 获取历史数据
	var history []model.SLODomainHistoryItem

	otel, err := h.querySvc.GetOTelSnapshot(ctx, clusterID)
	if err != nil {
		sloLog.Error("获取 OTelSnapshot 失败", "err", err)
	}

	windowHit := false
	if otel != nil && otel.SLOWindows != nil {
		if w, ok := otel.SLOWindows[timeRange]; ok && w.History != nil {
			for _, p := range w.History {
				if serviceKeys[p.ServiceKey] {
					history = append(history, model.SLODomainHistoryItem{
						Timestamp:    p.Timestamp.Format(time.RFC3339),
						Availability: p.Availability,
						P95Latency:   int(p.P95Ms),
						P99Latency:   int(p.P99Ms),
						RPS:          p.RPS,
						ErrorRate:    p.ErrorRate,
						ErrorBudget:  slo.CalculateErrorBudgetRemaining(p.Availability, availTarget),
					})
					windowHit = true
				}
			}
		}
	}

	// 回退: OTelTimeline ring buffer（≤15min，10s 精度）
	if !windowHit {
		since := time.Now().Add(-15 * time.Minute)
		timeline, tlErr := h.querySvc.GetOTelTimeline(ctx, clusterID, since)
		if tlErr != nil {
			sloLog.Error("获取 OTelTimeline 失败", "err", tlErr)
		}

		for _, entry := range timeline {
			if entry.Snapshot == nil {
				continue
			}
			for _, ing := range entry.Snapshot.SLOIngress {
				if serviceKeys[ing.ServiceKey] {
					avail := ing.SuccessRate
					p95 := int(ing.P95Ms)
					if p95 == 0 {
						p95 = int(ing.P90Ms)
					}
					history = append(history, model.SLODomainHistoryItem{
						Timestamp:    entry.Timestamp.Format(time.RFC3339),
						Availability: avail,
						P95Latency:   p95,
						P99Latency:   int(ing.P99Ms),
						RPS:          ing.RPS,
						ErrorRate:    ing.ErrorRate,
						ErrorBudget:  slo.CalculateErrorBudgetRemaining(avail, availTarget),
					})
					break
				}
			}
		}
	}

	if history == nil {
		history = []model.SLODomainHistoryItem{}
	}

	resp := model.SLODomainHistoryResponse{
		Host:    host,
		History: history,
	}

	handler.WriteJSON(w, http.StatusOK, resp)
}

// ==================== 域名辅助函数 ====================

// ingressToSLOMetrics 将 IngressSLO 转换为 SLOMetrics
// IngressSLO 的 SuccessRate/ErrorRate 已经是 0-100 百分比（ClickHouse 聚合结果）
func ingressToSLOMetrics(ing slomodel.IngressSLO) *model.SLOMetrics {
	p95 := int(ing.P95Ms)
	if p95 == 0 {
		p95 = int(ing.P90Ms) // 回退兼容
	}
	return &model.SLOMetrics{
		Availability:   ing.SuccessRate,
		P95Latency:     p95,
		P99Latency:     int(ing.P99Ms),
		ErrorRate:      ing.ErrorRate,
		RequestsPerSec: ing.RPS,
		TotalRequests:  ing.TotalRequests,
	}
}
