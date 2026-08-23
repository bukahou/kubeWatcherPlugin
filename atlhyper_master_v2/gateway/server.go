// atlhyper_master_v2/gateway/server.go
// Gateway HTTP Server（Web API）
// Gateway 是外部访问层，禁止直接访问 DataHub
// 读取通过 Service 统一接口
//
// 路由注册见 routes.go
package gateway

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"AtlHyper/atlhyper_master_v2/ai"
	"AtlHyper/atlhyper_master_v2/database"
	"AtlHyper/atlhyper_master_v2/deployer"
	aiopsHandler "AtlHyper/atlhyper_master_v2/gateway/handler/aiops"
	"AtlHyper/atlhyper_master_v2/github"
	"AtlHyper/atlhyper_master_v2/service"
	"AtlHyper/common/logger"
)

var serverLog = logger.Module("Gateway")

// Server Gateway HTTP Server
type Server struct {
	port           int
	service        service.Service
	database       *database.DB
	aiService      ai.AIService
	analyzeTrigger aiopsHandler.AnalyzeTrigger
	ghClient       github.Client
	deployer       deployer.Deployer
	httpServer     *http.Server
}

// Config Server 配置
type Config struct {
	Port           int
	Service        service.Service
	Database       *database.DB
	AIService      ai.AIService                // 可选，nil 表示 AI 功能未启用
	AnalyzeTrigger aiopsHandler.AnalyzeTrigger // 可选，nil 表示深度分析未启用
	GitHubClient   github.Client               // 可选，nil 表示 GitHub 集成未配置
	Deployer       deployer.Deployer           // 可选，nil 表示 Deployer 未启用
}

// NewServer 创建 Server
func NewServer(cfg Config) *Server {
	return &Server{
		port:           cfg.Port,
		service:        cfg.Service,
		database:       cfg.Database,
		aiService:      cfg.AIService,
		analyzeTrigger: cfg.AnalyzeTrigger,
		ghClient:       cfg.GitHubClient,
		deployer:       cfg.Deployer,
	}
}

// Start 启动 Server
func (s *Server) Start() error {
	// 使用 Router 统一管理路由（见 routes.go）
	router := NewRouter(s.service, s.database, s.aiService, s.analyzeTrigger, s.ghClient, s.deployer)

	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.port),
		Handler:      router.Handler(),
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 180 * time.Second, // AI SSE 需要较长超时（多轮 Tool 调用）
	}

	serverLog.Info("启动服务器", "port", s.port)

	go func() {
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			serverLog.Error("服务器错误", "err", err)
		}
	}()

	return nil
}

// Stop 停止 Server
func (s *Server) Stop() error {
	if s.httpServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpServer.Shutdown(ctx)
	}
	return nil
}
