import { defineConfig } from "vitest/config";
import path from "node:path";

// 只跑纯函数测试（时间换算、URL 编解码、跨信号链接构造）。
// 组件测试暂不引入 —— 按项目规范，组件只在有复杂交互逻辑时才测。
export default defineConfig({
  test: {
    environment: "node",
    include: ["src/**/*.test.ts"],
  },
  resolve: {
    alias: { "@": path.resolve(__dirname, "src") },
  },
});
