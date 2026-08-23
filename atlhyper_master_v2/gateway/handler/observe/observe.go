// atlhyper_master_v2/gateway/handler/observe.go
// 可观测性查询 Handler — 共用结构体、缓存、执行方法
//
// 各信号域 Handler 方法分布在:
//
//	observe_metrics.go     — MetricsSummary / MetricsNodes / MetricsNodeRoute / MetricsHardware
//	observe_logs.go        — LogsQuery / LogsSummary
//	observe_apm.go         — TracesList / TracesServices / TracesTopology / TracesOperations / TracesDetail / TracesStats / APMServiceSeries
//	observe_slo_query.go   — SLOSummary / SLOIngress / SLOServices / SLOEdges / SLOTimeSeries
//	observe_timeline.go    — 时序辅助函数
package observe

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/gateway/handler"
	"AtlHyper/atlhyper_master_v2/model"
	"AtlHyper/atlhyper_master_v2/service"
)

// ObserveHandler 可观测性查询 Handler
//
// Dashboard + Detail 端点（13 个）直读快照/预聚合时序，
// 仅 TracesDetail + LogsQuery（2 个）保留 Command 机制。
type ObserveHandler struct {
	svc      service.Ops
	querySvc service.Query
	cache    *observeCache
}

// NewObserveHandler 创建 ObserveHandler
func NewObserveHandler(svc service.Ops, querySvc service.Query) *ObserveHandler {
	return &ObserveHandler{
		svc:      svc,
		querySvc: querySvc,
		cache:    newObserveCache(),
	}
}

// ================================================================
// TTL 缓存（带容量上限）
// ================================================================

// maxCacheEntries 缓存最大条目数，防止不同查询参数导致缓存无限膨胀
const maxCacheEntries = 128

type cacheEntry struct {
	data      json.RawMessage
	expiresAt time.Time
}

type observeCache struct {
	mu    sync.RWMutex
	items map[string]*cacheEntry
}

func newObserveCache() *observeCache {
	c := &observeCache{items: make(map[string]*cacheEntry)}
	// 后台清理过期条目
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			c.cleanup()
		}
	}()
	return c
}

func (c *observeCache) get(key string) (json.RawMessage, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	entry, ok := c.items[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.data, true
}

func (c *observeCache) set(key string, data json.RawMessage, ttl time.Duration) {
	if ttl <= 0 {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	// 容量上限：超出时淘汰最早过期的条目
	if len(c.items) >= maxCacheEntries {
		c.evictOldest()
	}
	c.items[key] = &cacheEntry{data: data, expiresAt: time.Now().Add(ttl)}
}

func (c *observeCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for k, v := range c.items {
		if now.After(v.expiresAt) {
			delete(c.items, k)
		}
	}
}

// evictOldest 淘汰最早过期的 25% 条目（调用方已持有锁）
func (c *observeCache) evictOldest() {
	evictCount := len(c.items) / 4
	if evictCount < 1 {
		evictCount = 1
	}
	// 找最早过期的条目逐个删除
	for i := 0; i < evictCount; i++ {
		var oldestKey string
		var oldestTime time.Time
		for k, v := range c.items {
			if oldestKey == "" || v.expiresAt.Before(oldestTime) {
				oldestKey = k
				oldestTime = v.expiresAt
			}
		}
		if oldestKey != "" {
			delete(c.items, oldestKey)
		}
	}
}

// ================================================================
// 统一执行方法
// ================================================================

func (h *ObserveHandler) executeQuery(
	w http.ResponseWriter, r *http.Request,
	clusterID, action string,
	params map[string]interface{},
	cacheTTL time.Duration,
) {
	// 1. 检查缓存
	cacheKey := buildCacheKey(clusterID, action, params)
	if data, ok := h.cache.get(cacheKey); ok {
		handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
			"message": "获取成功",
			"data":    data,
		})
		return
	}

	// 2. 同步执行指令（创建 + 等待结果，30 秒超时）
	result, err := h.svc.ExecuteCommandSync(r.Context(), &model.CreateCommandRequest{
		ClusterID: clusterID,
		Action:    action,
		Params:    params,
		Source:    "web",
	}, 30*time.Second)
	if err != nil {
		if strings.Contains(err.Error(), "create command:") {
			handler.WriteError(w, http.StatusInternalServerError, "创建查询指令失败: "+err.Error())
		} else {
			handler.WriteError(w, http.StatusGatewayTimeout, "查询超时，请稍后重试")
		}
		return
	}
	if result == nil {
		handler.WriteError(w, http.StatusGatewayTimeout, "查询超时，请稍后重试")
		return
	}

	// 4. 检查执行结果
	if !result.Success {
		errMsg := result.Error
		if errMsg == "" {
			errMsg = "查询失败"
		}
		handler.WriteError(w, http.StatusInternalServerError, errMsg)
		return
	}

	// 5. JSON 透传：result.Output 是 Agent 返回的 JSON 字符串
	rawData := json.RawMessage(result.Output)

	// 6. 写缓存
	h.cache.set(cacheKey, rawData, cacheTTL)

	// 7. 返回
	handler.WriteJSON(w, http.StatusOK, map[string]interface{}{
		"message": "获取成功",
		"data":    rawData,
	})
}

// buildCacheKey 构建缓存 key
func buildCacheKey(clusterID, action string, params map[string]interface{}) string {
	if len(params) == 0 {
		return clusterID + ":" + action
	}
	b, _ := json.Marshal(params)
	hash := sha256.Sum256(b)
	return fmt.Sprintf("%s:%s:%x", clusterID, action, hash[:8])
}

// requireClusterID 提取并校验 cluster_id 参数
func requireClusterID(r *http.Request) (string, bool) {
	clusterID := r.URL.Query().Get("cluster_id")
	return clusterID, clusterID != ""
}

// parseTimeRangeMinutes 解析时间范围字符串为分钟数
// 支持格式: "15m", "1h", "30m" 等
func parseTimeRangeMinutes(s string) (int, bool) {
	if s == "" {
		return 0, false
	}
	d, err := time.ParseDuration(s)
	if err != nil {
		return 0, false
	}
	return int(d.Minutes()), true
}

// cacheTTLForMinutes 根据查询时间范围返回合适的缓存 TTL
func cacheTTLForMinutes(minutes int) time.Duration {
	switch {
	case minutes <= 60:
		return 30 * time.Second
	case minutes <= 360:
		return 2 * time.Minute
	case minutes <= 1440:
		return 5 * time.Minute
	default:
		return 10 * time.Minute
	}
}
