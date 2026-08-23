// atlhyper_master_v2/datahub/redis/store.go
// RedisStore Redis 数据存储实现
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"AtlHyper/common/logger"
	agentmodel "AtlHyper/model_v3/agent"
	"AtlHyper/model_v3/cluster"
)

var log = logger.Module("RedisStore")

// Key 前缀
const (
	keySnapshot = "datahub:snapshot:" // + clusterID -> JSON
	keyAgents   = "datahub:agents"    // SET of clusterIDs
	keyAgent    = "datahub:agent:"    // + clusterID -> JSON (AgentInfo)
)

// Config RedisStore 配置
type Config struct {
	Addr            string
	Password        string
	DB              int
	EventRetention  time.Duration
	HeartbeatExpire time.Duration
}

// RedisStore Redis 数据存储
type RedisStore struct {
	client          *redis.Client
	eventRetention  time.Duration
	heartbeatExpire time.Duration

	stopCh chan struct{}
	wg     sync.WaitGroup
}

// NewRedisStore 创建 RedisStore
func NewRedisStore(cfg Config) *RedisStore {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	return &RedisStore{
		client:          client,
		eventRetention:  cfg.EventRetention,
		heartbeatExpire: cfg.HeartbeatExpire,
		stopCh:          make(chan struct{}),
	}
}

// Start 启动 RedisStore
func (s *RedisStore) Start() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := s.client.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("redis ping failed: %w", err)
	}

	s.wg.Add(1)
	go s.cleanupLoop()

	log.Info("已启动")
	return nil
}

// Stop 停止 RedisStore
func (s *RedisStore) Stop() error {
	close(s.stopCh)
	s.wg.Wait()

	if err := s.client.Close(); err != nil {
		return fmt.Errorf("redis close failed: %w", err)
	}
	log.Info("已停止")
	return nil
}

// cleanupLoop 定期清理过期数据
func (s *RedisStore) cleanupLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case <-ticker.C:
			s.cleanupExpiredEvents()
			s.updateAgentStatus()
		}
	}
}

// cleanupExpiredEvents 清理过期 Events
func (s *RedisStore) cleanupExpiredEvents() {
	ctx := context.Background()
	cutoff := time.Now().Add(-s.eventRetention)

	// 获取所有集群 ID
	clusterIDs, err := s.client.SMembers(ctx, keyAgents).Result()
	if err != nil {
		return
	}

	for _, clusterID := range clusterIDs {
		data, err := s.client.Get(ctx, keySnapshot+clusterID).Bytes()
		if err != nil {
			continue
		}

		var snapshot cluster.ClusterSnapshot
		if err := json.Unmarshal(data, &snapshot); err != nil {
			continue
		}

		var validEvents []cluster.Event
		for _, e := range snapshot.Events {
			if e.LastTimestamp.After(cutoff) {
				validEvents = append(validEvents, e)
			}
		}

		if len(validEvents) < len(snapshot.Events) {
			snapshot.Events = validEvents
			newData, _ := json.Marshal(&snapshot)
			s.client.Set(ctx, keySnapshot+clusterID, newData, 0)
			log.Info("已清理过期事件", "clusterID", clusterID, "before", len(snapshot.Events), "after", len(validEvents))
		}
	}
}

// updateAgentStatus 更新 Agent 在线状态
func (s *RedisStore) updateAgentStatus() {
	ctx := context.Background()
	cutoff := time.Now().Add(-s.heartbeatExpire)

	clusterIDs, err := s.client.SMembers(ctx, keyAgents).Result()
	if err != nil {
		return
	}

	for _, clusterID := range clusterIDs {
		data, err := s.client.Get(ctx, keyAgent+clusterID).Bytes()
		if err != nil {
			continue
		}

		var agent agentmodel.AgentInfo
		if err := json.Unmarshal(data, &agent); err != nil {
			continue
		}

		if agent.LastHeartbeat.Before(cutoff) && agent.Status == agentmodel.StatusOnline {
			agent.Status = agentmodel.StatusOffline
			newData, _ := json.Marshal(&agent)
			s.client.Set(ctx, keyAgent+clusterID, newData, 0)
			log.Info("Agent 已标记为离线", "clusterID", clusterID)
		}
	}
}

// ==================== 快照管理 ====================

// SetSnapshot 存储集群快照
func (s *RedisStore) SetSnapshot(clusterID string, snapshot *cluster.ClusterSnapshot) error {
	ctx := context.Background()

	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("marshal snapshot: %w", err)
	}

	if err := s.client.Set(ctx, keySnapshot+clusterID, data, 0).Err(); err != nil {
		return fmt.Errorf("set snapshot: %w", err)
	}

	// 更新 Agent 状态
	s.client.SAdd(ctx, keyAgents, clusterID)

	agentData, err := s.client.Get(ctx, keyAgent+clusterID).Bytes()
	if err == redis.Nil {
		// 新 Agent
		agent := agentmodel.AgentInfo{
			ClusterID:     clusterID,
			Status:        agentmodel.StatusOnline,
			LastHeartbeat: time.Now(),
			LastSnapshot:  snapshot.FetchedAt,
		}
		newData, _ := json.Marshal(&agent)
		s.client.Set(ctx, keyAgent+clusterID, newData, 0)
	} else if err == nil {
		var agent agentmodel.AgentInfo
		if json.Unmarshal(agentData, &agent) == nil {
			agent.LastSnapshot = snapshot.FetchedAt
			newData, _ := json.Marshal(&agent)
			s.client.Set(ctx, keyAgent+clusterID, newData, 0)
		}
	}

	return nil
}

// GetSnapshot 获取集群快照
func (s *RedisStore) GetSnapshot(clusterID string) (*cluster.ClusterSnapshot, error) {
	ctx := context.Background()

	data, err := s.client.Get(ctx, keySnapshot+clusterID).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}

	var snapshot cluster.ClusterSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("unmarshal snapshot: %w", err)
	}
	return &snapshot, nil
}

// ==================== Agent 状态 ====================

// UpdateHeartbeat 更新 Agent 心跳
func (s *RedisStore) UpdateHeartbeat(clusterID string) error {
	ctx := context.Background()

	s.client.SAdd(ctx, keyAgents, clusterID)

	agentData, err := s.client.Get(ctx, keyAgent+clusterID).Bytes()
	if err == redis.Nil {
		agent := agentmodel.AgentInfo{
			ClusterID:     clusterID,
			Status:        agentmodel.StatusOnline,
			LastHeartbeat: time.Now(),
		}
		data, _ := json.Marshal(&agent)
		return s.client.Set(ctx, keyAgent+clusterID, data, 0).Err()
	}
	if err != nil {
		return fmt.Errorf("get agent: %w", err)
	}

	var agent agentmodel.AgentInfo
	if err := json.Unmarshal(agentData, &agent); err != nil {
		return fmt.Errorf("unmarshal agent: %w", err)
	}

	agent.LastHeartbeat = time.Now()
	agent.Status = agentmodel.StatusOnline
	data, _ := json.Marshal(&agent)
	return s.client.Set(ctx, keyAgent+clusterID, data, 0).Err()
}

// GetAgentStatus 获取 Agent 状态
func (s *RedisStore) GetAgentStatus(clusterID string) (*agentmodel.AgentStatus, error) {
	ctx := context.Background()

	data, err := s.client.Get(ctx, keyAgent+clusterID).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get agent: %w", err)
	}

	var agent agentmodel.AgentInfo
	if err := json.Unmarshal(data, &agent); err != nil {
		return nil, fmt.Errorf("unmarshal agent: %w", err)
	}

	return &agentmodel.AgentStatus{
		ClusterID:     agent.ClusterID,
		Status:        agent.Status,
		LastHeartbeat: agent.LastHeartbeat,
		LastSnapshot:  agent.LastSnapshot,
	}, nil
}

// ListAgents 列出所有 Agent
func (s *RedisStore) ListAgents() ([]agentmodel.AgentInfo, error) {
	ctx := context.Background()

	clusterIDs, err := s.client.SMembers(ctx, keyAgents).Result()
	if err != nil {
		return nil, fmt.Errorf("get agents set: %w", err)
	}

	result := make([]agentmodel.AgentInfo, 0, len(clusterIDs))
	for _, clusterID := range clusterIDs {
		data, err := s.client.Get(ctx, keyAgent+clusterID).Bytes()
		if err != nil {
			continue
		}
		var agent agentmodel.AgentInfo
		if json.Unmarshal(data, &agent) == nil {
			result = append(result, agent)
		}
	}
	return result, nil
}

// ==================== Event 查询 ====================

// GetEvents 获取集群当前所有 Events
func (s *RedisStore) GetEvents(clusterID string) ([]cluster.Event, error) {
	ctx := context.Background()

	data, err := s.client.Get(ctx, keySnapshot+clusterID).Bytes()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get snapshot: %w", err)
	}

	var snapshot cluster.ClusterSnapshot
	if err := json.Unmarshal(data, &snapshot); err != nil {
		return nil, fmt.Errorf("unmarshal snapshot: %w", err)
	}
	return snapshot.Events, nil
}

// ==================== OTel 时间线 ====================

// GetOTelTimeline Redis 暂不实现 OTel 时间线（仅内存支持）
func (s *RedisStore) GetOTelTimeline(clusterID string, since time.Time) ([]cluster.OTelEntry, error) {
	return nil, nil
}
