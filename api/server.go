package api

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
)

// Server API 服务器
type Server struct {
	router   *gin.Engine
	httpSrv  *http.Server
	port     int
	username string
	password string
	auth     bool
}

// NewServer 创建新的 API 服务器
func NewServer(port int, username, password string, enableAuth bool) *Server {
	return &Server{
		port:     port,
		username: username,
		password: password,
		auth:     enableAuth,
	}
}

// Start 启动服务器
func (s *Server) Start() error {
	// 设置路由
	s.router = SetupRouter(s.username, s.password, s.auth)
	
	// 创建 HTTP 服务器
	s.httpSrv = &http.Server{
		Addr:           fmt.Sprintf(":%d", s.port),
		Handler:        s.router,
		ReadTimeout:    60 * time.Second,
		WriteTimeout:   60 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}
	
	// 在 goroutine 中启动服务器
	go func() {
		log.Printf("🚀 API 服务器启动在端口 %d", s.port)
		if s.auth {
			log.Printf("🔐 Basic Auth 已启用 (用户名: %s)", s.username)
		}
		log.Printf("📖 API 文档: http://localhost:%d/swagger/index.html", s.port)
		
		if err := s.httpSrv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("启动服务器失败: %v", err)
		}
	}()
	
	// 等待中断信号以优雅地关闭服务器
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	
	log.Println("正在关闭服务器...")
	
	// 5秒超时关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	if err := s.httpSrv.Shutdown(ctx); err != nil {
		return fmt.Errorf("服务器强制关闭: %v", err)
	}
	
	log.Println("服务器已退出")
	return nil
}

// Stop 停止服务器
func (s *Server) Stop() error {
	if s.httpSrv != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return s.httpSrv.Shutdown(ctx)
	}
	return nil
}
