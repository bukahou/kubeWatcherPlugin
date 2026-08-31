"use client";

import { useState, useEffect, useMemo, useCallback } from "react";
import { Layout } from "@/components/layout/Layout";
import { useI18n } from "@/i18n/context";
import { useClusterStore } from "@/store/clusterStore";
import {
  RefreshCw,
  Server,
  AlertTriangle,
  Loader2,
  WifiOff,
} from "lucide-react";

// 组件
import {
  ClusterOverviewChart,
  SummaryCard,
  HardwareSummaryTiles,
  HardwareMatrix,
} from "./components";

// OTel 可用性守卫
import { OTelGuard } from "@/components/observe/OTelGuard";

// 数据源代理层（自动切换 mock / api）
import {
  getClusterNodeMetrics,
  getNodeMetricsHistory,
  getHardwareHealth,
} from "@/datasource/metrics";
import type { Summary } from "@/datasource/metrics";

import type { NodeMetrics, Point } from "@/types/node-metrics";
import type { HardwareHealth } from "@/types/hardware";
import { useObserveTimeRange } from "@/hooks/useObserveTimeRange";
import { useSignalFreshness } from "@/hooks/useSignalFreshness";
import { SignalFreshnessBadge } from "@/components/observe/SignalFreshnessBadge";
import { TimeRangePicker } from "@/components/common";
import { toSpanMs } from "@/lib/time-range";

// ==================== 主页面 ====================
export default function MetricsPage() {
  return (
    <OTelGuard>
      <MetricsPageContent />
    </OTelGuard>
  );
}

function MetricsPageContent() {
  const { t } = useI18n();
  const nm = t.nodeMetrics;
  const { currentClusterId } = useClusterStore();

  const [expandedNode, setExpandedNode] = useState<string | null>(null);
  const [lastUpdate, setLastUpdate] = useState<Date>(new Date());
  const [isRefreshing, setIsRefreshing] = useState(false);

  // 数据状态
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);
  const [summary, setSummary] = useState<Summary | null>(null);
  const [nodes, setNodes] = useState<NodeMetrics[]>([]);
  const [hardware, setHardware] = useState<HardwareHealth | null>(null);

  // 历史数据缓存（每个节点，按 metric 分组）
  const [historyCache, setHistoryCache] = useState<Record<string, Record<string, Point[]>>>({});

  // P3 统一：全页唯一时间控制点，作用于所有趋势图（集群趋势 + 节点展开趋势）。
  // 快照类内容（矩阵/速览）天然是当前值，不受时间范围影响 —— 这是快照的定义，无需提示。
  const { selection: timeSelection, setSelection: setTimeSelection } = useObserveTimeRange("trendOnly");
  const freshness = useSignalFreshness("metrics");
  const historyHours = Math.max(1, Math.round(toSpanMs(timeSelection) / 3_600_000));

  // 加载数据
  const loadData = useCallback(async (showLoading = true) => {
    if (!currentClusterId) return;

    if (showLoading) setIsRefreshing(true);

    try {
      // 硬件健康与节点指标来自同一份快照，并行取；硬件接口失败不应拖垮整页
      const [result, hw] = await Promise.all([
        getClusterNodeMetrics(currentClusterId),
        getHardwareHealth(currentClusterId).catch(() => null),
      ]);
      setSummary(result.summary);
      setNodes(result.nodes);
      setHardware(hw);
      setError(null);
      setLastUpdate(new Date());
    } finally {
      setLoading(false);
      setIsRefreshing(false);
    }
  }, [currentClusterId]);

  // 初始加载
  useEffect(() => {
    loadData();
  }, [loadData]);

  // 时间范围变了，已缓存的趋势数据作废
  useEffect(() => {
    setHistoryCache({});
  }, [historyHours]);

  // 自动刷新 (10秒)
  useEffect(() => {
    const interval = setInterval(() => {
      loadData(false);
    }, 10000);
    return () => clearInterval(interval);
  }, [loadData]);

  // 手动刷新
  const handleRefresh = () => {
    loadData(true);
  };

  // 节点展开/收起
  const handleNodeToggle = useCallback((nodeName: string) => {
    const isExpanding = expandedNode !== nodeName;
    setExpandedNode(isExpanding ? nodeName : null);

    // 展开时加载历史数据（如果尚未缓存）
    if (isExpanding && !historyCache[nodeName] && currentClusterId) {
      getNodeMetricsHistory(currentClusterId, nodeName, historyHours).then(historyResult => {
        setHistoryCache(prev => ({
          ...prev,
          [nodeName]: historyResult.data,
        }));
      });
    }
  }, [expandedNode, historyCache, currentClusterId, historyHours]);

  // 计算告警节点数
  const warningNodes = useMemo(() => {
    return nodes.filter((node) => {
      return (
        node.cpu.usagePct >= 80 ||
        node.memory.usagePct >= 80 ||
        node.temperature.cpuTempC >= 75
      );
    }).length;
  }, [nodes]);

  // Loading 状态
  if (loading) {
    return (
      <Layout>
        <div className="flex items-center justify-center h-96">
          <Loader2 className="w-8 h-8 animate-spin text-blue-500" />
        </div>
      </Layout>
    );
  }

  // 无集群选中
  if (!currentClusterId) {
    return (
      <Layout>
        <div className="flex flex-col items-center justify-center h-96 text-center">
          <WifiOff className="w-12 h-12 mb-4 text-muted" />
          <p className="text-default font-medium mb-2">{nm.noCluster}</p>
          <p className="text-sm text-muted">{nm.noClusterDesc}</p>
        </div>
      </Layout>
    );
  }

  // 错误状态
  if (error && nodes.length === 0) {
    return (
      <Layout>
        <div className="flex flex-col items-center justify-center h-96 text-center">
          <AlertTriangle className="w-12 h-12 mb-4 text-yellow-500" />
          <p className="text-default font-medium mb-2">{error}</p>
          <button
            onClick={handleRefresh}
            className="mt-4 px-4 py-2 bg-blue-500 text-white rounded-lg hover:bg-blue-600 transition-colors"
          >
            {nm.retry}
          </button>
        </div>
      </Layout>
    );
  }

  return (
    <Layout>
      <div className="space-y-4 sm:space-y-6">
        {/* 标题栏 */}
        <div className="flex items-center justify-between">
          <div>
            <h1 className="text-lg sm:text-xl font-bold text-default">{t.nav.metrics}</h1>
            <p className="text-xs sm:text-sm text-muted mt-1">
              {nm.pageDescription}
            </p>
          </div>
          <div className="flex items-center gap-2 sm:gap-3 flex-shrink-0">
            <SignalFreshnessBadge item={freshness} />
            <TimeRangePicker value={timeSelection} onChange={setTimeSelection} t={nm} />
            <span className="text-[10px] sm:text-xs text-muted hidden sm:block">
              {nm.lastUpdate}: {lastUpdate.toLocaleTimeString()}
            </span>
            <button
              onClick={handleRefresh}
              disabled={isRefreshing}
              className="p-2 rounded-lg hover:bg-[var(--hover-bg)] text-muted hover:text-default transition-colors disabled:opacity-50"
            >
              <RefreshCw className={`w-4 h-4 ${isRefreshing ? "animate-spin" : ""}`} />
            </button>
          </div>
        </div>

        {/* 快照带：集群规模/告警 + 硬件速览，合并为一排（只留有判定语义的）
            —— 平均 CPU/内存两张纯快照已删：矩阵每行有精确值，均值无行动价值 */}
        <div className="grid grid-cols-2 sm:grid-cols-4 lg:grid-cols-7 gap-3">
          {summary && (
            <>
              <SummaryCard
                icon={Server}
                label={nm.summary.nodes}
                value={`${summary.onlineNodes}/${summary.totalNodes}`}
                subValue={`${warningNodes} ${nm.summary.warnings}`}
                color="bg-indigo-500/10 text-indigo-500"
              />
              <SummaryCard
                icon={AlertTriangle}
                label={nm.summary.warnings}
                value={warningNodes.toString()}
                subValue={nm.summary.nodesNeedAttention}
                color={warningNodes > 0 ? "bg-yellow-500/10 text-yellow-500" : "bg-emerald-500/10 text-emerald-500"}
              />
            </>
          )}
          <HardwareSummaryTiles data={hardware} bare />
        </div>

        {/* 统一节点表：判定矩阵 + 行点击下钻（原 NodeCard 详情体） */}
        {nodes.length === 0 ? (
          <div className="text-center py-12 bg-card rounded-xl border border-[var(--border-color)]">
            <Server className="w-12 h-12 mx-auto mb-3 text-muted opacity-50" />
            <p className="text-default font-medium mb-2">{nm.noMetricsData}</p>
            <p className="text-sm text-muted">{nm.noMetricsDesc}</p>
          </div>
        ) : (
          <HardwareMatrix
            data={hardware}
            nodes={nodes}
            expandedNode={expandedNode}
            onToggle={handleNodeToggle}
            historyCache={historyCache}
            spanMs={toSpanMs(timeSelection)}
          />
        )}

        {/* 集群趋势（时间范围跟随页面级选择器） */}
        {nodes.length > 1 && currentClusterId && (
          <ClusterOverviewChart nodes={nodes} clusterId={currentClusterId} hours={historyHours} />
        )}
      </div>
    </Layout>
  );
}
