// Package logger 提供统一的结构化日志功能
// 基于 Go 1.21+ log/slog 标准库
//
// 初始化时序（理解本包设计的关键）:
//
//	各业务包通常用 `var log = logger.Module("X")` 在 import 期创建模块日志器，
//	这发生在 main() 调用 logger.Init(cfg) 之前。因此本包必须满足:
//	  1. Module() 在 Init 之前可用（lazy 兜底成 text/info）
//	  2. 之后的 Init 必须对"已创建"的模块日志器生效
//
//	历史教训 (2026-08): 旧实现用 sync.Once 包住 Init，且 ModuleLogger 缓存了创建时的
//	*slog.Logger —— lazy 初始化先消耗掉 Once，main() 里带 json 配置的 Init 成为空操作，
//	*_LOG_FORMAT / *_LOG_LEVEL 环境变量从未真正生效。现在 Init 总是"最后一次调用生效"，
//	模块日志器每次输出时从全局取当前 handler。
package logger

import (
	"context"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

var (
	mu            sync.RWMutex
	defaultLogger *slog.Logger
)

// Config 日志配置
type Config struct {
	Level  string    // debug/info/warn/error，默认 info
	Format string    // text/json，默认 text
	Output io.Writer // 输出目标，默认 os.Stdout
}

// Init 初始化（或重新初始化）全局日志器。
//
// 应在 main() 加载配置后调用一次；重复调用以最后一次为准。
// 在此之前通过 Module() / 全局函数产生的日志走 lazy 默认值（text, info）。
func Init(cfg Config) {
	mu.Lock()
	defer mu.Unlock()
	initLogger(cfg)
}

// initLogger 构建 handler 并替换全局日志器。调用方必须持有 mu 写锁。
func initLogger(cfg Config) {
	level := parseLevel(cfg.Level)
	isJSON := strings.ToLower(cfg.Format) == "json"

	output := cfg.Output
	if output == nil {
		output = os.Stdout
	}

	opts := &slog.HandlerOptions{Level: level}
	if !isJSON {
		// text 是给人看的终端输出，时间简化成 HH:MM:SS 提高可读性。
		// json 是给日志采集器 (OTel filelog) 解析的，必须保留 slog 原生 RFC3339Nano，
		// 否则平台侧无法还原真实时间戳。
		opts.ReplaceAttr = func(groups []string, a slog.Attr) slog.Attr {
			if a.Key == slog.TimeKey {
				a.Value = slog.StringValue(a.Value.Time().Format("15:04:05"))
			}
			return a
		}
	}

	var handler slog.Handler
	if isJSON {
		handler = slog.NewJSONHandler(output, opts)
	} else {
		handler = slog.NewTextHandler(output, opts)
	}

	defaultLogger = slog.New(handler)
	slog.SetDefault(defaultLogger)
}

func parseLevel(s string) slog.Level {
	switch strings.ToLower(s) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// current 返回当前全局日志器；未初始化时 lazy 构建默认值（text, info）。
func current() *slog.Logger {
	mu.RLock()
	l := defaultLogger
	mu.RUnlock()
	if l != nil {
		return l
	}

	mu.Lock()
	defer mu.Unlock()
	if defaultLogger == nil {
		initLogger(Config{})
	}
	return defaultLogger
}

// Module 创建带模块标签的日志器。
//
// 返回的 ModuleLogger 只记录附加字段，不缓存 handler —— 每次输出时从全局取当前
// 日志器，因此在 import 期创建、main() 里再 Init 也能正确切换格式和级别。
func Module(name string) *ModuleLogger {
	return &ModuleLogger{attrs: []any{"module", name}}
}

// ModuleLogger 模块日志器
type ModuleLogger struct {
	attrs []any // 成对的 key/value，输出时附加到每条记录
}

// l 基于当前全局日志器生成带附加字段的 logger。
// slog.Logger.With 的开销是一次小分配，对本项目的日志量可忽略；换来的是
// "Init 对已存在的模块日志器立即生效"这一正确性保证。
func (m *ModuleLogger) l() *slog.Logger {
	return current().With(m.attrs...)
}

// Debug 调试日志（周期性任务成功、详细追踪）
func (m *ModuleLogger) Debug(msg string, args ...any) { m.l().Debug(msg, args...) }

// Info 信息日志（关键业务事件、状态变化）
func (m *ModuleLogger) Info(msg string, args ...any) { m.l().Info(msg, args...) }

// Warn 警告日志（可恢复的异常）
func (m *ModuleLogger) Warn(msg string, args ...any) { m.l().Warn(msg, args...) }

// Error 错误日志（需要关注的错误）
func (m *ModuleLogger) Error(msg string, args ...any) { m.l().Error(msg, args...) }

// With 添加上下文字段，返回新的模块日志器（原对象不变）。
func (m *ModuleLogger) With(args ...any) *ModuleLogger {
	merged := make([]any, 0, len(m.attrs)+len(args))
	merged = append(merged, m.attrs...)
	merged = append(merged, args...)
	return &ModuleLogger{attrs: merged}
}

// WithContext 从 context 中提取追踪信息（request_id / user_id）。
func (m *ModuleLogger) WithContext(ctx context.Context) *ModuleLogger {
	var extra []any
	if reqID := ctx.Value(CtxKeyRequestID); reqID != nil {
		extra = append(extra, "request_id", reqID)
	}
	if userID := ctx.Value(CtxKeyUserID); userID != nil {
		extra = append(extra, "user_id", userID)
	}
	if len(extra) == 0 {
		return m
	}
	return m.With(extra...)
}

// 上下文 Key 定义
type ctxKey string

const (
	CtxKeyRequestID ctxKey = "request_id"
	CtxKeyUserID    ctxKey = "user_id"
)

// ContextWithRequestID 在 context 中设置 request_id
func ContextWithRequestID(ctx context.Context, requestID string) context.Context {
	return context.WithValue(ctx, CtxKeyRequestID, requestID)
}

// ContextWithUserID 在 context 中设置 user_id
func ContextWithUserID(ctx context.Context, userID int64) context.Context {
	return context.WithValue(ctx, CtxKeyUserID, userID)
}

// ----- 便捷全局函数 -----

// Debug 全局调试日志
func Debug(msg string, args ...any) { current().Debug(msg, args...) }

// Info 全局信息日志
func Info(msg string, args ...any) { current().Info(msg, args...) }

// Warn 全局警告日志
func Warn(msg string, args ...any) { current().Warn(msg, args...) }

// Error 全局错误日志
func Error(msg string, args ...any) { current().Error(msg, args...) }

// ----- 辅助函数 -----

// Duration 格式化耗时为易读形式
func Duration(d time.Duration) string {
	if d < time.Millisecond {
		return d.String()
	}
	if d < time.Second {
		return d.Round(time.Microsecond).String()
	}
	return d.Round(time.Millisecond).String()
}
