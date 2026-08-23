"use client";

import { memo } from "react";
import { Thermometer, AlertTriangle, Zap, Fan } from "lucide-react";
import type { NodeTemperature } from "@/types/node-metrics";
import type { HardwareRow, HardwareSensorCell, HardwareStatus } from "@/types/hardware";
import { useI18n } from "@/i18n/context";

interface TemperatureCardProps {
  data: NodeTemperature;
  /** 后端判定结果。为 null 时（硬件接口不可用）退化为只显示读数，不上色 */
  hardware: HardwareRow | null;
}

const statusText = (s: HardwareStatus) =>
  s === "crit" ? "text-red-500" : s === "warn" ? "text-yellow-500" : "text-green-500";

/** 传感器分组顺序固定：CPU 在最上面，这是最该先看到的一行 */
const GROUPS: HardwareSensorCell["class"][] = ["cpu", "disk", "other"];

export const TemperatureCard = memo(function TemperatureCard({ data, hardware }: TemperatureCardProps) {
  const { t } = useI18n();
  const nm = t.nodeMetrics;
  const hw = nm.hardware;

  const cpuCell = hardware?.cpuTemp ?? null;
  const cpuTemp = cpuCell?.value ?? data.cpuTempC;
  const overall: HardwareStatus = cpuCell?.status ?? "good";
  // 温度条的满量程：后端给的 warn 线，没有就退回读数量级
  const scaleMax = cpuCell?.max ?? (data.cpuMaxC > 0 ? data.cpuMaxC : 100);

  const groupLabel: Record<HardwareSensorCell["class"], string> = {
    cpu: nm.temperature.groupCpu,
    disk: nm.temperature.groupDisk,
    other: nm.temperature.groupOther,
  };

  const sensors = hardware?.sensors ?? [];

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] p-3 sm:p-5">
      {/* 头部：CPU 温度 + 阈值 */}
      <div className="flex items-center justify-between mb-3 sm:mb-4">
        <div className="flex items-center gap-2">
          <div className={`p-1.5 sm:p-2 rounded-lg ${overall === "crit" ? "bg-red-500/10" : overall === "warn" ? "bg-yellow-500/10" : "bg-cyan-500/10"}`}>
            <Thermometer className={`w-4 h-4 sm:w-5 sm:h-5 ${overall === "good" ? "text-cyan-500" : statusText(overall)}`} />
          </div>
          <div>
            <h3 className="text-sm sm:text-base font-semibold text-default">{nm.temperature.title}</h3>
            <p className="text-[10px] sm:text-xs text-muted">
              {cpuCell
                ? `${hw.threshold} ${cpuCell.max.toFixed(0)}/${cpuCell.crit.toFixed(0)}°C`
                : nm.temperature.na}
            </p>
          </div>
        </div>
        <div className="text-right">
          <div className={`text-xl sm:text-2xl font-bold tabular-nums ${cpuCell ? statusText(overall) : "text-default"}`}>
            {cpuTemp > 0 ? `${cpuTemp.toFixed(1)}°C` : nm.temperature.na}
          </div>
          <div className="text-[10px] sm:text-xs text-muted">{nm.temperature.cpuTemp}</div>
        </div>
      </div>

      {/* 温度条 */}
      <div className="mb-3 sm:mb-4">
        <div className="relative h-3 sm:h-4 bg-[var(--background)] rounded-full overflow-hidden">
          <div className="absolute inset-0 flex">
            <div className="flex-1 bg-gradient-to-r from-blue-500/20 via-green-500/20 to-green-500/20" />
            <div className="w-[15%] bg-yellow-500/20" />
            <div className="w-[10%] bg-red-500/20" />
          </div>
          <div
            className={`h-full rounded-full transition-all duration-300 ${overall === "crit" ? "bg-red-500" : overall === "warn" ? "bg-yellow-500" : "bg-green-500"}`}
            style={{ width: `${Math.min(100, (cpuTemp / scaleMax) * 100)}%`, opacity: 0.8 }}
          />
        </div>
        <div className="flex justify-between text-[10px] sm:text-xs text-muted mt-1">
          <span>0°C</span>
          <span>{Math.round(scaleMax * 0.5)}°C</span>
          <span>{Math.round(scaleMax)}°C</span>
        </div>
      </div>

      {/* 全部传感器，按 CPU / 磁盘 / 其他 分组 */}
      <div className="space-y-2">
        <div className="text-[10px] sm:text-xs text-muted">{nm.temperature.sensors}</div>
        {sensors.length === 0 ? (
          <div className="py-3 text-center text-xs text-muted">{hw.noData}</div>
        ) : (
          GROUPS.filter((g) => sensors.some((s) => s.class === g)).map((group) => (
            <div key={group}>
              <div className="text-[10px] text-muted mb-1 uppercase tracking-wide">{groupLabel[group]}</div>
              <div className="space-y-1.5">
                {sensors
                  .filter((s) => s.class === group)
                  .map((s, i) => (
                    <div key={`${s.label}-${s.sensor}-${i}`} className="flex items-center justify-between p-1.5 sm:p-2 bg-[var(--background)] rounded-lg">
                      <div className="flex-1 min-w-0">
                        <div className="text-xs sm:text-sm text-default truncate">{s.label}</div>
                        <div className="text-[10px] text-muted">{s.sensor}</div>
                      </div>
                      <div className="flex items-center gap-2">
                        <span className={`text-xs sm:text-sm font-medium tabular-nums ${statusText(s.status)}`}>
                          {s.value.toFixed(1)}°C
                        </span>
                        <span className="text-[10px] text-muted hidden sm:inline tabular-nums">
                          {s.max.toFixed(0)}/{s.crit.toFixed(0)}
                        </span>
                      </div>
                    </div>
                  ))}
              </div>
            </div>
          ))
        )}
      </div>

      {/* 供电与风扇：没有传感器的节点显示「无数据」而不是整行消失 */}
      <div className="mt-3 pt-3 border-t border-[var(--border-color)] grid grid-cols-2 gap-3">
        <div className="flex items-center gap-2">
          <Zap className={`w-3.5 h-3.5 flex-shrink-0 ${hardware?.undervolt ? statusText(hardware.undervolt.status) : "text-muted"}`} />
          <div className="min-w-0">
            <div className="text-[10px] text-muted">{nm.temperature.power}</div>
            <div className={`text-xs font-medium truncate ${hardware?.undervolt ? statusText(hardware.undervolt.status) : "text-muted"}`}>
              {hardware?.undervolt ? (hardware.undervolt.alarm ? hw.alarm : hw.normal) : hw.noData}
            </div>
          </div>
        </div>
        <div className="flex items-center gap-2">
          <Fan className={`w-3.5 h-3.5 flex-shrink-0 ${hardware?.fan ? statusText(hardware.fan.status) : "text-muted"}`} />
          <div className="min-w-0">
            <div className="text-[10px] text-muted">{nm.temperature.fanSpeed}</div>
            <div className={`text-xs font-medium truncate tabular-nums ${hardware?.fan ? statusText(hardware.fan.status) : "text-muted"}`}>
              {hardware?.fan
                ? hardware.fan.rpm === null
                  ? `${hardware.fan.state}/${hardware.fan.maxState}`
                  : hardware.fan.rpm === 0
                    ? hw.stopped
                    : `${Math.round(hardware.fan.rpm)} rpm`
                : hw.noData}
            </div>
          </div>
        </div>
      </div>

      {/* 告警提示 */}
      {overall !== "good" && (
        <div className={`mt-3 pt-3 border-t border-[var(--border-color)] flex items-center gap-2 ${statusText(overall)}`}>
          <AlertTriangle className="w-3.5 h-3.5 sm:w-4 sm:h-4 flex-shrink-0" />
          <span className="text-xs sm:text-sm">
            {overall === "crit" ? nm.temperature.critical : nm.temperature.highWarning}
          </span>
        </div>
      )}
    </div>
  );
});
