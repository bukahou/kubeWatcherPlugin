# APM 面板重构 — 任务归档

> 原设计文档: [apm-panel-redesign.md](../../design/archive/apm-panel-redesign.md)
> UI 设计稿: https://claude.ai/code/artifact/af7edbfb-7d18-46fd-a31e-5c8352e4f5e5
> 完成：2026-08-29
> 线上：agent v0.7.0 / controller v0.6.0 / web v0.7.0

## 起因

2026-08-28 排查 geass 注册 500 暴露的「排查者视角失效链」：focus 裁剪丢上游、
日志按服务过滤挡住根因、ListTraces 聚合 bug、error 红色覆盖服务色。
根因 `Data truncated for column 'auth_type'` 一直在 ClickHouse 里，UI 一个字没显示。

## Phase 1: 瀑布图视觉重构 + ListTraces 聚合修正 — ✅ (ef59e08)

- ListTraces 两段式：条件只定位 TraceId 集合，外层聚合完整 trace
  （修正前实测 1 span/1 服务，真实 3/2）；入口 span 规则抽 entrySpanRootExpr
- 颜色只编码服务（error 改红描边+角标）；focus 从裁剪改高亮
  （filterTraceForService 与 trace-utils.ts 删除）；图例控件化；
  跨服务边界标签；self-time 分段上 bar
- Go 测试 3 例 + vitest 8 例

## Phase 2: 错误证据链 — ✅ (7369095)

- SpanError 加 Source/SourceService；Agent 用同 trace ERROR 日志按
  「SpanId 精确 > 同服务 > 跨服务」回填证据（Go 测试 6 例）
- SpanLogs 全量跨服务 + 服务色点 + ERROR 行展开结构化字段

## Phase 3: 延迟分布 brush 下钻 — ✅ (b509e3c)

**实现偏差**：纯前端（ECharts brush + 已加载 200 条样本即时过滤）。
设计文档中 ListTraces 加 max_duration_ms 的方案不需要 —— 分析阶段发现
直方图数据本就来自前端对已加载样本的聚合。

## Phase 4: Correlations — ✅ (bb20d9a)

**实现偏差**：打分与 impact 分级在 Agent 而非 Master —— 分析阶段确认
APM 域 Master 是纯透传 + 缓存（无 convert 层），Agent 直出前端形态。

## 部署验收（2026-08-29，真实 register 500 trace）

| 验收项 | 结果 |
|---|---|
| 从 geass-user 进入，3 span / 2 服务 | ✅ spanCount=3 serviceCount=2 |
| Error span 证据链 | ✅ 三个 span 全部 source=trace_log，msg=Data truncated for 'auth_type' |
| Correlations（服务级 failure） | ✅ url.path=/api/auth/register lift **354×**、iOS lift **11×**、lowSample 正确标注 |

## 验收时顺手发现（转入底层能力盘点任务）

correlations 结果里 `k8s.node.name / k8s.pod.name / service.version` 全为
`(none)` —— **geass 服务的 OTel resource 未注入 K8s 元数据**，
这三个维度的相关性分析暂时无效。属于「数据链路有位置、没数据」类。

## 经验

- operation 级 failure 相关性在低流量接口上会退化（背景=前景，全部 lift 1×），
  服务级对比才有效 —— 前端挂载在服务级是对的
- QEMU 模拟 arm64 构建：单核 5.3GHz 打满、Tctl 贴 95°C、总 CPU 却只有 7%——
  已登记交叉编译优化任务
