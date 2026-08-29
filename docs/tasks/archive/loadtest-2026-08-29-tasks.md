# 压测周边任务归档（2026-08-29）

> geass-v3 压测（阶段 1–4）前后完成的 AtlHyper 任务。
> 压测完整发现与未决项仍在 tracker「压测暴露的观测缺陷」章节。

## ✅ G1 基线展示 + R1 参数统一（commit 见 git log 2026-08-29）

- G1：风险实体展开行新增「指标基线」卡片。核心是「基线能不能信」：
  冷启动阈值（coldStartMinCount）与标准差由后端下发，ready 状态 +
  学习进度条为第一信息。健康实体展开不再只有「暂无异常详情」
- R1：aiops 7 个读取点统一 `handler.ClusterIDFromQuery`，
  `cluster_id`（规范名）与 `cluster`（兼容）都接受，错误信息提示规范名
- 测试：helper 6 例 + baseline 响应 5 例，先红后绿
- 后续：基线因 emptyDir 每次重启清零 → 触发集群首个持久化方案
  （local-path-provisioner + PVC，见 config 仓 2026-08-29 commit），
  验证 count 30→(重启)→33 跨重启存活

## ✅ 构建提速：消灭 QEMU 模拟

- 三个 Dockerfile 改 `FROM --platform=$BUILDPLATFORM` + `GOARCH=$TARGETARCH`
  交叉编译；web 的 next build 产物架构无关只跑一次
- controller 附带换纯 Go SQLite 驱动（modernc.org/sqlite）消除 CGO，
  补 factory_test.go 契约测试 5 例（WAL/busy_timeout/CRUD/并发写）
- 实测：agent 10+ 分钟 → 约 40 秒；后续多轮部署全部秒级

## ✅ 压测缺陷 ①③⑤（详情与证据保留在 tracker 压测章节）

- ① 静默零值三层联动修复（39763c8）
- ③ collector 队列 50→1000 + 3核/3Gi（config 仓）
- ⑤ 三组件时区 Asia/Shanghai → Asia/Tokyo（023edff）

## ✅ 代码整洁三项 + 一处遗漏（commit 7699bc9 / b4de331）

- **SLO 组件目录归位**：`components/slo/`(11 文件) → `app/observe/slo/components/`，
  仅被 SLO 页面引用、内部全相对路径，改 2 行 import 即可，与 metrics/apm/logs 对齐
- **ObserveHandler 接口隔离**：此前持完整 `service.Query`(65 方法)，实际只用 6 个，
  其中 5 个恰是现成的 `service.QueryOTel` —— 项目早已按域拆好接口，只是 handler 没用上。
  顺带修同包 `NodeMetricsHandler`（同问题，用的是前者子集）
- **消除 handler 越层直连 database**：新增 `QueryDeploy`/`OpsDeploy`/`QueryUser`/`OpsUser`
  四个域子接口 + service 实现，deploy / github / user 三个 handler 改依赖包内最小接口。
  边界：密码哈希与权限校验仍留在 handler，service 只做数据存取
- 核查确认 gateway 层已无任何 handler 直连 database repository
- 验证：全量 build+test 通过；登录路径部署后实测返回业务错误而非 500
