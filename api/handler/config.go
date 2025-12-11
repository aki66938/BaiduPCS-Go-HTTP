package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/qjfoidnh/BaiduPCS-Go/api/model"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcsconfig"
)

// ConfigGet 获取当前配置
// ConfigGet 获取当前配置
// @Summary 获取当前配置
// @Description 获取 API 服务的所有配置项
// @Tags 配置管理
// @Accept json
// @Produce json
// @Success 200 {object} model.Response
// @Router /api/config [get]
func ConfigGet(c *gin.Context) {
	// 返回除了用户列表之外的配置
	// 这里直接返回 Config 结构体可能包含过多信息，可以筛选一下
	// 为了简单，先返回一些核心配置
	cfg := pcsconfig.Config

	response := gin.H{
		"appid":               cfg.AppID,
		"cache_size":          cfg.CacheSize,
		"max_parallel":        cfg.MaxParallel,
		"max_download_load":   cfg.MaxDownloadLoad,
		"max_upload_parallel": cfg.MaxUploadParallel,
		"user_agent":          cfg.UserAgent,
		"pcs_ua":              cfg.PCSUA,
		"pan_ua":              cfg.PanUA,
		"enable_https":        cfg.EnableHTTPS,
		// "proxy":              cfg.Proxy, // 敏感?
	}

	c.JSON(http.StatusOK, model.SuccessResponse(response))
}

// ConfigSet 设置配置
// 这里为了简化，假设接收一个 JSON body，包含要修改的字段
// 实际 pcsconfig.Config 是全局变量，可以直接修改后 Save
type ConfigSetRequest struct {
	AppID             int    `json:"appid"`
	CacheSize         int    `json:"cache_size"`
	MaxParallel       int    `json:"max_parallel"`
	MaxDownloadLoad   int    `json:"max_download_load"`
	MaxUploadParallel int    `json:"max_upload_parallel"`
	UserAgent         string `json:"user_agent"`
	PCSUA             string `json:"pcs_ua"`
	PanUA             string `json:"pan_ua"`
	EnableHTTPS       *bool  `json:"enable_https"`
}

// ConfigSet 设置配置
// @Summary 设置配置
// @Description 修改 API 服务的配置项
// @Tags 配置管理
// @Accept json
// @Produce json
// @Param request body handler.ConfigSetRequest true "配置设置请求"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/config/set [post]
func ConfigSet(c *gin.Context) {
	var req ConfigSetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	cfg := pcsconfig.Config

	if req.AppID != 0 {
		cfg.AppID = req.AppID
	}
	if req.CacheSize != 0 {
		cfg.CacheSize = req.CacheSize
	}
	if req.MaxParallel != 0 {
		cfg.MaxParallel = req.MaxParallel
	}
	if req.MaxDownloadLoad != 0 {
		cfg.MaxDownloadLoad = req.MaxDownloadLoad
	}
	if req.MaxUploadParallel != 0 {
		cfg.MaxUploadParallel = req.MaxUploadParallel
	}
	if req.UserAgent != "" {
		cfg.UserAgent = req.UserAgent
	}
	if req.PCSUA != "" {
		cfg.PCSUA = req.PCSUA
	}
	if req.PanUA != "" {
		cfg.PanUA = req.PanUA
	}
	if req.EnableHTTPS != nil {
		cfg.EnableHTTPS = *req.EnableHTTPS
	}

	err := cfg.Save()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"message": "配置已更新",
	}))
}

// Health 健康检查
// Health 健康检查
// @Summary 健康检查
// @Description 检查 API 服务是否存活
// @Tags 系统
// @Accept json
// @Produce json
// @Success 200 {object} map[string]string
// @Router /api/health [get]
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}
