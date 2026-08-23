import { useI18n } from "@/i18n/context";
import type { HardwareRow } from "@/types/hardware";
import {
  Server,
  ChevronDown,
  ChevronRight,
} from "lucide-react";

import {
  CPUCard,
  MemoryCard,
  DiskCard,
  NetworkCard,
  TemperatureCard,
  ResourceChart,
  PSICard,
  TCPCard,
  SystemResourcesCard,
  VMStatCard,
} from "./index";

import type { NodeMetrics, Point } from "@/types/node-metrics";

// ==================== 工具函数 ====================
const uptimeStr = (s: number) => {
  const d = Math.floor(s / 86400), h = Math.floor((s % 86400) / 3600);
  return d > 0 ? `${d}d ${h}h` : `${h}h`;
};

const getUsageColor = (usage: number) => {
  if (usage >= 80) return "text-red-500";
  if (usage >= 60) return "text-yellow-500";
  return "text-emerald-500";
};

const getTempColor = (t: number) => {
  if (t >= 80) return "text-red-500";
  if (t >= 65) return "text-yellow-500";
  return "text-emerald-500";
};

// ==================== 节点卡片组件 ====================
export function NodeCard({
  metrics,
  historyData,
  hardware,
  expanded,
  onToggle,
}: {
  metrics: NodeMetrics;
  historyData: Record<string, Point[]>;
  /** 该节点的硬件判定结果（/observe/metrics/hardware）；接口不可用时为 null */
  hardware: HardwareRow | null;
  expanded: boolean;
  onToggle: () => void;
}) {
  const { t } = useI18n();
  const nm = t.nodeMetrics;
  const cpuUsage = metrics.cpu.usagePct;
  const memUsage = metrics.memory.usagePct;
  const temp = metrics.temperature.cpuTempC;
  const rootDisk = metrics.disks.find(d => d.mountPoint === "/") || metrics.disks[0];
  const diskPct = rootDisk?.usagePct || 0;

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      {/* 节点摘要行 - 紧凑内联风格 */}
      <button
        onClick={onToggle}
        className="w-full flex items-center justify-between p-3 sm:p-4 hover:bg-[var(--background)] transition-colors"
      >
        <div className="flex items-center gap-3">
          {expanded ? <ChevronDown className="w-4 h-4 text-muted" /> : <ChevronRight className="w-4 h-4 text-muted" />}
          <div className="flex items-center gap-2">
            <Server className="w-4 h-4 text-indigo-500" />
            <span className="text-sm font-semibold text-default">{metrics.nodeName}</span>
            <span className="text-[10px] text-muted hidden sm:inline">{metrics.nodeIP}</span>
          </div>
        </div>
        <div className="flex items-center gap-3 sm:gap-5 text-xs">
          <span><span className="text-muted">CPU </span><span className={getUsageColor(cpuUsage)}>{cpuUsage.toFixed(1)}%</span></span>
          <span><span className="text-muted">Mem </span><span className={getUsageColor(memUsage)}>{memUsage.toFixed(1)}%</span></span>
          <span className="hidden sm:inline"><span className="text-muted">Disk </span><span className={getUsageColor(diskPct)}>{diskPct.toFixed(1)}%</span><span className="text-muted"> ({rootDisk?.mountPoint || "/"})</span></span>
          <span className="hidden sm:inline"><span className="text-muted">Temp </span><span className={getTempColor(temp)}>{temp > 0 ? `${temp.toFixed(1)}°C` : "N/A"}</span></span>
          {metrics.uptime !== undefined && (
            <span className="hidden lg:inline text-muted">up {uptimeStr(metrics.uptime)}</span>
          )}
        </div>
      </button>

      {/* 展开详情 */}
      {expanded && (
        <div className="px-3 sm:px-4 pb-3 sm:pb-4 space-y-4 sm:space-y-6">
          {/* 系统信息条 */}
          <div className="flex flex-wrap gap-x-4 gap-y-1 text-[10px] text-muted px-1">
            {metrics.kernel && <span>{metrics.kernel}</span>}
            {metrics.uptime !== undefined && <span>{nm.node.uptime}: {uptimeStr(metrics.uptime)}</span>}
          </div>

          {/* 资源趋势图 */}
          <ResourceChart data={historyData} />

          {/* 第一行：CPU + Memory */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-3 sm:gap-6">
            <CPUCard data={metrics.cpu} />
            <MemoryCard data={metrics.memory} />
          </div>

          {/* 第二行：Disk + Network */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-3 sm:gap-6">
            <DiskCard data={metrics.disks} />
            <NetworkCard data={metrics.networks} />
          </div>

          {/* 第三行：Temperature */}
          <TemperatureCard data={metrics.temperature} hardware={hardware} />

          {/* 第四行：PSI + TCP */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-3 sm:gap-6">
            <PSICard data={metrics.psi} />
            <TCPCard tcp={metrics.tcp} softnet={metrics.softnet} />
          </div>

          {/* 第五行：System Resources + VMStat */}
          <div className="grid grid-cols-1 lg:grid-cols-2 gap-3 sm:gap-6">
            <SystemResourcesCard system={metrics.system} />
            <VMStatCard data={metrics.vmstat} />
          </div>
        </div>
      )}
    </div>
  );
}
