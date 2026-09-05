import type { Metadata } from "next";
import "@/styles/globals.css";
import { Providers } from "./providers";

// 图标不在此声明 —— 走 Next.js 的文件约定（同目录 icon.svg / icon.png / apple-icon.png）。
// 手写 icons 字段会覆盖文件约定，反而丢掉 apple-icon 与 svg 的优先级。
// 图标从 config/branding/ 的矢量主源生成，⛔ 不要在本仓手改：
//   python3 generate.py --color '#14B8A6' --style metal --out <此目录> --sizes 512,180
export const metadata: Metadata = {
  title: "AtlHyper - Kubernetes Monitoring Platform",
  description: "AtlHyper Kubernetes 监控平台",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="zh" suppressHydrationWarning>
      <body className="antialiased">
        <Providers>{children}</Providers>
      </body>
    </html>
  );
}
