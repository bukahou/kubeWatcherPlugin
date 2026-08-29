/**
 * AIOps API
 *
 * 类型定义对齐设计文档: docs/design/active/aiops-phase3-frontend.md §2
 */

import { get, post } from "./request";

// ==================== 类型定义 ====================

// 风险相关
export interface ClusterRisk {
  clusterId: string;
  risk: number; // [0, 100]
  level: string; // "healthy" | "low" | "warning" | "critical"
  topEntities: EntityRisk[];
  totalEntities: number;
  anomalyCount: number;
  updatedAt: number;
}

/**
 * EntityRisk 实体风险
 *
 * 风险分数单位：所有 rLocal / rWeighted / rFinal 字段均为
 * **百分制 [0, 100] 整数**（后端 gateway/handler/aiops/scale_risk.go
 * 已将核心层的 [0, 1] 概率 Scale 为百分制）。
 *
 * 前端消费这些字段时必须通过 `@/lib/risk` 的 formatRiskScore / riskColor
 * / riskLevel 等函数，禁止再 × 100 或按 [0, 1] 写阈值。
 */
export interface EntityRisk {
  entityKey: string;
  entityType: string; // "service" | "pod" | "node" | "ingress"
  namespace: string;
  name: string;
  /** @unit 百分制 [0, 100] 整数 — 本地风险 */
  rLocal: number;
  wTime: number;
  /** @unit 百分制 [0, 100] 整数 — 时间加权风险 */
  rWeighted: number;
  /** @unit 百分制 [0, 100] 整数 — 最终风险（含传播） */
  rFinal: number;
  riskLevel: string;
  firstAnomaly: number;
}

export interface CausalTreeNode {
  entityKey: string;
  entityType: string;
  /** @unit 百分制 [0, 100] 整数 */
  rFinal: number;
  edgeType?: string;
  direction?: string;
  metrics?: AnomalyResult[];
  children?: CausalTreeNode[];
}

export interface EntityRiskDetail extends EntityRisk {
  metrics: AnomalyResult[];
  propagation: PropagationPath[];
  causalChain: CausalEntry[];
  causalTree?: CausalTreeNode[];
}

// ── 基线（EMA 学习状态）────────────────────────────────
// 后端 aiops/baseline 引擎持续学习每个实体的指标常态：
// EMA 是「平时长什么样」，方差是波动幅度。
//
// ready=false 表示采样数未达冷启动阈值 —— 引擎此时只学习、不判异常，
// 该基线值不可信。阈值（coldStartMinCount）由后端下发而非前端硬编码：
// 复制后端常量会在阈值调整时静默失效。
export interface BaselineState {
  metricName: string;
  ema: number;
  variance: number;
  stdDev: number;
  count: number;
  consecutiveZero: number;
  updatedAt: number;
  ready: boolean;
}

export interface EntityBaseline {
  entityKey: string;
  states: BaselineState[];
  coldStartMinCount: number;
}

export interface AnomalyResult {
  entityKey: string;
  metricName: string;
  currentValue: number;
  baseline: number;
  deviation: number;
  score: number;
  isAnomaly: boolean;
  detectedAt: number;
}

export interface PropagationPath {
  from: string;
  to: string;
  edgeType: string;
  contribution: number;
}

export interface CausalEntry {
  entityKey: string;
  metricName: string;
  deviation: number;
  detectedAt: number;
}

// 依赖图相关
export interface DependencyGraph {
  clusterId: string;
  nodes: Record<string, GraphNode>;
  edges: GraphEdge[];
  updatedAt: string;
}

export interface GraphNode {
  key: string;
  type: string;
  namespace: string;
  name: string;
  metadata: Record<string, string>;
}

export interface GraphEdge {
  from: string;
  to: string;
  type: string;
  weight: number;
}

// 事件相关
export interface Incident {
  id: string;
  clusterId: string;
  state: string;
  severity: string;
  rootCause: string;
  peakRisk: number;
  startedAt: string;
  resolvedAt: string | null;
  durationS: number;
  recurrence: number;
  createdAt: string;
}

export interface IncidentDetail extends Incident {
  entities: IncidentEntity[];
  timeline: IncidentTimeline[];
}

export interface IncidentEntity {
  incidentId: string;
  entityKey: string;
  entityType: string;
  /** @unit 百分制 [0, 100] 整数 */
  rLocal: number;
  /** @unit 百分制 [0, 100] 整数 */
  rFinal: number;
  role: string;
}

export interface IncidentTimeline {
  id: number;
  incidentId: string;
  timestamp: string;
  eventType: string;
  entityKey: string;
  detail: string;
}

export interface IncidentStats {
  totalIncidents: number;
  activeIncidents: number;
  mttr: number;
  recurrenceRate: number;
  bySeverity: Record<string, number>;
  byState: Record<string, number>;
  topRootCauses: { entityKey: string; count: number }[];
}

// AI 增强
export interface SummarizeResponse {
  incidentId: string;
  summary: string;
  rootCauseAnalysis: string;
  recommendations: Recommendation[];
  similarIncidents: SimilarMatch[];
  generatedAt: number;
  reportId?: number;
  fromCache?: boolean;
}

export interface Recommendation {
  priority: number;
  action: string;
  reason: string;
  impact: string;
}

export interface SimilarMatch {
  incidentId: string;
  similarity: number;
  rootCause: string;
  occurredAt: string;
  durationS: number;
}

// AI 报告
export interface AIReport {
  id: number;
  incidentId: string;
  clusterId: string;
  role: string;
  trigger: string;
  summary: string;
  providerName: string;
  model: string;
  inputTokens: number;
  outputTokens: number;
  durationMs: number;
  createdAt: string;
}

export interface AIReportDetail extends AIReport {
  rootCauseAnalysis: string;
  recommendations: string;
  similarIncidents: string;
  investigationSteps: string;
  evidenceChain: string;
}

// 查询参数
export interface IncidentListParams {
  cluster: string;
  state?: string;
  from?: string;
  to?: string;
  limit?: number;
  offset?: number;
}

// ==================== API 方法 ====================

// 风险
export async function getClusterRisk(cluster: string): Promise<ClusterRisk> {
  const data = (await get<ClusterRisk | null>("/api/v2/aiops/risk/cluster", { cluster })).data;
  return data ?? {
    clusterId: cluster, risk: 0, level: "healthy",
    topEntities: [], totalEntities: 0, anomalyCount: 0, updatedAt: Date.now(),
  };
}

export async function getEntityRisks(cluster: string, sort = "r_final", limit = 20): Promise<EntityRisk[]> {
  return (await get<EntityRisk[] | null>("/api/v2/aiops/risk/entities", { cluster, sort, limit })).data ?? [];
}

export async function getEntityRiskDetail(cluster: string, entityKey: string): Promise<EntityRiskDetail> {
  const detail = (await get<EntityRiskDetail>("/api/v2/aiops/risk/entity", { cluster, entity: entityKey })).data;
  // Go nil slice → JSON null，兜底为空数组
  detail.metrics = detail.metrics ?? [];
  detail.propagation = detail.propagation ?? [];
  detail.causalChain = detail.causalChain ?? [];
  return detail;
}

// 实体基线（EMA 学习状态）
export async function getEntityBaseline(cluster: string, entityKey: string): Promise<EntityBaseline> {
  const data = (await get<EntityBaseline>("/api/v2/aiops/baseline", { cluster_id: cluster, entity: entityKey })).data;
  return { ...data, states: data.states ?? [] };
}

// 依赖图追踪（G3 接线，2026-08-29）：从某实体出发沿边遍历上/下游。
// ⚠️ 后端参数名是 from（非 entity），与 aiops 其余端点不一致 —— 盘点时踩过
export interface GraphTraceResult {
  nodes: { key: string; type: string; namespace: string; name: string }[];
  edges: { from: string; to: string; type: string; weight: number }[];
  depth: number;
}

export async function getGraphTrace(
  cluster: string,
  fromKey: string,
  direction: "upstream" | "downstream",
  depth = 3
): Promise<GraphTraceResult> {
  const data = (await get<GraphTraceResult>("/api/v2/aiops/graph/trace", {
    cluster_id: cluster, from: fromKey, direction, depth,
  })).data;
  return { nodes: data.nodes ?? [], edges: data.edges ?? [], depth: data.depth ?? 0 };
}

// 依赖图
export async function getGraph(cluster: string): Promise<DependencyGraph> {
  return (await get<DependencyGraph>("/api/v2/aiops/graph", { cluster })).data;
}

// 事件
export async function getIncidents(params: IncidentListParams): Promise<Incident[]> {
  const resp = (await get<{ data: Incident[]; total: number }>("/api/v2/aiops/incidents", params)).data;
  return resp?.data ?? [];
}

// 历史事件模式（G2 接线，2026-08-29）：同一实体过去反复出的事件归组。
// 回答「这个问题是不是老毛病」—— 复发次数、平均时长、上次发生时间。
export interface IncidentPattern {
  entityKey: string;
  patternCount: number;
  avgDuration: number;       // 秒
  lastOccurrence: string;
  commonMetrics: string[] | null;
  incidents: Incident[];
}

export async function getIncidentPatterns(
  cluster: string, entityKey: string, period = "30d"
): Promise<IncidentPattern[]> {
  const data = (await get<IncidentPattern[]>("/api/v2/aiops/incidents/patterns", {
    cluster_id: cluster, entity: entityKey, period,
  })).data;
  return data ?? [];
}

export async function getIncidentDetail(id: string): Promise<IncidentDetail> {
  return (await get<IncidentDetail>(`/api/v2/aiops/incidents/${encodeURIComponent(id)}`)).data;
}

export async function getIncidentStats(cluster: string, period = "7d"): Promise<IncidentStats> {
  return (await get<IncidentStats>("/api/v2/aiops/incidents/stats", { cluster, period })).data;
}

// AI 增强
export async function summarizeIncident(incidentId: string): Promise<SummarizeResponse> {
  return (await post<SummarizeResponse>("/api/v2/aiops/ai/summarize", { incidentId })).data;
}

export async function recommendActions(incidentId: string): Promise<SummarizeResponse> {
  return (await post<SummarizeResponse>("/api/v2/aiops/ai/recommend", { incidentId })).data;
}

// AI 报告
export async function getAIReports(incidentId: string): Promise<AIReport[]> {
  const resp = (await get<{ data: AIReport[] }>(`/api/v2/aiops/ai/reports`, { incident_id: incidentId })).data;
  return resp?.data ?? [];
}

export async function getAIReportDetail(reportId: number): Promise<AIReportDetail> {
  return (await get<{ data: AIReportDetail }>(`/api/v2/aiops/ai/reports/${reportId}`)).data.data;
}

export async function triggerAnalysis(incidentId: string): Promise<{ message: string; reportId: number }> {
  return (await post<{ message: string; reportId: number }>("/api/v2/aiops/ai/analyze", { incidentId })).data;
}
