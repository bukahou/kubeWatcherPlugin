"use client";

import { memo } from "react";
import { HardDrive, RefreshCw, Skull } from "lucide-react";
import type { NodeMemory } from "@/types/node-metrics";
import { formatBytes } from "@/lib/format";
import { useI18n } from "@/i18n/context";

interface MemoryCardProps {
  data: NodeMemory;
}

const getUsageColor = (usage: number) => {
  if (usage >= 80) return "bg-red-500";
  if (usage >= 60) return "bg-yellow-500";
  return "bg-green-500";
};

const getUsageTextColor = (usage: number) => {
  if (usage >= 80) return "text-red-500";
  if (usage >= 60) return "text-yellow-500";
  return "text-green-500";
};

export const MemoryCard = memo(function MemoryCard({ data }: MemoryCardProps) {
  const { t } = useI18n();
  const nm = t.nodeMetrics;
  const usedPercent = data.usagePct;
  const usedBytes = data.totalBytes - data.availableBytes;
  const cachedPercent = (data.cachedBytes / data.totalBytes) * 100;
  const buffersPercent = (data.buffersBytes / data.totalBytes) * 100;

  // swap used = total - free
  const swapUsedBytes = data.swapTotalBytes - data.swapFreeBytes;

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-3 sm:p-5">
      {/* 头部 */}
      <div className="flex items-center justify-between mb-3 sm:mb-4">
        <div className="flex items-center gap-2">
          <div className="p-1.5 sm:p-2 bg-green-500/10 rounded-lg">
            <HardDrive className="w-4 h-4 sm:w-5 sm:h-5 text-green-500" />
          </div>
          <div>
            <h3 className="text-sm sm:text-base font-semibold text-default">{nm.memory.title}</h3>
            <p className="text-[10px] sm:text-xs text-muted">{nm.memory.total}: {formatBytes(data.totalBytes)}</p>
          </div>
        </div>
        <div className="text-right">
          <div className={`text-xl sm:text-2xl font-bold ${getUsageTextColor(usedPercent)}`}>
            {usedPercent.toFixed(1)}%
          </div>
          <div className="text-[10px] sm:text-xs text-muted">{nm.memory.usage}</div>
        </div>
      </div>

      {/* 内存使用条 (分段显示) */}
      <div className="mb-3 sm:mb-4">
        <div className="h-3 sm:h-4 bg-[var(--background)] rounded-full overflow-hidden flex">
          {/* Used (excluding cached and buffers) */}
          <div
            className="h-full bg-green-500 transition-all duration-300"
            style={{ width: `${usedPercent - cachedPercent - buffersPercent}%` }}
            title={`Used: ${formatBytes(usedBytes - data.cachedBytes - data.buffersBytes)}`}
          />
          {/* Cached */}
          <div
            className="h-full bg-blue-500 transition-all duration-300"
            style={{ width: `${cachedPercent}%` }}
            title={`Cached: ${formatBytes(data.cachedBytes)}`}
          />
          {/* Buffers */}
          <div
            className="h-full bg-purple-500 transition-all duration-300"
            style={{ width: `${buffersPercent}%` }}
            title={`Buffers: ${formatBytes(data.buffersBytes)}`}
          />
        </div>
        {/* 图例 */}
        <div className="flex items-center gap-3 sm:gap-4 mt-2 text-[10px] sm:text-xs">
          <div className="flex items-center gap-1">
            <div className="w-1.5 h-1.5 sm:w-2 sm:h-2 rounded-full bg-green-500" />
            <span className="text-muted">{nm.memory.used}</span>
          </div>
          <div className="flex items-center gap-1">
            <div className="w-1.5 h-1.5 sm:w-2 sm:h-2 rounded-full bg-blue-500" />
            <span className="text-muted">{nm.memory.cached}</span>
          </div>
          <div className="flex items-center gap-1">
            <div className="w-1.5 h-1.5 sm:w-2 sm:h-2 rounded-full bg-purple-500" />
            <span className="text-muted">{nm.memory.buffers}</span>
          </div>
        </div>
      </div>

      {/* 详细数据 */}
      <div className="grid grid-cols-2 gap-2 sm:gap-3 mb-3 sm:mb-4">
        <div className="p-2 sm:p-3 bg-[var(--background)] rounded-lg">
          <div className="text-[10px] sm:text-xs text-muted mb-0.5 sm:mb-1">{nm.memory.used}</div>
          <div className="text-xs sm:text-sm font-semibold text-default">{formatBytes(usedBytes)}</div>
        </div>
        <div className="p-2 sm:p-3 bg-[var(--background)] rounded-lg">
          <div className="text-[10px] sm:text-xs text-muted mb-0.5 sm:mb-1">{nm.memory.available}</div>
          <div className="text-xs sm:text-sm font-semibold text-default">{formatBytes(data.availableBytes)}</div>
        </div>
        <div className="p-2 sm:p-3 bg-[var(--background)] rounded-lg">
          <div className="text-[10px] sm:text-xs text-muted mb-0.5 sm:mb-1">{nm.memory.cached}</div>
          <div className="text-xs sm:text-sm font-semibold text-blue-500">{formatBytes(data.cachedBytes)}</div>
        </div>
        <div className="p-2 sm:p-3 bg-[var(--background)] rounded-lg">
          <div className="text-[10px] sm:text-xs text-muted mb-0.5 sm:mb-1">{nm.memory.buffers}</div>
          <div className="text-xs sm:text-sm font-semibold text-purple-500">{formatBytes(data.buffersBytes)}</div>
        </div>
      </div>

      {/* Swap：未启用也要显示。K8s 默认关 swap，但「关着」和「查不到」是两回事 */}
      <div className="pt-3 sm:pt-4 border-t border-[var(--border-color)]">
        <div className="flex items-center gap-2 mb-2">
          <RefreshCw className="w-3.5 h-3.5 sm:w-4 sm:h-4 text-muted" />
          <span className="text-xs sm:text-sm font-medium text-default">{nm.memory.swap}</span>
          <span className="text-[10px] sm:text-xs text-muted ml-auto">
            {data.swapTotalBytes > 0 ? `${formatBytes(swapUsedBytes)} / ${formatBytes(data.swapTotalBytes)}` : nm.memory.noSwap}
          </span>
        </div>
        <div className="h-1.5 sm:h-2 bg-[var(--background)] rounded-full overflow-hidden">
          <div
            className={`h-full rounded-full transition-all duration-300 ${getUsageColor(data.swapUsagePct)}`}
            style={{ width: `${Math.min(100, data.swapUsagePct)}%` }}
          />
        </div>
        <div className="text-[10px] sm:text-xs text-muted mt-1 text-right">
          {data.swapUsagePct.toFixed(1)}% {nm.memory.swapUsed}
        </div>
      </div>

      {/* OOM：使用率回落后现场就没了，这个计数是唯一留得住的痕迹 */}
      <div className="mt-3 pt-3 border-t border-[var(--border-color)] flex items-center justify-between">
        <div className="flex items-center gap-1.5">
          <Skull className={`w-3.5 h-3.5 ${data.oomKillTotal > 0 ? "text-red-500" : "text-muted"}`} />
          <span className="text-[10px] sm:text-xs text-muted">{nm.memory.oomKill}</span>
        </div>
        <span className={`text-xs sm:text-sm font-semibold tabular-nums ${data.oomKillTotal > 0 ? "text-red-500" : "text-emerald-500"}`}>
          {data.oomKillTotal > 0 ? data.oomKillTotal.toLocaleString() : nm.memory.oomKillNone}
        </span>
      </div>
    </div>
  );
});
