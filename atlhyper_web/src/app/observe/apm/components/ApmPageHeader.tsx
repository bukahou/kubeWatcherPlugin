import { RefreshCw, ChevronRight } from "lucide-react";
import { TimeRangePicker } from "@/components/common";
import { SignalFreshnessBadge } from "@/components/observe/SignalFreshnessBadge";
import type { SignalFreshnessItem } from "@/types/observe";
import type { TimeRangeSelection } from "@/types/time-range";
import type { ApmTranslations, CommonTranslations } from "@/types/i18n";

type ViewState =
  | { level: "services" }
  | { level: "service-detail"; serviceName: string }
  | { level: "trace-detail"; serviceName: string; operationName: string; traceId: string; traceIndex: number };

interface ApmPageHeaderProps {
  ta: ApmTranslations;
  tc: CommonTranslations;
  view: ViewState;
  timeSelection: TimeRangeSelection;
  /** 信号新鲜度；null 时徽章显示「未知」，不隐藏 */
  freshness: SignalFreshnessItem | null;
  isRefreshing: boolean;
  onTimeChange: (v: TimeRangeSelection) => void;
  onRefresh: () => void;
  onGoToServices: () => void;
  onGoToService: (name: string) => void;
}

export function ApmPageHeader({
  ta,
  tc,
  view,
  timeSelection,
  freshness,
  isRefreshing,
  onTimeChange,
  onRefresh,
  onGoToServices,
  onGoToService,
}: ApmPageHeaderProps) {
  return (
    <div className="flex items-start justify-between gap-4">
      <div>
        <nav className="flex items-center gap-1 text-sm mb-1">
          <button
            onClick={onGoToServices}
            className={`transition-colors ${
              view.level === "services"
                ? "text-default font-semibold"
                : "text-primary hover:text-primary/80"
            }`}
          >
            {ta.pageTitle}
          </button>

          {view.level !== "services" && (
            <>
              <ChevronRight className="w-4 h-4 text-muted" />
              <button
                onClick={() => onGoToService(view.serviceName)}
                className={`transition-colors ${
                  view.level === "service-detail"
                    ? "text-default font-semibold"
                    : "text-primary hover:text-primary/80"
                }`}
              >
                {view.serviceName}
              </button>
            </>
          )}

          {view.level === "trace-detail" && (
            <>
              <ChevronRight className="w-4 h-4 text-muted" />
              <span className="text-default text-xs truncate max-w-[200px]">
                {view.operationName}
              </span>
              <ChevronRight className="w-4 h-4 text-muted" />
              <span className="text-default font-semibold font-mono text-xs">
                {view.traceId.slice(0, 12)}...
              </span>
            </>
          )}
        </nav>

        <p className="text-xs text-muted">{ta.pageDescription}</p>
      </div>

      <div className="flex items-center gap-2">
        <SignalFreshnessBadge item={freshness} />
        <TimeRangePicker
          value={timeSelection}
          onChange={onTimeChange}
          t={ta}
        />
        <button
          onClick={onRefresh}
          disabled={isRefreshing}
          className="flex items-center gap-1.5 px-3 py-1.5 text-sm rounded-lg bg-primary text-white hover:bg-primary/90 disabled:opacity-50 transition-colors"
        >
          <RefreshCw className={`w-3.5 h-3.5 ${isRefreshing ? "animate-spin" : ""}`} />
          {tc.refresh}
        </button>
      </div>
    </div>
  );
}
