package log

import "testing"

// TestExpectedSeverityTexts_CoversAgentSQL 保证预期集合覆盖 Agent 聚合 SQL 里
// 实际参与计数的全部字符串 (repository/ch/query/log.go GetSummary):
//
//	countIf(SeverityText IN ('ERROR', 'SEVERE'))
//	countIf(SeverityText IN ('WARN', 'WARNING'))
//	countIf(SeverityText = 'INFO')
//	countIf(SeverityText = 'DEBUG')
//
// 两边不同步时自检会对正常数据误报, 或对漂移漏报。
func TestExpectedSeverityTexts_CoversAgentSQL(t *testing.T) {
	usedBySQL := []string{"ERROR", "SEVERE", "WARN", "WARNING", "INFO", "DEBUG"}
	set := make(map[string]struct{}, len(ExpectedSeverityTexts))
	for _, s := range ExpectedSeverityTexts {
		set[s] = struct{}{}
	}
	for _, s := range usedBySQL {
		if _, ok := set[s]; !ok {
			t.Errorf("Agent SQL 使用的 %q 不在 ExpectedSeverityTexts 中", s)
		}
	}
}

// TestExpectedSeverityTexts_UpperCaseOnly 预期值全部大写 —— slog / logback 均输出大写,
// 出现小写即视为契约漂移, 不应被预期集合吞掉。
func TestExpectedSeverityTexts_UpperCaseOnly(t *testing.T) {
	for _, s := range ExpectedSeverityTexts {
		for _, r := range s {
			if r >= 'a' && r <= 'z' {
				t.Errorf("预期值 %q 含小写字母", s)
				break
			}
		}
	}
}
