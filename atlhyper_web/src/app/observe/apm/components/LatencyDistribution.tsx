"use client";

import { useRef, useEffect } from "react";
import * as echarts from "echarts";
import { useI18n } from "@/i18n/context";
import type { LatencyBucket } from "@/types/model/apm";

function getThemeColors() {
  const isDark = document.documentElement.classList.contains("dark");
  return {
    textColor: isDark ? "#9ca3af" : "#6b7280",
    lineColor: isDark ? "#374151" : "#e5e7eb",
    splitLineColor: isDark ? "#1f2937" : "#f3f4f6",
    tooltipBg: isDark ? "#1f2937" : "#fff",
    tooltipBorder: isDark ? "#374151" : "#e5e7eb",
    tooltipText: isDark ? "#e5e7eb" : "#111827",
    barColor: isDark ? "#60a5fa" : "#93c5fd",
    highlightColor: "#22c55e",
  };
}

function formatBucketLabel(ms: number): string {
  if (ms < 1) return `${(ms * 1000).toFixed(0)}μs`;
  if (ms < 1000) {
    return ms % 1 === 0 ? `${ms}ms` : `${ms.toFixed(1)}ms`;
  }
  const s = ms / 1000;
  return s % 1 === 0 ? `${s}s` : `${s.toFixed(1)}s`;
}

interface LatencyDistributionProps {
  title: string;
  totalTraces: number;
  buckets: LatencyBucket[];
  highlightBucket?: number;
  /** 拖选延迟区间（ES 同款 brush 下钻）；null = 清除选区。不传则只读。 */
  onSelectRange?: (range: { startMs: number; endMs: number } | null) => void;
}

export function LatencyDistribution({
  title,
  totalTraces,
  buckets,
  highlightBucket,
  onSelectRange,
}: LatencyDistributionProps) {
  const { t } = useI18n();
  const containerRef = useRef<HTMLDivElement>(null);
  const chartRef = useRef<echarts.ECharts | null>(null);
  // brush 回调里要读最新的 buckets/onSelectRange，但不想因此重建图表
  const bucketsRef = useRef(buckets);
  bucketsRef.current = buckets;
  const onSelectRef = useRef(onSelectRange);
  onSelectRef.current = onSelectRange;

  useEffect(() => {
    if (!containerRef.current) return;
    const chart = echarts.init(containerRef.current);
    chartRef.current = chart;

    // 直接拖选（不经过 toolbox 按钮）—— ES 的 click-and-drag 交互
    chart.dispatchAction({
      type: "takeGlobalCursor",
      key: "brush",
      brushOption: { brushType: "lineX", brushMode: "single" },
    });
    chart.on("brushEnd", (raw: unknown) => {
      const params = raw as { areas?: { coordRange?: [number, number] }[] };
      const cb = onSelectRef.current;
      if (!cb) return;
      const area = params.areas?.[0];
      if (!area?.coordRange) { cb(null); return; }
      const bks = bucketsRef.current;
      const lo = Math.max(0, Math.ceil(area.coordRange[0]));
      const hi = Math.min(bks.length - 1, Math.floor(area.coordRange[1]));
      if (lo > hi || !bks[lo]) { cb(null); return; }
      const end = bks[hi].rangeEnd === Infinity ? Number.MAX_VALUE : bks[hi].rangeEnd;
      cb({ startMs: bks[lo].rangeStart, endMs: end });
    });

    const handleResize = () => chart.resize();
    window.addEventListener("resize", handleResize);

    const observer = new MutationObserver(() => {
      if (!chartRef.current) return;
      const c = getThemeColors();
      chartRef.current.setOption({
        tooltip: {
          backgroundColor: c.tooltipBg,
          borderColor: c.tooltipBorder,
          textStyle: { color: c.tooltipText },
        },
        xAxis: {
          axisLine: { lineStyle: { color: c.lineColor } },
          axisLabel: { color: c.textColor },
        },
        yAxis: {
          axisLabel: { color: c.textColor },
          splitLine: { lineStyle: { color: c.splitLineColor } },
        },
      });
    });
    observer.observe(document.documentElement, {
      attributes: true,
      attributeFilter: ["class"],
    });

    return () => {
      window.removeEventListener("resize", handleResize);
      observer.disconnect();
      chart.dispose();
    };
  }, []);

  useEffect(() => {
    if (!chartRef.current || buckets.length === 0) return;
    const c = getThemeColors();

    const labels = buckets.map((b) => formatBucketLabel(b.rangeStart));
    const values = buckets.map((b) => b.count);

    const markLineData: { xAxis: number; label: { show: boolean; formatter: string; position: string; color: string; fontSize: number } }[] = [];
    if (highlightBucket !== undefined && highlightBucket >= 0 && highlightBucket < buckets.length) {
      markLineData.push({
        xAxis: highlightBucket,
        label: {
          show: true,
          formatter: t.apm.currentSample,
          position: "insideEndTop",
          color: c.highlightColor,
          fontSize: 10,
        },
      });
    }

    chartRef.current.setOption(
      {
        brush: onSelectRange
          ? {
              toolbox: [],
              xAxisIndex: 0,
              brushType: "lineX",
              brushMode: "single",
              transformable: false,
              brushStyle: {
                color: "rgba(45, 212, 191, 0.10)",
                borderColor: "rgba(45, 212, 191, 0.6)",
                borderWidth: 1,
              },
              outOfBrush: { colorAlpha: 0.35 },
            }
          : undefined,
        tooltip: {
          trigger: "axis",
          backgroundColor: c.tooltipBg,
          borderColor: c.tooltipBorder,
          textStyle: { color: c.tooltipText, fontSize: 12 },
          formatter: (params: { dataIndex: number; value: number }[]) => {
            const p = params[0];
            const bucket = buckets[p.dataIndex];
            const rangeEnd =
              bucket.rangeEnd === Infinity
                ? "+"
                : formatBucketLabel(bucket.rangeEnd);
            return `${formatBucketLabel(bucket.rangeStart)} – ${rangeEnd}<br/>${p.value} trace(s)`;
          },
        },
        grid: { top: 16, right: 24, bottom: 40, left: 45 },
        xAxis: {
          type: "category",
          data: labels,
          axisLine: { lineStyle: { color: c.lineColor } },
          axisTick: { alignWithLabel: true },
          axisLabel: {
            color: c.textColor,
            fontSize: 10,
            rotate: 0,
            interval: (index: number) => {
              const ms = buckets[index]?.rangeStart ?? 0;
              // Kibana-style: show labels at key round values (ms)
              const roundValues = [
                1, 2, 3, 4, 5, 6, 8,
                10, 20, 30, 40, 50, 60, 80,
                100, 200, 300, 400, 500, 600, 800,
                1000, 2000, 3000, 4000, 5000, 6000, 8000,
                10000, 20000, 30000, 40000, 50000,
              ];
              return roundValues.includes(ms);
            },
          },
          splitLine: { show: false },
        },
        yAxis: {
          type: "value",
          minInterval: 1,
          axisLabel: { color: c.textColor, fontSize: 10 },
          splitLine: { lineStyle: { color: c.splitLineColor } },
        },
        series: [
          {
            type: "bar",
            data: values.map((v) => Math.max(v, 0)),
            itemStyle: {
              color: c.barColor,
              borderRadius: 0,
            },
            barGap: "0%",
            barCategoryGap: "30%",
            markLine:
              markLineData.length > 0
                ? {
                    silent: true,
                    symbol: "none",
                    lineStyle: {
                      color: c.highlightColor,
                      width: 2,
                      type: "solid",
                    },
                    data: markLineData,
                  }
                : undefined,
          },
        ],
        animation: false,
      },
      true
    );
    if (onSelectRange) {
      chartRef.current.dispatchAction({
        type: "takeGlobalCursor",
        key: "brush",
        brushOption: { brushType: "lineX", brushMode: "single" },
      });
    }
  }, [buckets, highlightBucket, title, t.apm.currentSample, onSelectRange]);

  return (
    <div>
      <div className="flex items-center gap-2 mb-2">
        <h4 className="text-sm font-semibold text-primary">{title}</h4>
        <span className="text-xs text-muted">
          | {totalTraces} traces
        </span>
      </div>
      <div ref={containerRef} className="w-full h-[180px]" />
    </div>
  );
}
