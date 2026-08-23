package query

import (
	"os"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// ──────────────────────────────────────────────────────────────
// 指标清单与查询源码的同步守护
// ──────────────────────────────────────────────────────────────
//
// NodeExporterMetrics 是 Agent 依赖的 node-exporter 指标全集，同时被两处消费：
//   - Collector 的 keep regex 以它为准（采什么）
//   - 契约自检以它为准（查的是否都采到了）
//
// 风险在于清单与 SQL 源码各自演进后漂移：SQL 里新加了指标但没进清单 → Collector
// 不采 → 查询静默返回空（2026-08 实测 39 个指标处于此状态，四张卡片长期空白）。
// 本测试直接扫描查询源码里的 'node_*' 字面量，与清单做集合比对，两边任一漏项即失败。
//
// 扫描源码而非引用常量，是因为 metrics.go 的 SQL 是内联字符串（1100+ 行），
// 为此全部改写成常量拼接会严重损害 SQL 的可读性 —— 用测试守护比重构更合算。

var nodeMetricLiteral = regexp.MustCompile(`['"](node_[A-Za-z0-9_]+)['"]`)

// metricsSourceFiles 是所有会出现 node_* 查询字面量的源文件。
var metricsSourceFiles = []string{"metrics.go", "../summary.go"}

func scanSourceMetricNames(t *testing.T) map[string]struct{} {
	t.Helper()
	found := make(map[string]struct{})
	for _, f := range metricsSourceFiles {
		src, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("读取 %s 失败: %v", f, err)
		}
		for _, m := range nodeMetricLiteral.FindAllStringSubmatch(string(src), -1) {
			found[m[1]] = struct{}{}
		}
	}
	return found
}

func TestNodeExporterMetrics_MatchesQuerySource(t *testing.T) {
	inSource := scanSourceMetricNames(t)
	inCatalog := make(map[string]struct{}, len(NodeExporterMetrics))
	for _, m := range NodeExporterMetrics {
		inCatalog[m] = struct{}{}
	}

	var missingFromCatalog, unusedInSource []string
	for m := range inSource {
		if _, ok := inCatalog[m]; !ok {
			missingFromCatalog = append(missingFromCatalog, m)
		}
	}
	for m := range inCatalog {
		if _, ok := inSource[m]; !ok {
			unusedInSource = append(unusedInSource, m)
		}
	}
	sort.Strings(missingFromCatalog)
	sort.Strings(unusedInSource)

	if len(missingFromCatalog) > 0 {
		t.Errorf("查询源码用到但清单缺失（Collector 不会采集，查询将静默为空）:\n  %s",
			strings.Join(missingFromCatalog, "\n  "))
	}
	if len(unusedInSource) > 0 {
		t.Errorf("清单里有但查询源码未使用（采而不查，白占存储）:\n  %s",
			strings.Join(unusedInSource, "\n  "))
	}
}

func TestNodeExporterMetrics_NoDuplicates(t *testing.T) {
	seen := make(map[string]struct{})
	for _, m := range NodeExporterMetrics {
		if _, dup := seen[m]; dup {
			t.Errorf("清单重复: %s", m)
		}
		seen[m] = struct{}{}
		if !strings.HasPrefix(m, "node_") {
			t.Errorf("非 node-exporter 指标混入清单: %s", m)
		}
	}
}

// TestNodeExporterMetrics_KeepRegex 验证生成的 keep regex 能匹配清单里每一个名字，
// 且是精确匹配（不会把 node_cpu_seconds_total 误匹配成 node_cpu_seconds_total_extra）。
func TestNodeExporterMetrics_KeepRegex(t *testing.T) {
	re, err := regexp.Compile("^(" + NodeExporterKeepRegex() + ")$")
	if err != nil {
		t.Fatalf("keep regex 无法编译: %v", err)
	}
	for _, m := range NodeExporterMetrics {
		if !re.MatchString(m) {
			t.Errorf("keep regex 未匹配清单项 %s", m)
		}
	}
	if re.MatchString("node_cpu_seconds_total_extra") {
		t.Error("keep regex 发生前缀误匹配")
	}
	if re.MatchString("kube_pod_info") {
		t.Error("keep regex 不应匹配非 node_* 指标")
	}
}
