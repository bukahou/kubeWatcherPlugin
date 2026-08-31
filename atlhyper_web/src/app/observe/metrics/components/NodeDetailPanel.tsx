"use client";

import { useI18n } from "@/i18n/context";
import type { NodeMetrics, Point } from "@/types/node-metrics";
import type { HardwareRow } from "@/types/hardware";
import { ResourceChart } from "./ResourceChart";
import { CPUCard } from "./CPUCard";
import { MemoryCard } from "./MemoryCard";
import { DiskCard } from "./DiskCard";
import { NetworkCard } from "./NetworkCard";
import { TemperatureCard } from "./TemperatureCard";
import { PSICard } from "./PSICard";
import { TCPCard } from "./TCPCard";
import { SystemResourcesCard } from "./SystemResourcesCard";
import { VMStatCard } from "./VMStatCard";

// ──────────────────────────────────────────────────────────────
// 单节点详情面板（2026-08-31 指标页重构）
// ──────────────────────────────────────────────────────────────
//
// 原为 NodeCard 的展开体。重构后 NodeCard 列表整块删除 ——
// 其折叠行的 4 个数字与硬件健康矩阵三重复叠且无阈值判定；
// 本面板改由矩阵行点击展开（判定 + 下钻同一入口）。

interface NodeDetailPanelProps {
  metrics: NodeMetrics;
  historyData: Record<string, Point[]>;
  hardware: HardwareRow | null;
  /** 趋势图时间窗（毫秒），跟随页面级 TimeRangePicker */
  spanMs: number;
}

function uptimeStr(seconds: number): string {
  const d = Math.floor(seconds / 86400);
  const h = Math.floor((seconds % 86400) / 3600);
  if (d > 0) return `${d}d ${h}h`;
  const m = Math.floor((seconds % 3600) / 60);
  return h > 0 ? `${h}h ${m}m` : `${m}m`;
}

export function NodeDetailPanel({ metrics, historyData, hardware, spanMs }: NodeDetailPanelProps) {
  const { t } = useI18n();
  const nm = t.nodeMetrics;

  return (
    <div className="px-3 sm:px-4 py-3 sm:py-4 space-y-4 sm:space-y-6 bg-[var(--background)]/40">
      {/* 系统信息条 */}
      <div className="flex flex-wrap gap-x-4 gap-y-1 text-[10px] text-muted px-1">
        {metrics.nodeIP && <span>{metrics.nodeIP}</span>}
        {metrics.kernel && <span>{metrics.kernel}</span>}
        {metrics.uptime !== undefined && <span>{nm.node.uptime}: {uptimeStr(metrics.uptime)}</span>}
      </div>

      {/* 资源趋势图 */}
      <ResourceChart data={historyData} spanMs={spanMs} />

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-3 sm:gap-6">
        <CPUCard data={metrics.cpu} />
        <MemoryCard data={metrics.memory} />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-3 sm:gap-6">
        <DiskCard data={metrics.disks} />
        <NetworkCard data={metrics.networks} />
      </div>

      <TemperatureCard data={metrics.temperature} hardware={hardware} />

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-3 sm:gap-6">
        <PSICard data={metrics.psi} />
        <TCPCard tcp={metrics.tcp} softnet={metrics.softnet} />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-2 gap-3 sm:gap-6">
        <SystemResourcesCard system={metrics.system} />
        <VMStatCard data={metrics.vmstat} />
      </div>
    </div>
  );
}
