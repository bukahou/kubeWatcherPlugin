package scheduler

import (
	"context"
	"sync"
	"time"

	"AtlHyper/atlhyper_agent_v2/gateway"
	"AtlHyper/atlhyper_agent_v2/service"
	"AtlHyper/common/logger"
)

var log = logger.Module("Scheduler")

// Config 调度器配置
type Config struct {
	// SnapshotInterval 快照采集间隔
	// 每隔此时间采集一次集群资源并推送给 Master
	SnapshotInterval time.Duration

	// CommandPollInterval 指令轮询间隔
	// 长轮询返回后等待此时间再次轮询
	CommandPollInterval time.Duration

	// HeartbeatInterval 心跳间隔
	// 每隔此时间向 Master 发送心跳
	HeartbeatInterval time.Duration

	// SnapshotTimeout 快照采集操作超时
	SnapshotTimeout time.Duration

	// SnapshotPushTimeout 快照推送操作超时
	// 独立于 SnapshotTimeout，避免 Collect 慢耗尽 ctx 预算后连累 Push
	SnapshotPushTimeout time.Duration

	// CommandPollTimeout 指令轮询操作超时
	CommandPollTimeout time.Duration

	// HeartbeatTimeout 心跳操作超时
	HeartbeatTimeout time.Duration
}

// Scheduler 调度器
//
// 管理 Agent 各项后台任务的生命周期:
//   - 快照采集循环 - 定时采集集群资源（含 SLO 数据），推送给 Master
//   - 指令轮询循环 - 长轮询获取 Master 指令，执行后上报结果
//   - 心跳循环 - 定时向 Master 发送心跳，维持连接状态
type Scheduler struct {
	config Config

	// 依赖的服务
	snapshotSvc service.SnapshotService
	commandSvc  service.CommandService
	masterGw    gateway.MasterGateway

	// 生命周期控制
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup

	// 状态追踪（用于变化检测）
	lastPodCount        int
	lastNodeCount       int
	lastDeploymentCount int
}

// New 创建调度器
//
// 参数:
//   - config: 调度器配置
//   - snapshotSvc: 快照服务，用于采集集群资源（含 SLO 数据）
//   - commandSvc: 指令服务，用于执行 Master 下发的指令
//   - masterGw: Master 网关，用于与 Master 通信
func New(
	config Config,
	snapshotSvc service.SnapshotService,
	commandSvc service.CommandService,
	masterGw gateway.MasterGateway,
) *Scheduler {
	return &Scheduler{
		config:      config,
		snapshotSvc: snapshotSvc,
		commandSvc:  commandSvc,
		masterGw:    masterGw,
	}
}

// Start 启动调度器
func (s *Scheduler) Start(ctx context.Context) error {
	s.ctx, s.cancel = context.WithCancel(ctx)

	// 启动后台任务: 快照 + ops轮询 + ai轮询 + 心跳
	s.wg.Add(4)
	go s.runSnapshotLoop()     // 快照采集（含 OTel 概览数据）
	go s.runCommandLoop("ops") // 系统操作指令轮询
	go s.runCommandLoop("ai")  // AI 查询指令轮询
	go s.runHeartbeatLoop()    // 心跳

	log.Info("调度器已启动")
	return nil
}

// Stop 停止调度器
func (s *Scheduler) Stop() error {
	if s.cancel != nil {
		s.cancel() // 通知所有 goroutine 停止
	}
	s.wg.Wait() // 等待所有 goroutine 退出
	log.Info("调度器已停止")
	return nil
}

// =============================================================================
// 快照采集循环
// =============================================================================

// runSnapshotLoop 快照采集循环
//
// 工作流程:
//  1. 启动时立即采集一次
//  2. 之后每隔 SnapshotInterval 采集一次
//  3. 采集失败只记录日志，不中断循环
func (s *Scheduler) runSnapshotLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.SnapshotInterval)
	defer ticker.Stop()

	// 立即执行一次
	s.collectAndPushSnapshot()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.collectAndPushSnapshot()
		}
	}
}

// collectAndPushSnapshot 采集并推送快照
//
// Collect 和 Push 使用独立 ctx，互不干扰：
//   - Collect ctx 超时只影响采集本身
//   - Push ctx 拥有全新的超时预算，即使 Collect 耗时较长也能正常发送
func (s *Scheduler) collectAndPushSnapshot() {
	// 采集快照（独立超时）
	collectCtx, cancelCollect := context.WithTimeout(s.ctx, s.config.SnapshotTimeout)
	snapshot, err := s.snapshotSvc.Collect(collectCtx)
	cancelCollect()
	if err != nil {
		log.Error("采集快照失败", "err", err)
		return
	}

	// 推送到 Master（独立超时，不受 Collect 耗时影响）
	pushCtx, cancelPush := context.WithTimeout(s.ctx, s.config.SnapshotPushTimeout)
	defer cancelPush()
	if err := s.masterGw.PushSnapshot(pushCtx, snapshot); err != nil {
		log.Error("推送快照失败", "err", err)
		return
	}

	// 检测资源数量变化
	podCount := len(snapshot.Pods)
	nodeCount := len(snapshot.Nodes)
	deploymentCount := len(snapshot.Deployments)

	if podCount != s.lastPodCount || nodeCount != s.lastNodeCount || deploymentCount != s.lastDeploymentCount {
		// 资源数量变化，输出 INFO 日志
		log.Info("快照已推送（资源变化）",
			"pods", podCount,
			"nodes", nodeCount,
			"deployments", deploymentCount,
		)
		s.lastPodCount = podCount
		s.lastNodeCount = nodeCount
		s.lastDeploymentCount = deploymentCount
	} else {
		// 无变化，输出 DEBUG 日志
		log.Debug("快照已推送",
			"pods", podCount,
			"nodes", nodeCount,
			"deployments", deploymentCount,
		)
	}
}

// =============================================================================
// 指令轮询循环
// =============================================================================

// runCommandLoop 指令轮询循环
//
// 每个 topic 独立运行一个循环，互不阻塞。
// 工作流程:
//  1. 长轮询获取 Master 指令 (超时 60s)
//  2. 执行每个指令
//  3. 上报执行结果
//  4. 有指令时立即再 poll（排空队列），无指令时等待间隔
func (s *Scheduler) runCommandLoop(topic string) {
	defer s.wg.Done()

	backoff := s.config.CommandPollInterval
	maxBackoff := 30 * time.Second
	var failCount int
	var lastWarnTime time.Time

	for {
		select {
		case <-s.ctx.Done():
			return
		default:
			ok, connErr := s.pollAndExecuteCommands(topic)
			if ok {
				backoff = s.config.CommandPollInterval
				failCount = 0
				continue // 有指令，立即再 poll（排空队列）
			}

			if connErr {
				failCount++
				// 首次失败 + 之后每 30s 打一次 WARN，避免刷屏
				if failCount == 1 || time.Since(lastWarnTime) >= 30*time.Second {
					log.Warn("拉取指令失败，等待重连", "topic", topic, "consecutiveFails", failCount)
					lastWarnTime = time.Now()
				}
				// 指数退避: 100ms → 200ms → 400ms → ... → 30s
				backoff = min(backoff*2, maxBackoff)
			} else {
				backoff = s.config.CommandPollInterval
				if failCount > 0 {
					log.Info("指令轮询恢复", "topic", topic, "previousFails", failCount)
					failCount = 0
				}
			}

			select {
			case <-s.ctx.Done():
				return
			case <-time.After(backoff):
			}
		}
	}
}

// pollAndExecuteCommands 轮询并执行指令
// 返回 (hasCommands, isConnError)
func (s *Scheduler) pollAndExecuteCommands(topic string) (bool, bool) {
	ctx, cancel := context.WithTimeout(s.ctx, s.config.CommandPollTimeout)
	defer cancel()

	// 拉取指令
	commands, err := s.masterGw.PollCommands(ctx, topic)
	if err != nil {
		return false, true
	}

	if len(commands) == 0 {
		return false, false
	}

	// 并发执行所有指令
	var wg sync.WaitGroup
	for i := range commands {
		cmd := &commands[i]
		wg.Add(1)
		go func() {
			defer wg.Done()
			start := time.Now()
			result := s.commandSvc.Execute(ctx, cmd)
			elapsed := time.Since(start)

			// 构建可读的 action 标识
			action := cmd.Action
			if sub, ok := cmd.Params["sub_action"].(string); ok && sub != "" {
				action += "/" + sub
			}

			// 上报结果
			if err := s.masterGw.ReportResult(ctx, result); err != nil {
				log.Error("指令失败", "action", action, "elapsed", elapsed.Round(time.Millisecond), "err", err)
			} else {
				log.Info("指令完成", "action", action, "elapsed", elapsed.Round(time.Millisecond), "success", result.Success)
			}
		}()
	}
	wg.Wait()
	return true, false
}

// =============================================================================
// 心跳循环
// =============================================================================

// runHeartbeatLoop 心跳循环
//
// 定时向 Master 发送心跳，让 Master 知道 Agent 存活
func (s *Scheduler) runHeartbeatLoop() {
	defer s.wg.Done()

	ticker := time.NewTicker(s.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(s.ctx, s.config.HeartbeatTimeout)
			if err := s.masterGw.Heartbeat(ctx); err != nil {
				log.Warn("心跳发送失败", "err", err)
			}
			cancel()
		}
	}
}
