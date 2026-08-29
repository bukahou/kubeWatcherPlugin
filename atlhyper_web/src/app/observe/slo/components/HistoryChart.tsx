"use client";

import { useRef, useState } from "react";
import { TrendingUp } from "lucide-react";

export interface HistoryChartTranslations {
  p95Latency: string;
  errorRate: string;
  target: string;
  sloTrend: string;
  noData: string;
}

/** Compute evenly-spaced X-axis labels from a timestamped array. */
export function buildXLabels(
  timestamps: string[],
  padLeft: number,
  chartW: number,
): { x: number; label: string }[] {
  if (timestamps.length === 0) return [];
  const first = new Date(timestamps[0]);
  const last = new Date(timestamps[timestamps.length - 1]);
  const sameDay =
    first.getFullYear() === last.getFullYear() &&
    first.getMonth() === last.getMonth() &&
    first.getDate() === last.getDate();
  const step = Math.max(1, Math.floor(timestamps.length / 6));
  const labels: { x: number; label: string }[] = [];
  for (let i = 0; i < timestamps.length; i += step) {
    const d = new Date(timestamps[i]);
    labels.push({
      x: padLeft + (i / Math.max(timestamps.length - 1, 1)) * chartW,
      label: sameDay
        ? `${d.getHours()}:${String(d.getMinutes()).padStart(2, "0")}`
        : `${d.getMonth() + 1}/${d.getDate()}`,
    });
  }
  return labels;
}

/** SLO Trend Chart -- always renders axes/grid/target; data fills in when available */
export function HistoryChart({
  history,
  targets,
  t,
}: {
  history: { timestamp: string; p95Latency: number; errorRate: number }[];
  targets?: { availability: number; p95Latency: number };
  t: HistoryChartTranslations;
}) {
  const [activeMetric, setActiveMetric] = useState<"p95" | "error">("p95");
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null);
  const svgRef = useRef<SVGSVGElement>(null);

  const metrics = [
    { id: "p95" as const, label: t.p95Latency, unit: "ms", color: "#0891b2" },
    { id: "error" as const, label: t.errorRate, unit: "%", color: "#ef4444" },
  ];
  const currentMetric = metrics.find((m) => m.id === activeMetric)!;

  const values = history.map((p) =>
    activeMetric === "p95" ? p.p95Latency : p.errorRate,
  );
  const hasData = values.length > 0;

  // SLO target value (P95 latency has target line)
  let targetVal: number | null = null;
  if (activeMetric === "p95" && targets) targetVal = targets.p95Latency;

  // Y axis range -- use defaults when no data
  let minVal: number, maxVal: number;
  if (hasData) {
    let rawMin = Math.min(...values);
    let rawMax = Math.max(...values);
    if (targetVal !== null) {
      rawMin = Math.min(rawMin, targetVal);
      rawMax = Math.max(rawMax, targetVal);
    }
    const pad = (rawMax - rawMin) * 0.05 || 1;
    minVal = Math.max(0, rawMin - pad);
    maxVal = rawMax + pad;
  } else if (targetVal !== null) {
    minVal = 0;
    maxVal = targetVal * 2 || 1;
  } else {
    minVal = 0;
    maxVal = activeMetric === "p95" ? 500 : 1;
  }
  const range = maxVal - minVal || 1;

  const width = 660,
    height = 180;
  const padLeft = 55,
    padRight = 5,
    padTop = 10,
    padBottom = 25;
  const chartH = height - padTop - padBottom;
  const chartW = width - padLeft - padRight;

  const points = values.map((v, i) => ({
    x: padLeft + (i / Math.max(values.length - 1, 1)) * chartW,
    y: padTop + (1 - (v - minVal) / range) * chartH,
    value: v,
    timestamp: history[i].timestamp,
  }));

  const linePath = points
    .map((p, i) => `${i === 0 ? "M" : "L"} ${p.x} ${p.y}`)
    .join(" ");
  const areaPath =
    points.length > 1
      ? `${linePath} L ${points[points.length - 1].x} ${padTop + chartH} L ${points[0].x} ${padTop + chartH} Z`
      : "";
  const gradientId = `hist-grad-${activeMetric}`;

  const targetY =
    targetVal !== null
      ? padTop + (1 - (targetVal - minVal) / range) * chartH
      : null;

  const formatVal = (v: number) =>
    activeMetric === "p95" ? Math.round(v) + "ms" : v.toFixed(3) + "%";

  const xLabels = buildXLabels(
    history.map((h) => h.timestamp),
    padLeft,
    chartW,
  );

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      <div className="px-4 py-3 border-b border-[var(--border-color)] flex items-center justify-between">
        <div className="flex items-center gap-2">
          <TrendingUp className="w-4 h-4 text-primary" />
          <span className="text-sm font-medium text-default">
            {t.sloTrend}
          </span>
        </div>
        <div className="flex items-center gap-1 p-0.5 rounded-lg bg-[var(--hover-bg)]">
          {metrics.map((m) => (
            <button
              key={m.id}
              onClick={() => setActiveMetric(m.id)}
              className={`px-2.5 py-1 text-[10px] rounded-md transition-colors ${
                activeMetric === m.id
                  ? "bg-card text-default shadow-sm font-medium"
                  : "text-muted hover:text-default"
              }`}
            >
              {m.label}
            </button>
          ))}
        </div>
      </div>
      {hasData ? (
        <div className="p-4">
          <svg
            ref={svgRef}
            viewBox={`0 0 ${width} ${height}`}
            className="w-full h-auto"
            onMouseLeave={() => setHoveredIndex(null)}
            onMouseMove={(e) => {
              const svg = svgRef.current;
              if (!svg) return;
              const rect = svg.getBoundingClientRect();
              const mouseX =
                ((e.clientX - rect.left) / rect.width) * width;
              let closest = 0,
                minDist = Infinity;
              points.forEach((p, i) => {
                const d = Math.abs(p.x - mouseX);
                if (d < minDist) {
                  minDist = d;
                  closest = i;
                }
              });
              setHoveredIndex(closest);
            }}
          >
            <defs>
              <linearGradient id={gradientId} x1="0" y1="0" x2="0" y2="1">
                <stop
                  offset="0%"
                  stopColor={currentMetric.color}
                  stopOpacity="0.3"
                />
                <stop
                  offset="100%"
                  stopColor={currentMetric.color}
                  stopOpacity="0.02"
                />
              </linearGradient>
            </defs>
            {/* Y axis grid + labels */}
            {[0, 0.25, 0.5, 0.75, 1].map((r, i) => {
              const y = padTop + r * chartH;
              const val = maxVal - r * range;
              return (
                <g key={i}>
                  <line
                    x1={padLeft}
                    y1={y}
                    x2={padLeft + chartW}
                    y2={y}
                    stroke="#e2e8f0"
                    strokeWidth="0.5"
                    strokeDasharray={i === 0 || i === 4 ? "0" : "3 3"}
                    className="dark:stroke-slate-700"
                  />
                  <text
                    x={padLeft - 6}
                    y={y + 3}
                    textAnchor="end"
                    className="text-[9px] fill-slate-400"
                  >
                    {formatVal(val)}
                  </text>
                </g>
              );
            })}
            {/* SLO target line */}
            {targetY !== null &&
              targetY >= padTop &&
              targetY <= padTop + chartH && (
                <g>
                  <line
                    x1={padLeft}
                    y1={targetY}
                    x2={padLeft + chartW}
                    y2={targetY}
                    stroke="#f59e0b"
                    strokeWidth="1.5"
                    strokeDasharray="6 3"
                  />
                  <text
                    x={padLeft + 4}
                    y={targetY - 4}
                    className="text-[9px] fill-amber-500 font-medium"
                  >
                    SLO {t.target}: {targetVal}
                    {currentMetric.unit}
                  </text>
                </g>
              )}
            {/* Data: area + line + points */}
            {points.length > 1 && (
              <path d={areaPath} fill={`url(#${gradientId})`} />
            )}
            {points.length > 1 && (
              <path
                d={linePath}
                fill="none"
                stroke={currentMetric.color}
                strokeWidth="2"
                strokeLinejoin="round"
              />
            )}
            {points.map((p, i) => (
              <circle
                key={i}
                cx={p.x}
                cy={p.y}
                r={hoveredIndex === i ? 4 : 0}
                fill={currentMetric.color}
                stroke="white"
                strokeWidth="2"
              />
            ))}
            {/* X axis labels */}
            {xLabels.map((l, i) => (
              <text
                key={i}
                x={l.x}
                y={height - 4}
                textAnchor="middle"
                className="text-[9px] fill-slate-400"
              >
                {l.label}
              </text>
            ))}
            {/* Hover tooltip */}
            {hoveredIndex !== null && points[hoveredIndex] && (
              <g>
                <line
                  x1={points[hoveredIndex].x}
                  y1={padTop}
                  x2={points[hoveredIndex].x}
                  y2={padTop + chartH}
                  stroke="#94a3b8"
                  strokeWidth="0.5"
                  strokeDasharray="3 3"
                />
                <rect
                  x={Math.min(points[hoveredIndex].x - 50, width - 105)}
                  y={Math.max(points[hoveredIndex].y - 38, padTop)}
                  width="100"
                  height="30"
                  rx="4"
                  fill="#1e293b"
                  opacity="0.95"
                />
                <text
                  x={
                    Math.min(points[hoveredIndex].x - 50, width - 105) + 50
                  }
                  y={Math.max(points[hoveredIndex].y - 38, padTop) + 13}
                  textAnchor="middle"
                  className="text-[9px] fill-white font-medium"
                >
                  {formatVal(points[hoveredIndex].value)}
                </text>
                <text
                  x={
                    Math.min(points[hoveredIndex].x - 50, width - 105) + 50
                  }
                  y={Math.max(points[hoveredIndex].y - 38, padTop) + 24}
                  textAnchor="middle"
                  className="text-[8px] fill-slate-400"
                >
                  {new Date(
                    points[hoveredIndex].timestamp,
                  ).toLocaleString("zh-CN", {
                    month: "numeric",
                    day: "numeric",
                    hour: "2-digit",
                    minute: "2-digit",
                  })}
                </text>
              </g>
            )}
          </svg>
        </div>
      ) : (
        <div className="flex items-center justify-center py-10 text-sm text-muted">
          <TrendingUp className="w-5 h-5 mr-2 opacity-40" />
          {t.noData}
        </div>
      )}
    </div>
  );
}
