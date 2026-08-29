"use client";

import { useRef, useState } from "react";
import { Target } from "lucide-react";
import { buildXLabels } from "./HistoryChart";

export interface ErrorBudgetBurnChartTranslations {
  errorBudgetBurn: string;
  current: string;
  estimatedExhaust: string;
  noData: string;
}

/** Error Budget Burn Chart -- always renders framework */
export function ErrorBudgetBurnChart({
  history,
  errorBudgetRemaining,
  t,
}: {
  history: { timestamp: string; errorBudget: number }[];
  errorBudgetRemaining: number;
  t: ErrorBudgetBurnChartTranslations;
}) {
  const svgRef = useRef<SVGSVGElement>(null);
  const [hoveredIndex, setHoveredIndex] = useState<number | null>(null);
  const values = history.map((p) => p.errorBudget);
  const hasData = values.length > 0;
  const width = 600,
    height = 160;
  const padX = 0,
    padTop = 10,
    padBottom = 25;
  const chartH = height - padTop - padBottom;
  const chartW = width - padX * 2;

  const points = values.map((v, i) => ({
    x: padX + (i / Math.max(values.length - 1, 1)) * chartW,
    y: padTop + (1 - v / 100) * chartH,
    value: v,
    timestamp: history[i].timestamp,
  }));
  const linePath = points
    .map((p, i) => `${i === 0 ? "M" : "L"} ${p.x} ${p.y}`)
    .join(" ");

  // Use actual errorBudgetRemaining from props for header display (always available)
  const currentBudget = hasData
    ? (values[values.length - 1] ?? errorBudgetRemaining)
    : errorBudgetRemaining;
  const budgetColor =
    currentBudget > 50
      ? "#10b981"
      : currentBudget > 20
        ? "#f59e0b"
        : "#ef4444";

  // Predict exhaust date via linear regression
  const n = values.length;
  let exhaustDate = "";
  if (n > 2) {
    const first = values[0];
    const last = values[n - 1];
    const rate = (first - last) / n;
    if (rate > 0) {
      const pointsToZero = last / rate;
      const hoursPerPoint = 4;
      const hoursLeft = pointsToZero * hoursPerPoint;
      const d = new Date(history[n - 1].timestamp);
      d.setHours(d.getHours() + hoursLeft);
      exhaustDate = `${d.getMonth() + 1}/${d.getDate()}`;
    }
  }

  const xLabels = buildXLabels(
    history.map((h) => h.timestamp),
    padX,
    chartW,
  );

  return (
    <div className="bg-card rounded-xl border border-[var(--border-color)] overflow-hidden">
      <div className="px-4 py-3 border-b border-[var(--border-color)] flex items-center justify-between">
        <div className="flex items-center gap-2">
          <Target className="w-4 h-4 text-primary" />
          <span className="text-sm font-medium text-default">
            {t.errorBudgetBurn}
          </span>
        </div>
        <div className="flex items-center gap-3 text-[11px]">
          <span className="text-muted">{t.current}</span>
          <span className="font-semibold" style={{ color: budgetColor }}>
            {currentBudget.toFixed(1)}%
          </span>
          {exhaustDate && (
            <>
              <span className="text-muted">{t.estimatedExhaust}</span>
              <span className="font-semibold text-red-500">
                ~{exhaustDate}
              </span>
            </>
          )}
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
              <linearGradient
                id="budget-grad"
                x1="0"
                y1="0"
                x2="0"
                y2="1"
              >
                <stop
                  offset="0%"
                  stopColor="#10b981"
                  stopOpacity="0.15"
                />
                <stop
                  offset="50%"
                  stopColor="#f59e0b"
                  stopOpacity="0.15"
                />
                <stop
                  offset="100%"
                  stopColor="#ef4444"
                  stopOpacity="0.15"
                />
              </linearGradient>
            </defs>
            {/* Background gradient */}
            <rect
              x={padX}
              y={padTop}
              width={chartW}
              height={chartH}
              fill="url(#budget-grad)"
              rx="4"
            />
            {/* Y axis grid */}
            {[0, 25, 50, 75, 100].map((v, i) => {
              const y = padTop + (1 - v / 100) * chartH;
              return (
                <g key={i}>
                  <line
                    x1={padX}
                    y1={y}
                    x2={padX + chartW}
                    y2={y}
                    stroke="#e2e8f0"
                    strokeWidth="0.5"
                    strokeDasharray="3 3"
                    className="dark:stroke-slate-700"
                  />
                  <text
                    x={padX + chartW + 4}
                    y={y + 3}
                    className="text-[9px] fill-slate-400"
                  >
                    {v}%
                  </text>
                </g>
              );
            })}
            {/* Data line */}
            {points.length > 1 && (
              <path
                d={linePath}
                fill="none"
                stroke={budgetColor}
                strokeWidth="2.5"
                strokeLinejoin="round"
              />
            )}
            {/* Prediction dashed line */}
            {exhaustDate && points.length > 1 && (
              <line
                x1={points[points.length - 1].x}
                y1={points[points.length - 1].y}
                x2={padX + chartW}
                y2={padTop + chartH}
                stroke="#ef4444"
                strokeWidth="1.5"
                strokeDasharray="6 4"
                opacity="0.6"
              />
            )}
            {/* Data points */}
            {points.map((p, i) => (
              <circle
                key={i}
                cx={p.x}
                cy={p.y}
                r={hoveredIndex === i ? 4 : 0}
                fill={budgetColor}
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
                  x={Math.min(points[hoveredIndex].x - 40, width - 85)}
                  y={Math.max(points[hoveredIndex].y - 38, padTop)}
                  width="80"
                  height="30"
                  rx="4"
                  fill="#1e293b"
                  opacity="0.95"
                />
                <text
                  x={
                    Math.min(points[hoveredIndex].x - 40, width - 85) + 40
                  }
                  y={Math.max(points[hoveredIndex].y - 38, padTop) + 13}
                  textAnchor="middle"
                  className="text-[9px] fill-white font-medium"
                >
                  {points[hoveredIndex].value.toFixed(1)}%
                </text>
                <text
                  x={
                    Math.min(points[hoveredIndex].x - 40, width - 85) + 40
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
          <Target className="w-5 h-5 mr-2 opacity-40" />
          {t.noData}
        </div>
      )}
    </div>
  );
}
