/**
 * 跨信号跳转链接
 *
 * 观测平台的价值在于「从一个信号追到另一个信号」：SLO 说这个域名超支了，
 * 下一步必然是「哪些请求错了」。此前只有 Logs → APM 一条单向链路，
 * SLO 发现的 123 个 5xx 点不进去看。
 *
 * 时间上下文由全局时间轴（timeRangeStore）自动携带，这里只负责业务维度。
 */

/**
 * SLO 的 serviceKey（`{namespace}/{service}`）→ APM/Logs 的 OTel ServiceName。
 *
 * 当前部署下 K8s Service 名与应用的 OTEL_SERVICE_NAME 一致，取斜杠后半段即可。
 * 这是约定而非保证 —— 所以映射只在这一处，且目标页查无数据时要给明确提示，
 * 不能静默显示空白让人以为「这个服务没问题」。
 */
export function serviceKeyToOTelName(serviceKey: string): string {
  if (!serviceKey) return "";
  const idx = serviceKey.indexOf("/");
  return idx < 0 ? serviceKey : serviceKey.slice(idx + 1);
}

/** 构造带参数的路径；参数为空时返回裸路径，宁可跳过去看全部也不要坏链接 */
function buildLink(path: string, params: Record<string, string>): string {
  const usable = Object.entries(params).filter(([, v]) => v !== "");
  if (usable.length === 0) return path;
  const qs = new URLSearchParams(usable).toString();
  return `${path}?${qs}`;
}

/** SLO / 任意位置 → APM，按服务过滤 */
export function apmLinkForService(serviceKey: string): string {
  return buildLink("/observe/apm", { service: serviceKeyToOTelName(serviceKey) });
}

/**
 * SLO → Logs，按服务过滤。
 * 默认只看 ERROR —— 从 SLO 点过来就是为了查那些失败请求，
 * 落到全量日志里还要自己再筛一次没有意义。
 */
export function logsLinkForService(
  serviceKey: string,
  opts: { errorsOnly?: boolean } = {},
): string {
  const { errorsOnly = true } = opts;
  const service = serviceKeyToOTelName(serviceKey);
  return buildLink("/observe/logs", {
    service,
    severity: errorsOnly && service ? "ERROR" : "",
  });
}

/** APM trace → Logs，看这条链路上打了哪些日志 */
export function logsLinkForTrace(traceId: string): string {
  return buildLink("/observe/logs", { trace: traceId });
}
