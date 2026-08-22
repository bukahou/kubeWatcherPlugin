package logger

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"
)

// resetForTest 清空全局状态，让每个用例从"未初始化"开始。
func resetForTest() {
	mu.Lock()
	defaultLogger = nil
	mu.Unlock()
}

// TestInit_AppliesAfterLazyModule 是本次修复的核心回归用例。
//
// 真实场景: 各包用 `var log = logger.Module("X")` 在 import 期创建模块日志器，
// 这比 main() 里的 logger.Init(json) 更早执行。修复前 Init 被 sync.Once 吞掉，
// 且 ModuleLogger 缓存了创建时的 handler —— 结果 *_LOG_FORMAT / *_LOG_LEVEL
// 配置从未生效过。
func TestInit_AppliesAfterLazyModule(t *testing.T) {
	resetForTest()

	mod := Module("Scheduler") // 触发 lazy 默认初始化 (text)

	var buf bytes.Buffer
	Init(Config{Level: "info", Format: "json", Output: &buf})

	mod.Info("快照已推送", "pods", 90)

	line := strings.TrimSpace(buf.String())
	if line == "" {
		t.Fatal("Init 后模块日志器没有输出到新的 Output")
	}
	var rec map[string]any
	if err := json.Unmarshal([]byte(line), &rec); err != nil {
		t.Fatalf("期望 JSON 输出，实际: %q", line)
	}
	if rec["module"] != "Scheduler" {
		t.Errorf("module 字段 = %v, want Scheduler", rec["module"])
	}
	if rec["msg"] != "快照已推送" {
		t.Errorf("msg 字段 = %v", rec["msg"])
	}
	if rec["pods"] != float64(90) {
		t.Errorf("pods 字段 = %v, want 90", rec["pods"])
	}
}

// TestInit_LevelApplied 验证级别配置在 lazy Module 之后仍能生效。
func TestInit_LevelApplied(t *testing.T) {
	resetForTest()
	mod := Module("X")

	var buf bytes.Buffer
	Init(Config{Level: "warn", Format: "text", Output: &buf})

	mod.Info("不应出现")
	mod.Warn("应出现")

	out := buf.String()
	if strings.Contains(out, "不应出现") {
		t.Errorf("info 级别未被 warn 阈值过滤: %q", out)
	}
	if !strings.Contains(out, "应出现") {
		t.Errorf("warn 日志丢失: %q", out)
	}
}

// TestJSON_TimeIsRFC3339 验证 JSON 格式保留机器可解析的完整时间戳。
//
// text 格式为可读性把时间简化成 15:04:05；JSON 是给采集器 (filelog) 解析的，
// 必须是 RFC3339，否则日志平台无法还原真实时间。
func TestJSON_TimeIsRFC3339(t *testing.T) {
	resetForTest()
	var buf bytes.Buffer
	Init(Config{Format: "json", Output: &buf})

	Module("X").Info("t")

	var rec map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &rec); err != nil {
		t.Fatalf("非 JSON: %q", buf.String())
	}
	ts, _ := rec["time"].(string)
	if _, err := time.Parse(time.RFC3339Nano, ts); err != nil {
		t.Errorf("JSON time 应为 RFC3339Nano，实际 %q: %v", ts, err)
	}
}

// TestText_TimeIsShort 验证 text 格式保持原有的可读短时间。
func TestText_TimeIsShort(t *testing.T) {
	resetForTest()
	var buf bytes.Buffer
	Init(Config{Format: "text", Output: &buf})

	Module("X").Info("t")

	out := buf.String()
	if !strings.HasPrefix(out, "time=") || len(strings.SplitN(out, " ", 2)[0]) != len("time=15:04:05") {
		t.Errorf("text 格式时间应为 time=HH:MM:SS，实际: %q", out)
	}
}

// TestModule_WithAndContext 验证 With / WithContext 的附加字段在 Init 之后仍随新 handler 输出。
func TestModule_WithAndContext(t *testing.T) {
	resetForTest()
	mod := Module("Gateway").With("component", "router")

	var buf bytes.Buffer
	Init(Config{Format: "json", Output: &buf})

	ctx := ContextWithUserID(ContextWithRequestID(context.Background(), "req-1"), 42)
	mod.WithContext(ctx).Warn("hit")

	var rec map[string]any
	if err := json.Unmarshal([]byte(strings.TrimSpace(buf.String())), &rec); err != nil {
		t.Fatalf("非 JSON: %q", buf.String())
	}
	for k, want := range map[string]any{"module": "Gateway", "component": "router", "request_id": "req-1", "user_id": float64(42), "level": "WARN"} {
		if rec[k] != want {
			t.Errorf("%s = %v, want %v", k, rec[k], want)
		}
	}
}

// TestGlobalFuncs_FollowInit 验证包级 Info/Warn 也跟随最新 Init。
func TestGlobalFuncs_FollowInit(t *testing.T) {
	resetForTest()
	Info("预热") // lazy 初始化到 stdout

	var buf bytes.Buffer
	Init(Config{Format: "json", Output: &buf})
	Error("boom", "code", 7)

	if !strings.Contains(buf.String(), `"code":7`) {
		t.Errorf("全局函数未跟随 Init 输出: %q", buf.String())
	}
}
