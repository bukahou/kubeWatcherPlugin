// atlhyper_master_v2/aiops/risk/scorer.go
// 三阶段风险评分流水线
package risk

import (
	"sort"
	"sync"
	"time"

	"AtlHyper/atlhyper_master_v2/aiops"
)

// Scorer 风险评分引擎
type Scorer struct {
	config       *RiskConfig
	mu           sync.RWMutex
	results      map[string]*aiops.ClusterRisk           // clusterID -> ClusterRisk
	entityMap    map[string]map[string]*aiops.EntityRisk // clusterID -> entityKey -> EntityRisk
	propagations map[string][]*aiops.PropagationPath     // clusterID -> propagation paths
	firstAnomaly map[string]int64                        // entityKey -> 首次异常时间
}

// NewScorer 创建风险评分引擎
func NewScorer(config *RiskConfig) *Scorer {
	if config == nil {
		config = DefaultRiskConfig()
	}
	return &Scorer{
		config:       config,
		results:      make(map[string]*aiops.ClusterRisk),
		entityMap:    make(map[string]map[string]*aiops.EntityRisk),
		propagations: make(map[string][]*aiops.PropagationPath),
		firstAnomaly: make(map[string]int64),
	}
}

// Calculate 执行三阶段风险评分
func (s *Scorer) Calculate(
	clusterID string,
	graph *aiops.DependencyGraph,
	anomalies []*aiops.AnomalyResult,
	sloCtx *SLOContext,
) *aiops.ClusterRisk {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now().Unix()

	// 更新首次异常时间记录
	s.updateFirstAnomalyTimes(anomalies, now)

	// Stage 1: 局部风险
	localRisks := ComputeLocalRisks(anomalies, s.config)

	// Stage 2: 时序权重
	weightedRisks := ApplyTemporalWeights(localRisks, s.firstAnomaly, now, s.config.TemporalHalfLife)

	// Stage 3: 图传播
	finalRisks, paths := Propagate(graph, weightedRisks, s.config.SelfWeight)
	s.propagations[clusterID] = paths

	// 构建 EntityRisk 列表
	entityRisks := s.buildEntityRisks(graph, localRisks, weightedRisks, finalRisks)
	s.entityMap[clusterID] = entityRisks

	// 聚合 ClusterRisk
	clusterRisk := Aggregate(clusterID, entityRisks, finalRisks, sloCtx, s.config, now)
	s.results[clusterID] = clusterRisk

	return clusterRisk
}

// GetClusterRisk 获取集群风险
func (s *Scorer) GetClusterRisk(clusterID string) *aiops.ClusterRisk {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.results[clusterID]
}

// GetEntityRisks 获取实体风险列表
func (s *Scorer) GetEntityRisks(clusterID, sortBy string, limit int) []*aiops.EntityRisk {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entityMap := s.entityMap[clusterID]
	if entityMap == nil {
		return nil
	}

	risks := make([]*aiops.EntityRisk, 0, len(entityMap))
	for _, r := range entityMap {
		risks = append(risks, r)
	}

	switch sortBy {
	case "r_local":
		sort.Slice(risks, func(i, j int) bool { return risks[i].RLocal > risks[j].RLocal })
	default: // r_final
		sort.Slice(risks, func(i, j int) bool { return risks[i].RFinal > risks[j].RFinal })
	}

	if limit > 0 && limit < len(risks) {
		risks = risks[:limit]
	}
	return risks
}

// GetEntityRisk 获取单个实体风险
func (s *Scorer) GetEntityRisk(clusterID, entityKey string) *aiops.EntityRisk {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if m := s.entityMap[clusterID]; m != nil {
		return m[entityKey]
	}
	return nil
}

// GetPropagationPaths 获取指定实体的传播路径
func (s *Scorer) GetPropagationPaths(clusterID, entityKey string) []*aiops.PropagationPath {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*aiops.PropagationPath
	for _, p := range s.propagations[clusterID] {
		if p.To == entityKey || p.From == entityKey {
			result = append(result, p)
		}
	}
	return result
}

// GetEntityRiskMap 获取集群所有实体风险（供状态机评估）
func (s *Scorer) GetEntityRiskMap(clusterID string) map[string]*aiops.EntityRisk {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.entityMap[clusterID]
}

// GetEntityCount 获取集群实体数量
func (s *Scorer) GetEntityCount(clusterID string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.entityMap[clusterID])
}

// updateFirstAnomalyTimes 记录每个实体首次出现异常的时间
func (s *Scorer) updateFirstAnomalyTimes(anomalies []*aiops.AnomalyResult, now int64) {
	currentAnomaly := map[string]bool{}
	for _, a := range anomalies {
		if a.IsAnomaly {
			currentAnomaly[a.EntityKey] = true
			if _, exists := s.firstAnomaly[a.EntityKey]; !exists {
				s.firstAnomaly[a.EntityKey] = now
			}
		}
	}
	// 清除已恢复的实体
	for key := range s.firstAnomaly {
		if !currentAnomaly[key] {
			delete(s.firstAnomaly, key)
		}
	}
}

// buildEntityRisks 构建 EntityRisk 映射
func (s *Scorer) buildEntityRisks(
	graph *aiops.DependencyGraph,
	localRisks, weightedRisks, finalRisks map[string]float64,
) map[string]*aiops.EntityRisk {
	entityRisks := make(map[string]*aiops.EntityRisk, len(graph.Nodes))

	for key, node := range graph.Nodes {
		rLocal := localRisks[key]
		rWeighted := weightedRisks[key]
		rFinal := finalRisks[key]

		wTime := 1.0
		if rLocal > 0 {
			wTime = rWeighted / rLocal
		}

		firstAnomaly := s.firstAnomaly[key]

		entityRisks[key] = &aiops.EntityRisk{
			EntityKey:    key,
			EntityType:   node.Type,
			Namespace:    node.Namespace,
			Name:         node.Name,
			RLocal:       rLocal,
			WTime:        wTime,
			RWeighted:    rWeighted,
			RFinal:       rFinal,
			RiskLevel:    aiops.RiskLevel(rFinal),
			FirstAnomaly: firstAnomaly,
		}
	}

	return entityRisks
}
