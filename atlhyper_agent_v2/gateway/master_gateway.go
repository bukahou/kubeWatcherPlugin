package gateway

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"AtlHyper/common"
	"AtlHyper/model_v3/cluster"
	"AtlHyper/model_v3/command"
)

// masterGateway Master 通信实现
//
// 使用 HTTP 协议与 Master 通信:
//   - 快照推送使用 Gzip 压缩 (减少带宽)
//   - 指令拉取使用长轮询 (减少请求频率)
//   - 所有请求带 X-Cluster-ID 头标识集群
type masterGateway struct {
	masterURL  string       // Master 服务地址
	clusterID  string       // 集群标识
	httpClient *http.Client // HTTP 客户端 (复用连接)
}

// NewMasterGateway 创建 Master 网关
//
// 参数:
//   - masterURL: Master 服务地址，如 "http://master:8080"
//   - clusterID: 集群标识
//   - httpTimeout: HTTP 客户端超时时间 (长轮询需要较长超时)
func NewMasterGateway(masterURL, clusterID string, httpTimeout time.Duration) MasterGateway {
	return &masterGateway{
		masterURL: masterURL,
		clusterID: clusterID,
		httpClient: &http.Client{
			Timeout: httpTimeout,
		},
	}
}

// PushSnapshot 推送快照
func (g *masterGateway) PushSnapshot(ctx context.Context, snapshot *cluster.ClusterSnapshot) error {
	// 1. JSON 序列化
	data, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}

	// 2. Gzip 压缩 (快照数据较大，压缩可显著减少带宽)
	compressed, err := common.GzipBytes(data)
	if err != nil {
		return fmt.Errorf("failed to compress snapshot: %w", err)
	}

	// 3. 构建请求
	url := fmt.Sprintf("%s/agent/snapshot", g.masterURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(compressed))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Content-Encoding", "gzip")
	req.Header.Set("X-Cluster-ID", g.clusterID)

	// 4. 发送请求
	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// commandResponse Master 返回的指令响应格式
// 直接使用 command.Command，无需中间类型转换
// JSON tag 必须与 Master agentsdk/types.go 中的 CommandResponse 一致（camelCase）
type commandResponse struct {
	HasCommand bool             `json:"hasCommand"`
	Command    *command.Command `json:"command,omitempty"`
}

// PollCommands 拉取指令 (长轮询)
func (g *masterGateway) PollCommands(ctx context.Context, topic string) ([]command.Command, error) {
	url := fmt.Sprintf("%s/agent/commands?cluster_id=%s&topic=%s", g.masterURL, g.clusterID, topic)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Cluster-ID", g.clusterID)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		// 超时或取消是正常的长轮询行为，不返回错误
		if ctx.Err() != nil {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// 204 表示没有指令
	if resp.StatusCode == http.StatusNoContent {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	// 解析 Master 返回的格式，直接使用 command.Command
	var cmdResp commandResponse
	if err := json.NewDecoder(resp.Body).Decode(&cmdResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	// 没有指令
	if !cmdResp.HasCommand || cmdResp.Command == nil {
		return nil, nil
	}

	// 直接返回，无需转换
	return []command.Command{*cmdResp.Command}, nil
}

// ReportResult 上报执行结果
func (g *masterGateway) ReportResult(ctx context.Context, result *command.Result) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	url := fmt.Sprintf("%s/agent/result", g.masterURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Cluster-ID", g.clusterID)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("unexpected status code: %d, body: %s", resp.StatusCode, string(body))
	}

	return nil
}

// Heartbeat 心跳
func (g *masterGateway) Heartbeat(ctx context.Context) error {
	url := fmt.Sprintf("%s/agent/heartbeat", g.masterURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("X-Cluster-ID", g.clusterID)

	resp, err := g.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("heartbeat failed with status: %d", resp.StatusCode)
	}

	return nil
}
