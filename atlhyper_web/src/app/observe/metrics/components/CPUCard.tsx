"use client";

import { memo } from "react";
import { Cpu, Gauge } from "lucide-react";
import type { NodeCPU } from "@/types/node-metrics";
import { useI18n } from "@/i18n/context";

interface CPUCardProps {
  data: NodeCPU;
}

const getUsageTextColor = (usage: number) => {
  if (usage >= 80) return "text-red-500";
  if (usage >= 60) return "text-yellow-500";
  return "text-green-500";
};

export const CPUCard = memo(function CPUCard({ data }: CPUCardProps) {
  const { t } = useI18n();
  const nm = t.nodeMetrics;
  // 每核负载：load1 除以核数，跨机型可比（4 核的 load 4 和 12 核的 load 4 不是一回事）
  const perCoreLoad = data.cores > 0 ? data.load1 / data.cores : 0;
  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-3 sm:p-5">
      {/* 头部 */}
      <div className="flex items-center justify-between mb-3 sm:mb-4">
        <div className="flex items-center gap-2">
          <div className="p-1.5 sm:p-2 bg-orange-500/10 rounded-lg">
            <Cpu className="w-4 h-4 sm:w-5 sm:h-5 text-orange-500" />
          </div>
          <div>
            <h3 className="text-sm sm:text-base font-semibold text-default">{nm.cpu.title}</h3>
            <p className="text-[10px] sm:text-xs text-muted">{data.cores} cores</p>
          </div>
        </div>
        <div className="text-right">
          <div className={`text-xl sm:text-2xl font-bold ${getUsageTextColor(data.usagePct)}`}>
            {data.usagePct.toFixed(1)}%
          </div>
          <div className="text-[10px] sm:text-xs text-muted">{nm.cpu.usage}</div>
        </div>
      </div>

      {/* 饱和度：使用率满了只说明忙，每核负载 > 1 才说明有任务在排队等 CPU */}
      <div className="mb-3 sm:mb-4 p-2 sm:p-3 bg-[var(--background)] rounded-lg flex items-center justify-between">
        <div className="flex items-center gap-1.5">
          <Gauge className="w-3.5 h-3.5 text-muted" />
          <span className="text-[10px] sm:text-xs text-muted">{nm.cpu.saturation} · {nm.cpu.perCore}</span>
        </div>
        <span className={`text-xs sm:text-sm font-semibold tabular-nums ${perCoreLoad >= 1.5 ? "text-red-500" : perCoreLoad >= 1 ? "text-yellow-500" : "text-emerald-500"}`}>
          {perCoreLoad.toFixed(2)}
        </span>
      </div>

      {/* CPU 使用率分解 */}
      <div className="mb-3 sm:mb-4 p-2 sm:p-3 bg-[var(--background)] rounded-lg">
        <div className="grid grid-cols-3 gap-2 text-[10px] sm:text-xs">
          <div>
            <div className="text-muted">User</div>
            <div className="text-default font-medium">{data.userPct.toFixed(1)}%</div>
          </div>
          <div>
            <div className="text-muted">System</div>
            <div className="text-default font-medium">{data.systemPct.toFixed(1)}%</div>
          </div>
          <div>
            <div className="text-muted">IOWait</div>
            <div className={`font-medium ${data.iowaitPct > 10 ? "text-yellow-500" : "text-default"}`}>
              {data.iowaitPct.toFixed(1)}%
            </div>
          </div>
        </div>
      </div>

      {/* 负载平均值 - 底部 */}
      <div className="flex flex-wrap items-center gap-1.5 sm:gap-2 pt-2 sm:pt-3 border-t border-[var(--border-color)]">
        <Gauge className="w-3.5 h-3.5 sm:w-4 sm:h-4 text-muted" />
        <span className="text-[10px] sm:text-xs text-muted">{nm.cpu.load}:</span>
        <div className="flex gap-2 sm:gap-3 text-xs sm:text-sm">
          <span className="font-medium text-default">
            <span className="text-muted">1m:</span> {data.load1.toFixed(2)}
          </span>
          <span className="font-medium text-default">
            <span className="text-muted">5m:</span> {data.load5.toFixed(2)}
          </span>
          <span className="font-medium text-default hidden sm:inline">
            <span className="text-muted">15m:</span> {data.load15.toFixed(2)}
          </span>
        </div>
      </div>
    </div>
  );
});
