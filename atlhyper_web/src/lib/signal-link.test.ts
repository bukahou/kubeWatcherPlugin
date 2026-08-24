import { describe, it, expect } from "vitest";
import { serviceKeyToOTelName, apmLinkForService, logsLinkForService, logsLinkForTrace } from "./signal-link";

describe("serviceKeyToOTelName", () => {
  // SLO 用 {namespace}/{service}，APM/Logs 用 OTel 资源属性里的 ServiceName。
  // 当前部署下 K8s Service 名与 OTEL_SERVICE_NAME 一致，取斜杠后半段即可 ——
  // 但这是约定不是保证，所以映射集中一处，跳转后查无数据要有提示。
  it("取命名空间之后的服务名", () => {
    expect(serviceKeyToOTelName("geass-v3/geass-gateway")).toBe("geass-gateway");
    expect(serviceKeyToOTelName("atlhyper/atlhyper-web")).toBe("atlhyper-web");
  });

  it("没有斜杠时原样返回", () => {
    expect(serviceKeyToOTelName("akasha")).toBe("akasha");
  });

  it("服务名里含斜杠时只切第一个", () => {
    expect(serviceKeyToOTelName("ns/a/b")).toBe("a/b");
  });

  it("空值返回空字符串而不是抛异常", () => {
    expect(serviceKeyToOTelName("")).toBe("");
  });
});

describe("跨信号跳转链接", () => {
  // 时间上下文由全局时间轴携带，链接只带业务维度
  it("SLO → APM 带服务名", () => {
    expect(apmLinkForService("geass-v3/geass-gateway")).toBe(
      "/observe/apm?service=geass-gateway",
    );
  });

  it("SLO → Logs 默认只看错误 —— 从 SLO 点过来就是为了查那些失败请求", () => {
    expect(logsLinkForService("geass-v3/geass-gateway")).toBe(
      "/observe/logs?service=geass-gateway&severity=ERROR",
    );
  });

  it("SLO → Logs 可以要全部级别", () => {
    expect(logsLinkForService("geass-v3/geass-gateway", { errorsOnly: false })).toBe(
      "/observe/logs?service=geass-gateway",
    );
  });

  it("APM → Logs 带 traceId", () => {
    expect(logsLinkForTrace("abc123")).toBe("/observe/logs?trace=abc123");
  });

  it("服务名含特殊字符时正确编码", () => {
    expect(apmLinkForService("ns/svc with space")).toBe("/observe/apm?service=svc+with+space");
  });

  it("空服务名返回目标页而不带参数 —— 宁可跳过去看全部，也不要一个坏链接", () => {
    expect(apmLinkForService("")).toBe("/observe/apm");
    expect(logsLinkForTrace("")).toBe("/observe/logs");
  });
});
