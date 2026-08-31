"use client";

import { memo } from "react";
import { Thermometer, HardDrive, Zap, Gauge, Timer } from "lucide-react";
import { useI18n } from "@/i18n/context";
import { SummaryCard } from "./SummaryCard";
import type { HardwareHealth, HardwareStatus } from "@/types/hardware";

interface HardwareSummaryTilesProps {
  data: HardwareHealth | null;
  /** bare: 不自带 grid 容器，由父级统一排版（快照带合并为一排） */
  bare?: boolean;
}

/** 状态 → SummaryCard 的配色。计数类 tile 用「>0 即注意」的同一套语义 */
const tone = (s: HardwareStatus) =>
  s === "crit"
    ? "bg-red-500/10 text-red-500"
    : s === "warn"
      ? "bg-yellow-500/10 text-yellow-500"
      : "bg-emerald-500/10 text-emerald-500";

const countTone = (n: number, critical: boolean) =>
  n === 0
    ? "bg-emerald-500/10 text-emerald-500"
    : critical
      ? "bg-red-500/10 text-red-500"
      : "bg-yellow-500/10 text-yellow-500";

export const HardwareSummaryTiles = memo(function HardwareSummaryTiles({
  data,
  bare,
}: HardwareSummaryTilesProps) {
  const { t } = useI18n();
  const hw = t.nodeMetrics.hardware;
  const s = data?.summary;

  const maxTemp = s?.maxTemp ?? null;
  const maxDiskTemp = s?.maxDiskTemp ?? null;
  const maxAwait = s?.maxDiskAwait ?? null;

  const tiles = (
    <>
      <SummaryCard
        icon={Thermometer}
        label={hw.tiles.maxTemp}
        value={maxTemp ? `${maxTemp.value.toFixed(1)}°C` : hw.noData}
        subValue={maxTemp ? `${maxTemp.nodeName} · ${maxTemp.sensor}` : undefined}
        color={maxTemp ? tone(maxTemp.status) : "bg-[var(--background)] text-muted"}
      />
      <SummaryCard
        icon={HardDrive}
        label={hw.tiles.maxDiskTemp}
        value={maxDiskTemp ? `${maxDiskTemp.value.toFixed(1)}°C` : hw.noData}
        subValue={maxDiskTemp ? `${maxDiskTemp.nodeName} · ${maxDiskTemp.sensor}` : undefined}
        color={maxDiskTemp ? tone(maxDiskTemp.status) : "bg-[var(--background)] text-muted"}
      />
      <SummaryCard
        icon={Zap}
        label={hw.tiles.undervoltNodes}
        value={s ? `${s.undervoltNodes}` : hw.noData}
        subValue={hw.tiles.nodes}
        color={countTone(s?.undervoltNodes ?? 0, true)}
      />
      <SummaryCard
        icon={Gauge}
        label={hw.tiles.throttledNodes}
        value={s ? `${s.throttledNodes}` : hw.noData}
        subValue={hw.tiles.nodes}
        color={countTone(s?.throttledNodes ?? 0, false)}
      />
      <SummaryCard
        icon={Timer}
        label={hw.tiles.maxDiskAwait}
        value={maxAwait ? `${maxAwait.valueMs.toFixed(1)} ms` : hw.noData}
        subValue={maxAwait ? `${maxAwait.nodeName} · ${maxAwait.device}` : undefined}
        color={maxAwait ? tone(maxAwait.status) : "bg-[var(--background)] text-muted"}
      />
    </>
  );

  if (bare) return tiles;
  return <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-5 gap-3">{tiles}</div>;
});
