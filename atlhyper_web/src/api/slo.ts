/**
 * SLO 监控 API
 */

import { get, put } from "./request";
import type {
  DomainSLOListResponse,
  DomainSLOListResponseV2,
  DomainSLODetail,
  DomainSLOHistoryResponse,
  LatencyDistributionResponse,
  SLOTarget,
  SLOStatusHistoryResponse,
  SLODomainsParams,
  SLODomainDetailParams,
  SLODomainHistoryParams,
  SLOLatencyParams,
  SLOTargetCreateParams,
  SLOStatusHistoryParams,
} from "@/types/slo";

/**
 * 获取域名 SLO 列表 (V1 - 按 service key)
 */
export const getSLODomains = (params?: SLODomainsParams) => {
  return get<DomainSLOListResponse>("/api/v2/slo/domains", {
    cluster_id: params?.clusterId,
    time_range: params?.timeRange,
  });
};

/**
 * 获取域名 SLO 列表 (V2 - 按真实域名分组)
 * 返回 domain -> routes 层级结构
 */
export const getSLODomainsV2 = (params?: SLODomainsParams) => {
  return get<DomainSLOListResponseV2>("/api/v2/slo/domains/v2", {
    cluster_id: params?.clusterId,
    time_range: params?.timeRange,
  });
};

/**
 * 获取域名 SLO 详情
 */
export const getSLODomainDetail = (params: SLODomainDetailParams) => {
  return get<DomainSLODetail>("/api/v2/slo/domains/detail", {
    cluster_id: params.clusterId,
    host: params.host,
    time_range: params.timeRange,
  });
};

/**
 * 获取域名 SLO 历史数据
 */
export const getSLODomainHistory = (params: SLODomainHistoryParams) => {
  return get<DomainSLOHistoryResponse>("/api/v2/slo/domains/history", {
    cluster_id: params.clusterId,
    host: params.host,
    time_range: params.timeRange,
  });
};

/**
 * 获取域名延迟分布（bucket + 方法分布 + 状态码分布）
 */
export const getSLOLatencyDistribution = (params: SLOLatencyParams) => {
  return get<LatencyDistributionResponse>("/api/v2/slo/domains/latency", {
    cluster_id: params.clusterId,
    domain: params.domain,
    time_range: params.timeRange,
  });
};

/**
 * 获取 SLO 目标列表
 */
export const getSLOTargets = (clusterId?: string) => {
  return get<SLOTarget[]>("/api/v2/slo/targets", clusterId ? { cluster_id: clusterId } : {});
};

/**
 * 创建/更新 SLO 目标
 */
export const upsertSLOTarget = (params: SLOTargetCreateParams) => {
  return put<{ status: string }>("/api/v2/slo/targets", {
    clusterId: params.clusterId,
    host: params.host,
    windowDays: params.windowDays,
    availabilityTarget: params.availabilityTarget,
    p95LatencyTarget: params.p95LatencyTarget,
  });
};

/**
 * 获取状态变更历史
 */
export const getSLOStatusHistory = (params?: SLOStatusHistoryParams) => {
  return get<SLOStatusHistoryResponse>("/api/v2/slo/status-history", {
    cluster_id: params?.clusterId,
    host: params?.host,
    limit: params?.limit,
  });
};
