// atlhyper_master_v2/mq/memory/bus.go
// MemoryBus 内存消息队列实现
package memory

import (
	"context"
	"sync"
	"time"

	"AtlHyper/common/logger"
	"AtlHyper/model_v3/command"
)

var log = logger.Module("MemoryBus")

// MemoryBus 内存消息队列
type MemoryBus struct {
	// 指令队列
	queues   map[string]*commandQueue
	queuesMu sync.RWMutex

	// 指令状态（全局，用于查询）
	commands   map[string]*command.Status
	commandsMu sync.RWMutex

	// 指令结果等待者（用于同步等待）
	commandWaiters   map[string]chan *command.Result
	commandWaitersMu sync.Mutex

	// 生命周期
	stopCh chan struct{}
}

// commandQueue 单个集群的指令队列
type commandQueue struct {
	commands []*command.Command
	waiting  chan struct{} // 通知等待者
	mu       sync.Mutex
}

// NewMemoryBus 创建 MemoryBus
func NewMemoryBus() *MemoryBus {
	return &MemoryBus{
		queues:         make(map[string]*commandQueue),
		commands:       make(map[string]*command.Status),
		commandWaiters: make(map[string]chan *command.Result),
		stopCh:         make(chan struct{}),
	}
}

// Start 启动 MemoryBus
func (b *MemoryBus) Start() error {
	go b.cleanupLoop()
	log.Info("已启动")
	return nil
}

// Stop 停止 MemoryBus
func (b *MemoryBus) Stop() error {
	close(b.stopCh)
	log.Info("已停止")
	return nil
}

// queueKey 生成队列 key: clusterID:topic
func queueKey(clusterID, topic string) string {
	return clusterID + ":" + topic
}

// getOrCreateQueue 获取或创建队列
func (b *MemoryBus) getOrCreateQueue(clusterID, topic string) *commandQueue {
	key := queueKey(clusterID, topic)

	b.queuesMu.Lock()
	defer b.queuesMu.Unlock()

	if q, ok := b.queues[key]; ok {
		return q
	}
	q := &commandQueue{
		commands: make([]*command.Command, 0),
		waiting:  make(chan struct{}, 1),
	}
	b.queues[key] = q
	return q
}

// EnqueueCommand 入队指令到指定 topic
func (b *MemoryBus) EnqueueCommand(clusterID, topic string, cmd *command.Command) error {
	q := b.getOrCreateQueue(clusterID, topic)

	q.mu.Lock()
	q.commands = append(q.commands, cmd)
	q.mu.Unlock()

	// 记录指令状态
	b.commandsMu.Lock()
	b.commands[cmd.ID] = &command.Status{
		CommandID: cmd.ID,
		Status:    command.StatusPending,
		CreatedAt: cmd.CreatedAt,
	}
	b.commandsMu.Unlock()

	// 通知等待者
	select {
	case q.waiting <- struct{}{}:
	default:
	}

	log.Debug("指令已入队", "cmd", cmd.ID, "cluster", clusterID, "topic", topic)
	return nil
}

// WaitCommand 等待指定 topic 的指令（长轮询）
func (b *MemoryBus) WaitCommand(ctx context.Context, clusterID, topic string, timeout time.Duration) (*command.Command, error) {
	q := b.getOrCreateQueue(clusterID, topic)

	// 先检查是否有待处理的指令
	q.mu.Lock()
	if len(q.commands) > 0 {
		cmd := q.commands[0]
		q.commands = q.commands[1:]
		q.mu.Unlock()

		// 更新状态
		b.updateCommandStatus(cmd.ID, command.StatusRunning)
		return cmd, nil
	}
	q.mu.Unlock()

	// 没有指令，等待
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		return nil, nil // 超时，返回 nil
	case <-q.waiting:
		// 有新指令
		q.mu.Lock()
		if len(q.commands) > 0 {
			cmd := q.commands[0]
			q.commands = q.commands[1:]
			q.mu.Unlock()
			b.updateCommandStatus(cmd.ID, command.StatusRunning)
			return cmd, nil
		}
		q.mu.Unlock()
		return nil, nil
	}
}

// updateCommandStatus 更新指令状态
func (b *MemoryBus) updateCommandStatus(cmdID, status string) {
	b.commandsMu.Lock()
	defer b.commandsMu.Unlock()

	if cs, ok := b.commands[cmdID]; ok {
		cs.Status = status
		if status == command.StatusRunning {
			now := time.Now()
			cs.StartedAt = &now
		}
	}
}

// AckCommand 确认指令完成
func (b *MemoryBus) AckCommand(cmdID string, result *command.Result) error {
	b.commandsMu.Lock()

	cs, ok := b.commands[cmdID]
	if !ok {
		b.commandsMu.Unlock()
		return nil
	}

	now := time.Now()
	cs.FinishedAt = &now
	cs.Result = result

	if result.Success {
		cs.Status = command.StatusSuccess
	} else {
		cs.Status = command.StatusFailed
	}

	b.commandsMu.Unlock()

	// 唤醒等待者
	b.commandWaitersMu.Lock()
	if ch, ok := b.commandWaiters[cmdID]; ok {
		select {
		case ch <- result:
		default:
		}
		delete(b.commandWaiters, cmdID)
	}
	b.commandWaitersMu.Unlock()

	log.Debug("指令已完成", "cmd", cmdID, "status", cs.Status)
	return nil
}

// GetCommandStatus 获取指令状态
func (b *MemoryBus) GetCommandStatus(cmdID string) (*command.Status, error) {
	b.commandsMu.RLock()
	defer b.commandsMu.RUnlock()

	cs, ok := b.commands[cmdID]
	if !ok {
		return nil, nil
	}
	return cs, nil
}

// WaitCommandResult 等待指令执行完成（同步等待）
// 阻塞直到 Agent 上报结果、超时、或 ctx 取消
func (b *MemoryBus) WaitCommandResult(ctx context.Context, cmdID string, timeout time.Duration) (*command.Result, error) {
	// 先检查是否已完成
	b.commandsMu.RLock()
	if cs, ok := b.commands[cmdID]; ok && cs.Result != nil {
		b.commandsMu.RUnlock()
		return cs.Result, nil
	}
	b.commandsMu.RUnlock()

	// 创建等待 channel
	ch := make(chan *command.Result, 1)

	b.commandWaitersMu.Lock()
	b.commandWaiters[cmdID] = ch
	b.commandWaitersMu.Unlock()

	// 确保清理
	defer func() {
		b.commandWaitersMu.Lock()
		delete(b.commandWaiters, cmdID)
		b.commandWaitersMu.Unlock()
	}()

	// 等待结果、超时、或 ctx 取消
	select {
	case result := <-ch:
		return result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-time.After(timeout):
		return nil, nil // 超时返回 nil
	}
}

// cleanupLoop 定期清理已完成的指令记录
func (b *MemoryBus) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-b.stopCh:
			return
		case <-ticker.C:
			b.cleanupCompleted()
		}
	}
}

// cleanupCompleted 清理完成超过保留时间的指令记录
func (b *MemoryBus) cleanupCompleted() {
	const retention = 5 * time.Minute
	now := time.Now()

	b.commandsMu.Lock()
	defer b.commandsMu.Unlock()

	for id, cs := range b.commands {
		if cs.FinishedAt != nil && now.Sub(*cs.FinishedAt) > retention {
			delete(b.commands, id)
		}
	}
}
