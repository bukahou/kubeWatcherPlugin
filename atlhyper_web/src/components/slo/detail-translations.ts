// detail-translations.ts
// DomainDetail 及其子组件的翻译契约。
// 单独一个文件是因为它被 DomainDetail / BurnRateTable / GoodBadCount / LatencyTab /
// SLOTargetModal 五处共用，写在组件里会导致循环引用。

import type { BurnRateTableTranslations } from "./BurnRateTable";
import type { GoodBadCountTranslations } from "./GoodBadCount";

export interface DomainDetailTranslations
  extends BurnRateTableTranslations,
    GoodBadCountTranslations {
  tabBudget: string;
  tabLatency: string;
  configTarget: string;
  // 图表
  p95Latency: string;
  errorRate: string;
  target: string;
  sloTrend: string;
  errorBudgetBurn: string;
  current: string;
  estimatedExhaust: string;
  // 延迟 tab
  latencyDistribution: string;
  methodBreakdown: string;
  statusCodeBreakdown: string;
  clearSelection: string;
  requests: string;
  // 目标配置弹窗
  configSloTarget: string;
  targetDomain: string;
  sloWindow: string;
  sloWindowHint: string;
  days: string;
  targetAvailability: string;
  targetAvailabilityHint: string;
  targetP95: string;
  targetP95Hint: string;
  errorRateThreshold: string;
  errorRateAutoCalc: string;
  cancel: string;
  save: string;
  saving: string;
}
