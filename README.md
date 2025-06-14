# 🕸️ NeuroController · 插件化 Kubernetes 控制器  
🕸️ NeuroController · Plugin-Based Kubernetes Controller

**NeuroController** 是一个插件化设计的 Kubernetes 控制器，专注于集群异常监控与诊断。通过监听核心资源（如 Pod、Node、Service 等）的状态变更，结构化记录异常日志，支持去重、持久化，为系统构建统一的告警感知基础。  
**NeuroController** is a plugin-based Kubernetes controller focused on cluster anomaly monitoring and diagnostics. It listens to key resource changes (such as Pod, Node, Service), records structured alert logs, performs deduplication and persistence, and builds a unified alert perception foundation for the system.

---

## 🧠 当前功能特性  
## 🧠 Current Features

- **多资源监听器插件**  
  **Multi-Resource Watcher Plugins**  
  支持 Pod、Node、Service、Deployment、Endpoint、Event 六大核心资源的状态监听，基于 `controller-runtime` 实现，具备良好扩展性与模块隔离。  
  Supports status watching for six core resources: Pod, Node, Service, Deployment, Endpoint, and Event. Built on `controller-runtime`, it provides strong extensibility and modular isolation.

- **结构化告警日志系统**  
  **Structured Alert Logging System**  
  所有告警信息统一输出为 JSON 格式，包含时间戳、资源类型、异常等级、异常原因等字段，便于接入 Filebeat、Elasticsearch、Kibana 等日志分析平台。  
  All alert logs are output in JSON format with fields like timestamp, resource type, severity, and reason, making it easy to integrate with Filebeat, Elasticsearch, Kibana, etc.

- **日志清洗与去重机制**  
  **Log Cleaning and Deduplication**  
  内置清洗器可自动对重复告警信息进行去重与聚合，避免日志刷屏，提升可读性。  
  A built-in cleaner deduplicates and aggregates repeated alerts to reduce log flooding and improve readability.

- **日志持久化模块**  
  **Log Persistence Module**  
  清洗后的日志定时写入本地文件，默认路径为 `/var/log/neurocontroller/cleaned_events.log`，支持后续分析与归档。  
  Cleaned logs are periodically written to local files (default: `/var/log/neurocontroller/cleaned_events.log`) for analysis and archival.

- **插件注册机制**  
  **Plugin Registration System**  
  所有监听器采用集中注册方式，统一入口加载，降低耦合度，方便未来动态管理和扩展新插件。  
  All watchers are registered through a centralized entry point, reducing coupling and simplifying dynamic plugin management and future expansion.

---

## 📁 目录结构  
## 📁 Directory Structure

```bash
NeuroController/
├── build_and_push.sh         # 一键构建与推送 Docker 镜像的脚本  
                             # Script for building and pushing the Docker image
├── Dockerfile                # 多阶段容器构建配置  
                             # Multi-stage Docker build configuration
├── go.mod / go.sum           # Go 依赖管理文件  
                             # Go dependency management files
├── logs/
│   └── cleaned_events.log    # 清洗后的告警日志（可选日志持久化目录）  
                             # Cleaned alert log (optional persistent log output)
├── cmd/
│   └── neurocontroller/
│       └── main.go           # 控制器主入口（初始化管理器）  
                             # Controller main entry (manager initializer)
├── docs/
│   └── CHANGELOG_v0.2.md     # 项目更新日志  
                             # Project changelog
├── internal/
│   ├── bootstrap/
│   │   └── manager.go        # controller-runtime 管理器初始化  
                             # controller-runtime manager initializer
│   ├── diagnosis/            # 诊断模块  
│   │   ├── collector.go          # 告警信息收集器  
│   │   ├── cleaner.go            # 日志清洗与去重  
│   │   ├── dumper.go             # 持久化日志写入器  
│   │   ├── diagnosis_init.go     # 初始化入口  
│   │   └── rootcause/            # 主因识别模块（初步实现）  
│   │       ├── external_db.go    # 外部主因规则支持  
│   │       ├── internal_db.go    # 内置主因规则库  
│   │       ├── matcher.go        # 根因匹配逻辑  
│   │       └── types.go          # 主因定义结构  
│   ├── utils/                # 工具模块  
│   │   ├── k8s_client.go         # Kubernetes 客户端工具  
│   │   ├── k8s_checker.go        # 资源状态校验工具  
│   │   ├── logger.go             # 日志封装工具  
│   │   ├── exception_window.go   # 冷却窗口判断逻辑  
│   │   ├── deployment_util.go    # Deployment 专用工具  
│   │   ├── service_util.go       # Service 专用工具  
│   │   └── abnormal/             # 各资源异常识别器  
│   │       ├── pod_abnormal.go  
│   │       ├── node_abnormal.go  
│   │       ├── deployment_abnormal.go  
│   │       ├── endpoint_abnormal.go  
│   │       ├── event_abnormal.go  
│   │       ├── service_abnormal.go  
│   │       └── abnormal_utils.go  
│   └── watcher/             # 资源监听器插件模块  
│       ├── register.go          # 集中注册所有 Watcher  
│       ├── pod/
│       │   ├── pod_watcher.go  
│       │   ├── log_collector.go     # 采集 Pod 日志  
│       │   └── register.go  
│       ├── node/
│       │   ├── node_watcher.go  
│       │   └── register.go  
│       ├── service/
│       │   ├── service_watcher.go  
│       │   └── register.go  
│       ├── deployment/
│       │   ├── deployment_watcher.go  
│       │   └── register.go  
│       ├── endpoint/
│       │   ├── endpoint_watcher.go  
│       │   └── register.go  
│       └── event/
│           ├── event_watcher.go  
│           └── register.go  


## 📊 示例：结构化日志输出

以下是 NeuroController 在运行时记录的部分结构化告警日志（脱敏后的示例）：

```json
{
  "category": "Event",
  "eventTime": "2025-06-09T08:42:05Z",
  "kind": "Pod",
  "message": "健康检查未通过，容器状态异常",
  "name": "<pod-name>",
  "namespace": "default",
  "reason": "Unhealthy",
  "severity": "critical",
  "time": "2025-06-09T08:42:20Z"
}
{
  "category": "Condition",
  "eventTime": "2025-06-09T08:42:05Z",
  "kind": "Pod",
  "message": "Pod 未就绪，可能原因未知或未上报",
  "name": "<pod-name>",
  "namespace": "default",
  "reason": "NotReady",
  "severity": "warning",
  "time": "2025-06-09T08:42:20Z"
}
{
  "category": "Warning",
  "eventTime": "2025-06-09T08:42:05Z",
  "kind": "Deployment",
  "message": "Deployment 存在不可用副本，可能为镜像拉取失败、Pod 崩溃等",
  "name": "<deployment-name>",
  "namespace": "default",
  "reason": "UnavailableReplica",
  "severity": "info",
  "time": "2025-06-09T08:42:20Z"
}
{
  "category": "Endpoint",
  "eventTime": "2025-06-09T08:42:06Z",
  "kind": "Endpoints",
  "message": " 所有 Pod 已从 Endpoints 剔除（无可用后端）",
  "name": "<service-name>",
  "namespace": "default",
  "reason": "NoReadyAddress",
  "severity": "critical",
  "time": "2025-06-09T08:42:20Z"
}
```

这些日志记录展示了从 Pod 到 Deployment、Endpoint 的告警链路，便于后续根因分析和自动响应策略触发。



# 🕸️ NeuroController 使用说明 · Usage Guide

---

## ✅ 方式一：本地开发测试 · Local Development

### 📂 获取 kubeconfig 文件 · Obtain kubeconfig File

从 Kubernetes（如 K3s）集群中导出 kubeconfig 文件，例如命名为 `admin-k3s.yaml`。
Export your kubeconfig from the Kubernetes cluster (e.g., K3s), e.g., `admin-k3s.yaml`.

### 🛠️ 设置环境变量 · Set Environment Variable

将配置路径写入环境变量 `KUBECONFIG`，供控制器连接集群使用：
Set the file path to the `KUBECONFIG` environment variable so the controller can connect to the cluster:

```bash
export KUBECONFIG=/path/to/admin-k3s.yaml
```

### 🚀 启动控制器 · Run the Controller

直接通过 Go 命令启动 NeuroController：
Run NeuroController directly via Go:

```bash
go run ./cmd/neurocontroller/main.go
```

---

## ✅ 方式二：集群部署运行 · In-cluster Deployment

### 📦 构建并推送镜像 · Build & Push Image

你可以使用项目中的脚本 `build_and_push.sh` 构建并推送容器镜像：
Use the `build_and_push.sh` script to build and push the container image:

```bash
./build_and_push.sh
```

### 📜 配置 RBAC 权限 · Configure RBAC Permissions

部署前需配置访问权限，示例：
Before deploying, grant the required access permissions. Example:

```yaml
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: neurocontroller-role
rules:
  - apiGroups: [""]
    resources: ["pods", "nodes", "services", "events", "endpoints"]
    verbs: ["get", "list", "watch"]
  - apiGroups: ["apps"]
    resources: ["deployments", "replicasets"]
    verbs: ["get", "list", "watch"]
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: neurocontroller-binding
subjects:
  - kind: ServiceAccount
    name: default
    namespace: neuro
roleRef:
  kind: ClusterRole
  name: neurocontroller-role
  apiGroup: rbac.authorization.k8s.io
```

### 📦 编写 Deployment 清单 · (用户自行配置)

你可以根据集群情况编写对应的 Deployment 清单并部署该镜像。
Write a Deployment manifest using the pushed image and apply it to your cluster.

---

如需进一步帮助或演示配置示例，可随时联系维护者。
If you need more help or example manifests, feel free to reach out to the maintainer.
