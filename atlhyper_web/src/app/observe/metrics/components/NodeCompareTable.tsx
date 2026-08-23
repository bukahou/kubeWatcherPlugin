"use client";

import { memo } from "react";
import { LayoutGrid } from "lucide-react";
import { useI18n } from "@/i18n/context";
import type { NodeComparison, CompareCell, HardwareStatus } from "@/types/hardware";

interface NodeCompareTableProps {
  data: NodeComparison | null;
}

const statusText = (s: HardwareStatus) =>
  s === "crit" ? "text-red-500" : s === "warn" ? "text-yellow-500" : "text-default";

const statusChip = (s: HardwareStatus) =>
  s === "crit"
    ? "bg-red-500/10 text-red-500"
    : s === "warn"
      ? "bg-yellow-500/10 text-yellow-500"
      : "bg-green-500/10 text-green-500";

/** 硬件列与资源列之间画一条分隔，让「会烧板子的」和「只是忙」在视觉上分开 */
const RESOURCE_COLUMNS_START = "diskUsage";

export const NodeCompareTable = memo(function NodeCompareTable({ data }: NodeCompareTableProps) {
  const { t } = useI18n();
  const cmp = t.nodeMetrics.compare;
  const columns = data?.columns ?? [];
  const rows = data?.rows ?? [];

  const columnLabel = (key: string) =>
    (cmp.columns as Record<string, string>)[key] ?? key;

  /** 欠压没有单位可显示，用 i18n 文案；其他格用后端给的数字串 */
  const cellText = (key: string, cell: CompareCell) => {
    if (key === "undervolt") {
      if (cell.value === null) return cmp.noData;
      return cell.value > 0 ? cmp.alarm : cmp.normal;
    }
    if (cell.value === null) return cmp.noData;
    return cell.text ?? cell.value.toString();
  };

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-3 sm:p-5">
      <div className="flex items-center gap-2 mb-3 sm:mb-4">
        <div className="p-1.5 sm:p-2 rounded-lg bg-indigo-500/10">
          <LayoutGrid className="w-4 h-4 sm:w-5 sm:h-5 text-indigo-500" />
        </div>
        <div>
          <h3 className="text-sm sm:text-base font-semibold text-default">{cmp.title}</h3>
          <p className="text-[10px] sm:text-xs text-muted">{cmp.subtitle}</p>
        </div>
      </div>

      {rows.length === 0 ? (
        <div className="py-8 text-center text-sm text-muted">{cmp.noData}</div>
      ) : (
        <div className="overflow-x-auto">
          <table className="w-full min-w-[900px] text-left">
            <thead>
              <tr className="border-b border-[var(--border-color)]">
                <th className="py-2 pr-3 text-[11px] font-medium text-muted sticky left-0 bg-card">{cmp.node}</th>
                {columns.map((c) => (
                  <th
                    key={c}
                    className={`py-2 px-2 text-[11px] font-medium text-muted whitespace-nowrap ${c === RESOURCE_COLUMNS_START ? "border-l border-[var(--border-color)]" : ""}`}
                  >
                    {columnLabel(c)}
                  </th>
                ))}
                <th className="py-2 pl-2 text-[11px] font-medium text-muted">{cmp.overall}</th>
              </tr>
            </thead>
            <tbody>
              {rows.map((row) => (
                <tr
                  key={row.nodeName}
                  className="border-b border-[var(--border-color)] last:border-0 hover:bg-[var(--hover-bg)] transition-colors"
                >
                  <td className="py-2 pr-3 sticky left-0 bg-card">
                    <div className="text-xs sm:text-sm text-default font-medium whitespace-nowrap">{row.nodeName}</div>
                    <div className="text-[10px] text-muted">{row.profile}</div>
                  </td>
                  {columns.map((c) => {
                    const cell = row.cells[c];
                    return (
                      <td
                        key={c}
                        className={`py-2 px-2 whitespace-nowrap ${c === RESOURCE_COLUMNS_START ? "border-l border-[var(--border-color)]" : ""}`}
                      >
                        <span
                          className={`text-xs tabular-nums ${cell ? statusText(cell.status) : "text-muted"} ${cell && cell.status !== "good" ? "font-semibold" : ""}`}
                        >
                          {cell ? cellText(c, cell) : cmp.noData}
                        </span>
                      </td>
                    );
                  })}
                  <td className="py-2 pl-2">
                    <span className={`px-2 py-0.5 rounded-md text-[11px] font-medium ${statusChip(row.overall)}`}>
                      {t.nodeMetrics.hardware.status[row.overall]}
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
