/**
 * 环境变量配置
 * 集中管理所有环境变量，提供类型安全
 */

const isDev = process.env.NODE_ENV === "development";

export const env = {
  // API 配置
  // 统一使用相对路径，由 src/middleware.ts 代理到 controller。
  // ⚠️ 后端地址读的是运行时环境变量 API_URL，而不是 next.config.ts ——
  //    next.config.ts 里没有 rewrites 段（原注释指错了文件）。
  //    运行时读取意味着换后端地址不必重新构建镜像。
  apiUrl: "",

  // 数据刷新间隔（毫秒），默认 30 秒
  refreshInterval: Number(process.env.NEXT_PUBLIC_REFRESH_INTERVAL) || 30000,

  // 运行环境
  isDev,
  isProd: process.env.NODE_ENV === "production",
} as const;

// 导出类型
export type Env = typeof env;
