/**
 * 数据源代理层 — 统一导出
 *
 * 页面只需 import { xxx } from "@/datasource/metrics" 等
 * 不再直接 import @/mock/* 或 @/api/*
 */

export * as metrics from "./metrics";
export * as logs from "./logs";
export * as apm from "./apm";
export * as cluster from "./cluster";
export * as slo from "./slo";
export * as overview from "./overview";
export * as github from "./github";
export * as deploy from "./deploy";
