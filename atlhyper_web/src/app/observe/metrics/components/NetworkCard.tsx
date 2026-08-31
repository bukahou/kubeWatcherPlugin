"use client";

import { memo, useMemo, useState } from "react";
import { Network, ArrowDownToLine, ArrowUpFromLine, Wifi, WifiOff, ChevronDown, ChevronRight } from "lucide-react";
import type { NodeNetwork } from "@/types/node-metrics";
import { formatBytesPS, formatNumber } from "@/lib/format";
import { useI18n } from "@/i18n/context";

interface NetworkCardProps {
  data: NodeNetwork[];
}

/** 虚拟接口前缀列表 */
// ⚠️ 换 CNI 时必须同步此表 —— 2026-08-31 走查发现它停在旧网络栈时代
// （列着 flannel/calico 却没有 Cilium 的 lxc*/cilium_*），导致每个 Pod 的
// veth 都被当物理口展示。采集层已同步 drop（collector.yaml），此表兜底旧数据。
const VIRTUAL_PREFIXES = ["veth", "docker", "br-", "cni", "flannel", "calico", "tunl", "vxlan", "lxc", "cilium_"];

/** 判断是否为虚拟/容器接口 */
function isVirtualInterface(name: string): boolean {
  if (name === "lo") return true;
  return VIRTUAL_PREFIXES.some((prefix) => name.startsWith(prefix));
}

/** Format speedBps (bps) to human-readable: 1G / 10G / 100M */
function formatSpeed(bps: number): string {
  if (bps >= 1e9) return `${bps / 1e9}G`;
  if (bps >= 1e6) return `${bps / 1e6}M`;
  return `${bps}bps`;
}

/** 单个接口条目 */
function InterfaceItem({ iface, nm }: { iface: NodeNetwork; nm: ReturnType<typeof useI18n>["t"]["nodeMetrics"] }) {
  return (
    <div className="p-2 sm:p-3 bg-[var(--background)] rounded-lg">
      {/* 接口名 & 状态 */}
      <div className="flex items-center justify-between mb-2">
        <div className="flex items-center gap-1.5 sm:gap-2 min-w-0 flex-1">
          {iface.up ? (
            <Wifi className="w-3.5 h-3.5 sm:w-4 sm:h-4 text-green-500 flex-shrink-0" />
          ) : (
            <WifiOff className="w-3.5 h-3.5 sm:w-4 sm:h-4 text-red-500 flex-shrink-0" />
          )}
          <span className="text-xs sm:text-sm font-medium text-default truncate">{iface.interface}</span>
          <span className="text-[10px] sm:text-xs text-muted truncate hidden sm:inline">MTU {iface.mtu}</span>
        </div>
        <span className="text-[10px] sm:text-xs text-muted flex-shrink-0">
          {formatSpeed(iface.speedBps)}
        </span>
      </div>

      {/* 流量 */}
      <div className="grid grid-cols-2 gap-2 sm:gap-4 mb-2">
        <div>
          <div className="flex items-center justify-between text-[10px] sm:text-xs mb-1">
            <span className="text-muted flex items-center gap-0.5 sm:gap-1">
              <ArrowDownToLine className="w-2.5 h-2.5 sm:w-3 sm:h-3 text-green-500" />
              <span className="hidden sm:inline">{nm.network.receive}</span>
              <span className="sm:hidden">{nm.network.rxShort}</span>
            </span>
            <span className="text-green-500 font-medium">{formatBytesPS(iface.rxBytesPerSec)}</span>
          </div>
          <div className="h-1 sm:h-1.5 bg-[var(--background-secondary,#1f2937)] rounded-full overflow-hidden">
            <div
              className="h-full bg-green-500 rounded-full transition-all duration-300"
              style={{ width: `${Math.min(100, iface.speedBps > 0 ? (iface.rxBytesPerSec * 8 / iface.speedBps) * 100 : 0)}%` }}
            />
          </div>
        </div>
        <div>
          <div className="flex items-center justify-between text-[10px] sm:text-xs mb-1">
            <span className="text-muted flex items-center gap-0.5 sm:gap-1">
              <ArrowUpFromLine className="w-2.5 h-2.5 sm:w-3 sm:h-3 text-blue-500" />
              <span className="hidden sm:inline">{nm.network.transmit}</span>
              <span className="sm:hidden">{nm.network.txShort}</span>
            </span>
            <span className="text-blue-500 font-medium">{formatBytesPS(iface.txBytesPerSec)}</span>
          </div>
          <div className="h-1 sm:h-1.5 bg-[var(--background-secondary,#1f2937)] rounded-full overflow-hidden">
            <div
              className="h-full bg-blue-500 rounded-full transition-all duration-300"
              style={{ width: `${Math.min(100, iface.speedBps > 0 ? (iface.txBytesPerSec * 8 / iface.speedBps) * 100 : 0)}%` }}
            />
          </div>
        </div>
      </div>

      {/* 包 & 错误 - 仅桌面端显示 */}
      <div className="hidden sm:grid grid-cols-4 gap-2 text-xs">
        <div>
          <div className="text-muted">{nm.network.rxPackets}</div>
          <div className="text-default">{formatNumber(iface.rxPktPerSec)}/s</div>
        </div>
        <div>
          <div className="text-muted">{nm.network.txPackets}</div>
          <div className="text-default">{formatNumber(iface.txPktPerSec)}/s</div>
        </div>
        <div>
          <div className="text-muted">{nm.network.errors}</div>
          <div className="text-default">
            {formatNumber(iface.rxErrPerSec + iface.txErrPerSec)}/s
          </div>
        </div>
        <div>
          <div className="text-muted">{nm.network.dropped}</div>
          <div className="text-default">
            {formatNumber(iface.rxDropPerSec + iface.txDropPerSec)}/s
          </div>
        </div>
      </div>
    </div>
  );
}

export const NetworkCard = memo(function NetworkCard({ data }: NetworkCardProps) {
  const { t } = useI18n();
  const nm = t.nodeMetrics;
  const [showVirtual, setShowVirtual] = useState(false);

  // 分离物理接口和虚拟接口
  const { physical, virtual } = useMemo(() => {
    const physical: NodeNetwork[] = [];
    const virtual: NodeNetwork[] = [];
    for (const iface of data) {
      if (isVirtualInterface(iface.interface)) {
        virtual.push(iface);
      } else {
        physical.push(iface);
      }
    }
    return { physical, virtual };
  }, [data]);

  // 总流量仅统计物理接口（排除 lo 等内部回环流量）
  const totalRxPS = physical.reduce((acc, n) => acc + n.rxBytesPerSec, 0);
  const totalTxPS = physical.reduce((acc, n) => acc + n.txBytesPerSec, 0);
  const totalErrors = data.reduce((acc, n) => acc + n.rxErrPerSec + n.txErrPerSec, 0);
  const totalDropped = data.reduce((acc, n) => acc + n.rxDropPerSec + n.txDropPerSec, 0);

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-3 sm:p-5">
      {/* 头部 */}
      <div className="flex items-center justify-between mb-3 sm:mb-4">
        <div className="flex items-center gap-2">
          <div className="p-1.5 sm:p-2 bg-blue-500/10 rounded-lg">
            <Network className="w-4 h-4 sm:w-5 sm:h-5 text-blue-500" />
          </div>
          <div>
            <h3 className="text-sm sm:text-base font-semibold text-default">{nm.network.title}</h3>
            <p className="text-[10px] sm:text-xs text-muted">{data.length} {nm.network.interfaces}</p>
          </div>
        </div>
        {/* 总流量 */}
        <div className="flex items-center gap-2 sm:gap-4">
          <div className="flex items-center gap-1">
            <ArrowDownToLine className="w-3 h-3 sm:w-4 sm:h-4 text-green-500" />
            <span className="text-xs sm:text-sm font-medium text-default">{formatBytesPS(totalRxPS)}</span>
          </div>
          <div className="flex items-center gap-1">
            <ArrowUpFromLine className="w-3 h-3 sm:w-4 sm:h-4 text-blue-500" />
            <span className="text-xs sm:text-sm font-medium text-default">{formatBytesPS(totalTxPS)}</span>
          </div>
        </div>
      </div>

      {/* 物理接口列表 */}
      <div className="space-y-2 sm:space-y-3">
        {physical.map((iface) => (
          <InterfaceItem key={iface.interface} iface={iface} nm={nm} />
        ))}
      </div>

      {/* 虚拟接口折叠区域 */}
      {virtual.length > 0 && (
        <div className="mt-2 sm:mt-3">
          <button
            onClick={() => setShowVirtual(!showVirtual)}
            className="flex items-center gap-1.5 w-full py-1.5 sm:py-2 text-[10px] sm:text-xs text-muted hover:text-default transition-colors"
          >
            {showVirtual ? (
              <ChevronDown className="w-3 h-3 sm:w-3.5 sm:h-3.5" />
            ) : (
              <ChevronRight className="w-3 h-3 sm:w-3.5 sm:h-3.5" />
            )}
            <span>
              {showVirtual ? nm.network.hideVirtual : nm.network.showVirtual}
              {" "}({virtual.length} {nm.network.virtualInterfaces})
            </span>
          </button>

          {showVirtual && (
            <div className="space-y-2 sm:space-y-3 mt-1">
              {virtual.map((iface) => (
                <InterfaceItem key={iface.interface} iface={iface} nm={nm} />
              ))}
            </div>
          )}
        </div>
      )}

      {/* 底部统计信息 - 仅桌面端显示 */}
      {(totalErrors > 0 || totalDropped > 0) && (
        <div className="hidden sm:block mt-4 pt-4 border-t border-[var(--border-color)]">
          <div className="flex items-center gap-2 text-xs text-muted">
            <span>{nm.network.cumulativeStats}:</span>
            {totalErrors > 0 && (
              <span>{formatNumber(totalErrors)}/s {nm.network.errors}</span>
            )}
            {totalErrors > 0 && totalDropped > 0 && <span>·</span>}
            {totalDropped > 0 && (
              <span>{formatNumber(totalDropped)}/s {nm.network.dropped}</span>
            )}
          </div>
        </div>
      )}
    </div>
  );
});
