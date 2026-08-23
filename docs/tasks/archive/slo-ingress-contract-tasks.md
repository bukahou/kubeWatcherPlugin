# SLO Ingress 契约改造 — 已完成

> 完成时间: 2026-08-23
> 设计文档: [slo-ingress-contract-design.md](../../design/archive/slo-ingress-contract-design.md)

## 背景

SLO 两路数据源全死（实测 0 行）：Ingress 查 traefik_*（Traefik 已拆）、
Mesh 查 mesh_request_total（Istio 已拆）。根因是 Ingress 一路从未做过抽象。

## 完成的 Phase

- Phase 1: Collector 加 cilium-envoy scrape + ingress_* 归一化 transform（config 仓）
- Phase 2: 验证契约数据落库（5 服务 / 19 桶 / label 无实现名泄漏）
- Phase 3: Agent slo.go 改用契约 + 契约自检扩展 — 16cbd2c, d528129
- Phase 4: HTTPRoute 域名映射 + avgMs（Sum/Count）— 6e110f0
- Phase 5: 移除 Mesh 全链路（后端 af2f65d / 前端见 git log）
- Phase 6: 端到端验收通过

## 实施中发现的五个问题（都只有真跑才会暴露）

1. OTel Prometheus receiver 在 scrape 阶段就把 histogram 三联合并成单指标，
   `_bucket` 后缀不存在 → 改名规则永远匹配不上
2. OTTL 无法改写 ExplicitBounds → 契约单位从秒改为毫秒
3. 删 goroutine 时误改了另一组的 wg.Add → APM 采集被整组跳过（测试抓住）
4. 契约自检对千万行表做 SELECT DISTINCT → i/o timeout（线上才暴露）
5. IngressPath.Backend 是指针，缺 nil 检查会 panic（编译期不报）

## 遗留

- 65 个 kube_* 指标白采白存（1.08 亿行），见 ../future/metrics-collection-cleanup-tasks.md
- mock/slo/data.ts 里仍有 Linkerd/Traefik 的注释与示例数据（不影响功能）
