"use client";

import { memo } from "react";
import { Cpu, HardDrive, Thermometer, Zap, Fan, Gauge, Timer, ShieldCheck } from "lucide-react";
import { useI18n } from "@/i18n/context";
import type {
  HardwareHealth,
  HardwareRow,
  HardwareStatus,
} from "@/types/hardware";

interface HardwareMatrixProps {
  data: HardwareHealth | null;
}

/** 状态 → 文字色。所有判定都来自后端，这里只负责上色 */
const statusText = (s: HardwareStatus) =>
  s === "crit" ? "text-red-500" : s === "warn" ? "text-yellow-500" : "text-green-500";

/** 状态 → chip 背景 */
const statusChip = (s: HardwareStatus) =>
  s === "crit"
    ? "bg-red-500/10 text-red-500"
    : s === "warn"
      ? "bg-yellow-500/10 text-yellow-500"
      : "bg-green-500/10 text-green-500";

/** 无传感器的格子：灰底占位，与「读数为 0」明确区分 */
function NoData({ label }: { label: string }) {
  return <span className="text-[11px] text-muted opacity-60">{label}</span>;
}

/** 一格：主值 + 次要说明，颜色由 status 决定 */
function Cell({
  value,
  sub,
  status,
}: {
  value: string;
  sub?: string;
  status: HardwareStatus;
}) {
  return (
    <div className="flex flex-col">
      <span className={`text-sm font-medium tabular-nums ${statusText(status)}`}>{value}</span>
      {sub && <span className="text-[10px] text-muted truncate">{sub}</span>}
    </div>
  );
}

export const HardwareMatrix = memo(function HardwareMatrix({ data }: HardwareMatrixProps) {
  const { t } = useI18n();
  const hw = t.nodeMetrics.hardware;

  const columns: { key: string; label: string; icon: typeof Cpu }[] = [
    { key: "cpuTemp", label: hw.cpuTemp, icon: Thermometer },
    { key: "diskTemp", label: hw.diskTemp, icon: HardDrive },
    { key: "otherTemp", label: hw.otherTemp, icon: Thermometer },
    { key: "undervolt", label: hw.undervolt, icon: Zap },
    { key: "fan", label: hw.fan, icon: Fan },
    { key: "cpuFreq", label: hw.cpuFreq, icon: Gauge },
    { key: "diskAwait", label: hw.diskAwait, icon: Timer },
  ];

  const renderCell = (row: HardwareRow, key: string) => {
    switch (key) {
      case "cpuTemp":
      case "diskTemp":
      case "otherTemp": {
        const cell = row[key];
        if (!cell) return <NoData label={hw.noData} />;
        return (
          <Cell
            value={`${cell.value.toFixed(1)}°C`}
            sub={`${cell.label ?? ""} · ${hw.threshold} ${cell.max.toFixed(0)}/${cell.crit.toFixed(0)}`}
            status={cell.status}
          />
        );
      }
      case "undervolt": {
        const cell = row.undervolt;
        if (!cell) return <NoData label={hw.noData} />;
        return <Cell value={cell.alarm ? hw.alarm : hw.normal} status={cell.status} />;
      }
      case "fan": {
        const cell = row.fan;
        if (!cell) return <NoData label={hw.noData} />;
        const rpm = cell.rpm === null ? "—" : cell.rpm === 0 ? hw.stopped : `${Math.round(cell.rpm)} rpm`;
        return (
          <Cell
            value={rpm}
            sub={cell.maxState > 0 ? `${cell.state}/${cell.maxState}` : undefined}
            status={cell.status}
          />
        );
      }
      case "cpuFreq": {
        const cell = row.cpuFreq;
        if (!cell) return <NoData label={hw.noData} />;
        return (
          <Cell
            value={`${cell.currentGHz.toFixed(2)} GHz`}
            sub={`${cell.ratioPct.toFixed(0)}% · max ${cell.maxGHz.toFixed(1)}`}
            status={cell.status}
          />
        );
      }
      case "diskAwait": {
        const cell = row.diskAwait;
        if (!cell) return <NoData label={hw.noData} />;
        return (
          <Cell value={`${cell.valueMs.toFixed(1)} ms`} sub={cell.device} status={cell.status} />
        );
      }
      default:
        return <NoData label={hw.noData} />;
    }
  };

  const rows = data?.rows ?? [];

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-3 sm:p-5">
      <div className="flex items-center gap-2 mb-3 sm:mb-4">
        <div className="p-1.5 sm:p-2 rounded-lg bg-orange-500/10">
          <ShieldCheck className="w-4 h-4 sm:w-5 sm:h-5 text-orange-500" />
        </div>
        <div>
          <h3 className="text-sm sm:text-base font-semibold text-default">{hw.title}</h3>
          <p className="text-[10px] sm:text-xs text-muted">{hw.subtitle}</p>
        </div>
      </div>

      {rows.length === 0 ? (
        <div className="py-8 text-center text-sm text-muted">{hw.noData}</div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full min-w-[860px] text-left">
            <thead>
              <tr className="border-b border-[var(--border-color)]">
                <th className="py-2 pr-3 text-[11px] font-medium text-muted">{hw.node}</th>
                {columns.map((c) => (
                  <th key={c.key} className="py-2 px-3 text-[11px] font-medium text-muted">
                    <span className="inline-flex items-center gap-1">
                      <c.icon className="w-3 h-3" />
                      {c.label}
                    </span>
                  </th>
                ))}
                <th className="py-2 pl-3 text-[11px] font-medium text-muted">{hw.overall}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr
                  key={row.nodeName}
                  className="border-b border-[var(--border-color)] last:border-0 hover:bg-[var(--hover-bg)] transition-colors"
                >
                  <td className="py-2.5 pr-3">
                    <div className="text-xs sm:text-sm text-default font-medium">{row.nodeName}</div>
                    <div className="text-[10px] text-muted">{row.profileLabel}</div>
                  </td>
                  {columns.map((c) => (
                    <td key={c.key} className="py-2.5 px-3 whitespace-nowrap">
                      {renderCell(row, c.key)}
                    </td>
                  ))}
                  <td className="py-2.5 pl-3">
                    <span className={`px-2 py-0.5 rounded-md text-[11px] font-medium ${statusChip(row.overall)}`}>
                      {hw.status[row.overall]}
                    </span>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
});
