// atlhyper_master_v2/ai/prompts/tools.go
// Tool JSON 定义 + 加载/缓存函数
package prompts

import (
	"encoding/json"

	"AtlHyper/atlhyper_master_v2/ai/llm"
)

// toolsJSON Tool 定义 JSON — 单一通用 Tool
const toolsJSON = `[
  {
    "name": "query_cluster",
    "description": "查询 Kubernetes 集群数据（只读）。通过 API Server 直连获取任意资源信息。可在一次回复中多次调用以并行获取数据。",
    "parameters": {
      "type": "object",
      "properties": {
        "action": {
          "type": "string",
          "description": "操作类型: get, list, describe, get_logs, get_events, get_configmap",
          "enum": ["get", "list", "describe", "get_logs", "get_events", "get_configmap"]
        },
        "kind": {
          "type": "string",
          "description": "Kubernetes 资源类型: Pod, Deployment, Service, Node, HPA, StatefulSet, DaemonSet, Job, CronJob, PVC, PV, Ingress, ConfigMap, NetworkPolicy, ReplicaSet, Endpoints, Namespace, ServiceAccount, Event 等"
        },
        "namespace": {
          "type": "string",
          "description": "命名空间。list 操作时可不填以查询所有命名空间；get/describe 需要填写。集群级资源（Node、PV、Namespace）始终不需要填写"
        },
        "name": {
          "type": "string",
          "description": "资源名称（list 操作时可不填）"
        },
        "label_selector": {
          "type": "string",
          "description": "标签选择器，如 app=nginx（list 时可用，用于过滤）"
        },
        "container": {
          "type": "string",
          "description": "容器名称（get_logs 时多容器 Pod 需要指定）"
        },
        "tail_lines": {
          "type": "integer",
          "description": "返回日志的尾行数（get_logs 时使用，默认 100）"
        },
        "involved_kind": {
          "type": "string",
          "description": "关联资源类型（get_events 时过滤用，如 Pod、Node、Deployment）"
        },
        "involved_name": {
          "type": "string",
          "description": "关联资源名称（get_events 时过滤用）"
        }
      },
      "required": ["action", "kind"]
    }
  },
  {
    "name": "analyze_incident",
    "description": "分析指定事件的根因、影响面和处置建议。输入事件 ID，返回 AI 分析结果。",
    "parameters": {
      "type": "object",
      "properties": {
        "incident_id": {
          "type": "string",
          "description": "事件 ID，格式如 inc-1737364200"
        }
      },
      "required": ["incident_id"]
    }
  },
  {
    "name": "get_cluster_risk",
    "description": "获取集群当前的风险评分和高风险实体。返回 ClusterRisk 分数 (0-100) 和 Top N 风险实体列表。",
    "parameters": {
      "type": "object",
      "properties": {
        "top_n": {
          "type": "integer",
          "description": "返回前 N 个高风险实体，默认 10"
        }
      }
    }
  },
  {
    "name": "get_recent_incidents",
    "description": "获取最近的事件列表。可按状态过滤，返回事件摘要。",
    "parameters": {
      "type": "object",
      "properties": {
        "state": {
          "type": "string",
          "enum": ["warning", "incident", "recovery", "stable"],
          "description": "按状态过滤，不填则返回所有状态"
        },
        "limit": {
          "type": "integer",
          "description": "返回数量，默认 10"
        }
      }
    }
  },
  {
    "name": "query_traces",
    "description": "查询 APM 分布式追踪数据。可按服务名、操作名、错误状态过滤。返回 Trace 摘要列表含耗时、Span 数、错误信息。最多返回 10 条。",
    "parameters": {
      "type": "object",
      "properties": {
        "service": {
          "type": "string",
          "description": "服务名过滤（如 geass-gateway）"
        },
        "operation": {
          "type": "string",
          "description": "操作名过滤（如 GET /api/v1/users）"
        },
        "min_duration_ms": {
          "type": "number",
          "description": "最小耗时（毫秒），用于查找慢请求"
        },
        "status_code": {
          "type": "string",
          "description": "HTTP 状态码过滤（如 500、404）"
        },
        "since": {
          "type": "string",
          "description": "时间范围，如 5m、1h、24h。默认 1h",
          "default": "1h"
        }
      },
      "required": []
    }
  },
  {
    "name": "query_logs",
    "description": "查询 OpenTelemetry 结构化日志。支持全文搜索、按服务/级别/TraceId 过滤。最多返回 20 条，日志 Body 截断为 200 字符。",
    "parameters": {
      "type": "object",
      "properties": {
        "query": {
          "type": "string",
          "description": "全文搜索关键词（模糊匹配日志 Body）"
        },
        "service": {
          "type": "string",
          "description": "服务名过滤"
        },
        "level": {
          "type": "string",
          "enum": ["DEBUG", "INFO", "WARN", "ERROR"],
          "description": "日志级别过滤"
        },
        "trace_id": {
          "type": "string",
          "description": "按 TraceId 过滤（跨信号关联）"
        },
        "since": {
          "type": "string",
          "description": "时间范围，如 15m、1h、24h。默认 1h",
          "default": "1h"
        }
      },
      "required": []
    }
  },
  {
    "name": "query_slo",
    "description": "查询 SLO 指标数据。返回服务/域名的可用性、延迟分位数（P50/P90/P99）、错误率、RPS。支持 1 天/7 天/30 天窗口。",
    "parameters": {
      "type": "object",
      "properties": {
        "service": {
          "type": "string",
          "description": "服务名（Linkerd mesh 服务）"
        },
        "domain": {
          "type": "string",
          "description": "域名（Traefik ingress 域名）"
        },
        "window": {
          "type": "string",
          "enum": ["1h", "6h", "24h", "3d", "7d"],
          "description": "时间窗口，默认 7d",
          "default": "7d"
        }
      },
      "required": []
    }
  },
  {
    "name": "get_entity_detail",
    "description": "获取特定实体（Pod/Service/Node/Ingress）的风险详情。包含：风险分数、异常指标列表、因果树（上下游异常实体关系）、传播路径。用于深度分析某个实体为什么异常。",
    "parameters": {
      "type": "object",
      "properties": {
        "entity_type": {
          "type": "string",
          "enum": ["pod", "service", "node", "ingress"],
          "description": "实体类型"
        },
        "entity_name": {
          "type": "string",
          "description": "实体名称"
        },
        "namespace": {
          "type": "string",
          "description": "命名空间（Pod/Service 必需，Node/Ingress 可选）"
        }
      },
      "required": ["entity_type", "entity_name"]
    }
  },
  {
    "name": "get_deploy_history",
    "description": "查询指定路径的部署历史，包括 commit SHA、部署状态、触发方式等信息。用于排查部署相关问题时了解最近的变更。",
    "parameters": {
      "type": "object",
      "properties": {
        "path": {
          "type": "string",
          "description": "kustomize 部署路径，如 'Geass/backend' 或 'atlhyper/master'"
        },
        "limit": {
          "type": "integer",
          "description": "返回最近 N 条记录，默认 5",
          "default": 5
        }
      },
      "required": ["path"]
    }
  },
  {
    "name": "rollback_deployment",
    "description": "将指定路径回滚到历史 commit 版本。回滚会重新应用目标版本的 kustomize 配置。此操作需谨慎使用。",
    "parameters": {
      "type": "object",
      "properties": {
        "path": {
          "type": "string",
          "description": "kustomize 部署路径"
        },
        "target_commit_sha": {
          "type": "string",
          "description": "目标 commit SHA（从 deploy_history 获取）"
        }
      },
      "required": ["path", "target_commit_sha"]
    }
  },
  {
    "name": "github_read_file",
    "description": "读取关联 GitHub 仓库中的文件内容，用于代码分析和问题排查。",
    "parameters": {
      "type": "object",
      "properties": {
        "repo": {
          "type": "string",
          "description": "仓库全名，格式 'owner/repo'，如 'wuxiafeng/Geass'"
        },
        "path": {
          "type": "string",
          "description": "文件路径，相对于仓库根目录"
        },
        "branch": {
          "type": "string",
          "description": "分支名，默认 'main'",
          "default": "main"
        }
      },
      "required": ["repo", "path"]
    }
  },
  {
    "name": "github_search_code",
    "description": "在关联 GitHub 仓库中搜索代码，查找特定关键字、类名、函数名等。",
    "parameters": {
      "type": "object",
      "properties": {
        "repo": {
          "type": "string",
          "description": "仓库全名，格式 'owner/repo'"
        },
        "query": {
          "type": "string",
          "description": "搜索关键字（支持 GitHub 代码搜索语法）"
        }
      },
      "required": ["repo", "query"]
    }
  },
  {
    "name": "github_recent_commits",
    "description": "查看仓库最近的 commits，可限定文件路径范围。用于了解最近的代码变更。",
    "parameters": {
      "type": "object",
      "properties": {
        "repo": {
          "type": "string",
          "description": "仓库全名，格式 'owner/repo'"
        },
        "path": {
          "type": "string",
          "description": "限定路径（可选），如 'src/auth/'",
          "default": ""
        },
        "limit": {
          "type": "integer",
          "description": "返回最近 N 条，默认 10",
          "default": 10
        }
      },
      "required": ["repo"]
    }
  }
]`

// LoadToolDefinitions 加载 Tool 定义
// 从 toolsJSON 常量解析为 llm.ToolDefinition 列表
func LoadToolDefinitions() ([]llm.ToolDefinition, error) {
	var rawTools []struct {
		Name        string          `json:"name"`
		Description string          `json:"description"`
		Parameters  json.RawMessage `json:"parameters"`
	}
	if err := json.Unmarshal([]byte(toolsJSON), &rawTools); err != nil {
		return nil, err
	}

	tools := make([]llm.ToolDefinition, len(rawTools))
	for i, t := range rawTools {
		tools[i] = llm.ToolDefinition{
			Name:        t.Name,
			Description: t.Description,
			Parameters:  t.Parameters,
		}
	}
	return tools, nil
}

// toolsCache 缓存加载的 Tool 定义
var toolsCache []llm.ToolDefinition

// GetToolDefinitions 获取 Tool 定义（带缓存）
func GetToolDefinitions() []llm.ToolDefinition {
	if toolsCache == nil {
		var err error
		toolsCache, err = LoadToolDefinitions()
		if err != nil {
			return nil
		}
	}
	return toolsCache
}

// ResetToolCache 重置 Tool 定义缓存（新增 Tool 后调用）
func ResetToolCache() {
	toolsCache = nil
}
