# AIOps 风险分数单位统一（根治）

> 建立前端 `lib/risk.ts` 单一信任源，彻底清理所有风险分数显示/阈值散落的 `* 100`、`>= 0.8` 等按 [0,1] 写的代码。

---

## 1. 背景

拓扑图节点 badge 显示 `4500`（实际 45% 风险），`raspi4-zero` 等风险很低的节点全被染红。根因：

- 后端 AIOps 核心层使用 `rFinal ∈ [0, 1]` 浮点
- Gateway `scale_risk.go::toPercent` 已统一 Scale 为 `[0, 100]` 整数
- 前端多数位置（TopEntities、CausalTreeNodeView）已按百分制处理
- 但 **topology 目录 + incidents 相关 3 个组件**漏改，仍按 [0,1] 写死：
  - 乘 `* 100` 造成 4500 显示
  - 阈值 `>= 0.8 / 0.5 / 0.3 / 0.1` 造成几乎所有有风险节点被染红

## 2. 历史原因

commit `86987bc feat: AIOps 风险分数统一百分制` 那次只改了 `TopEntities` 等有限文件。`topology-graph-utils.ts` 当时还不存在，是后续 refactor commit `2dff263` 从旧拓扑代码拆出，继承了旧的 `* 100` 写法但没有人再去统一。incidents 目录下的组件也被遗漏。

## 3. 根治思路

**建立单一信任源**：所有消费风险分数的地方必须通过 `atlhyper_web/src/lib/risk.ts` 的共享函数获取颜色、等级、显示字符串，禁止本地再写阈值和 `* 100`。

## 4. 单位契约（统一约定）

```
AIOps 核心层（Go aiops/risk/）: [0, 1] 浮点
Gateway 边界（scale_risk.go）:  [0, 100] 整数
前端 API 类型以下全部:          [0, 100] 整数
```

前端**禁止**再做 `* 100`、`/ 100` 或 `>= 0.8` 之类写法。

## 5. 实施清单

### 5.1 新建 `atlhyper_web/src/lib/risk.ts`

导出 API：
- `RISK_THRESHOLDS = { critical:80, warning:50, caution:30, info:10 }`
- `type RiskLevel = "critical" | "warning" | "caution" | "info" | "healthy"`
- `riskLevel(score): RiskLevel` — 分数 → 等级
- `riskColor(score): string` — 分数 → hex 颜色
- `formatRiskScore(score): string` — 统一显示（`Math.round().toString()`）
- `isRisky(score): boolean` — score >= info 阈值

### 5.2 迁移 5 个消费者

| 文件 | 改动 |
|------|------|
| `app/aiops/topology/components/topology-graph-utils.ts` | 删本地 `riskColor`；`import { riskColor, formatRiskScore, RISK_THRESHOLDS } from "@/lib/risk"`；badge 用 `formatRiskScore(rFinal)`；isAnomaly 用 `score >= RISK_THRESHOLDS.warning` |
| `app/aiops/topology/components/TopologyGraph.tsx` | tooltip `formatRiskScore(d.rFinal)` |
| `app/aiops/topology/components/NodeDetail.tsx` | 显示 `formatRiskScore(detail.rFinal)` |
| `app/aiops/incidents/components/RootCauseCard.tsx` | 显示 `formatRiskScore`；RiskBadge level 用 `riskLevel(score)` 映射（critical/warning/caution → critical/warning/low） |
| `app/aiops/incidents/components/IncidentDetailModal.tsx` | 显示 `formatRiskScore(e.rFinal)` |

### 5.3 补 API 类型注释（契约文档化）

`atlhyper_web/src/api/aiops.ts` 的 `EntityRisk.rLocal/rWeighted/rFinal` 和 `EntityRiskDetail.rFinal` 加 JSDoc `@unit 百分制 [0,100] 整数`。

### 5.4 后端注释补强（防回归）

`atlhyper_master_v2/gateway/handler/aiops/scale_risk.go` 文件头部加单位约定注释，明确：
- 核心层 `[0,1]` / Gateway 层 `[0,100]` 的分界
- 已踩过的单位混用坑（2026-04）

## 6. 文件变更清单

```
atlhyper_web/
├── src/lib/
│   └── risk.ts [新增]
├── src/api/
│   └── aiops.ts [修改 - 仅补 JSDoc]
└── src/app/aiops/
    ├── topology/components/
    │   ├── topology-graph-utils.ts [修改]
    │   ├── TopologyGraph.tsx       [修改]
    │   └── NodeDetail.tsx          [修改]
    └── incidents/components/
        ├── RootCauseCard.tsx       [修改]
        └── IncidentDetailModal.tsx [修改]

atlhyper_master_v2/
└── gateway/handler/aiops/
    └── scale_risk.go [修改 - 仅补注释]
```

## 7. 验证

- `npm run build`（或 dev 编译通过）
- 浏览器：
  - `/aiops/topology` 异常视图节点 badge 显示 `45` 而非 `4500`；染色按百分制阈值，大量红色减少
  - `/aiops/risk` 不变（原本就对）
  - `/aiops/incidents` 展示保持一致（精度简化，数值正确）

## 8. 不做的事

- ❌ 不引入 vitest/jest（YAGNI；风险工具都是纯函数，改动简单）
- ❌ 不改 TypeScript branded type（侵入性高收益低）
- ❌ 不加自定义 ESLint 规则（额外维护成本，共享函数 + 注释已足够）
- ❌ 不动后端算法和 `scale_risk.go::toPercent`（已正确）

## 9. 长期防御（由本次方案自然实现）

- 新组件如需消费风险分数，**只有**通过 `@/lib/risk` 才能拿到颜色/格式化函数
- API 类型 JSDoc 明确标注单位，IDE 能直接读到
- 后端 handler 注释强调约定，再有人改 scale 逻辑时第一眼看到坑
