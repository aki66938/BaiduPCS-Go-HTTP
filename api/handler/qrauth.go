package handler

import (
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/qjfoidnh/BaiduPCS-Go/api/model"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcscommand"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcsconfig"
)

// QRCodeSession 二维码会话缓存
type QRCodeSession struct {
	Client   *pcscommand.QRLoginClient
	QRInfo   *pcscommand.QRCodeInfo
	ExpireAt time.Time
}

var (
	// qrSessions 保存活跃的二维码会话
	qrSessions = make(map[string]*QRCodeSession)
	qrMutex    sync.RWMutex
)

// QRCodeGetRequest 获取二维码请求
type QRCodeGetRequest struct {
	IncludeASCII bool `json:"include_ascii"` // 是否包含 ASCII 格式二维码
}

// QRCodeGet 获取登录二维码
// @Summary 获取登录二维码
// @Description 获取百度网盘扫码登录的二维码
// @Tags 账号管理
// @Accept json
// @Produce json
// @Param request body QRCodeGetRequest false "请求参数"
// @Success 200 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/auth/qrcode [post]
func QRCodeGet(c *gin.Context) {
	var req QRCodeGetRequest
	// 允许空请求体
	c.ShouldBindJSON(&req)

	client := pcscommand.NewQRLoginClient()
	qrInfo, err := client.GetQRCode()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, "获取二维码失败: "+err.Error()))
		return
	}

	// 缓存会话
	qrMutex.Lock()
	qrSessions[qrInfo.Sign] = &QRCodeSession{
		Client:   client,
		QRInfo:   qrInfo,
		ExpireAt: qrInfo.ExpireAt,
	}
	qrMutex.Unlock()

	// 清理过期会话
	go cleanExpiredSessions()

	// 构建响应
	respData := gin.H{
		"sign":      qrInfo.Sign,
		"img_url":   qrInfo.ImgURL,
		"expire_at": qrInfo.ExpireAt.Format(time.RFC3339),
	}

	// 如果请求包含 ASCII，添加 ASCII 二维码
	if req.IncludeASCII {
		respData["ascii_qr"] = qrInfo.ASCIIQA
	}

	c.JSON(http.StatusOK, model.SuccessResponse(respData))
}

// QRCodeStatus 查询扫码状态
// @Summary 查询扫码状态
// @Description 查询二维码的扫码状态
// @Tags 账号管理
// @Produce json
// @Param sign query string true "二维码标识符"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 404 {object} model.Response
// @Router /api/auth/qrcode/status [get]
func QRCodeStatus(c *gin.Context) {
	sign := c.Query("sign")
	if sign == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "缺少 sign 参数"))
		return
	}

	// 获取会话
	qrMutex.RLock()
	session, exists := qrSessions[sign]
	qrMutex.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, model.ErrorResponse(404, "二维码会话不存在或已过期"))
		return
	}

	// 检查是否过期
	if time.Now().After(session.ExpireAt) {
		qrMutex.Lock()
		delete(qrSessions, sign)
		qrMutex.Unlock()
		c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
			"status":  pcscommand.QRStatusExpired.String(),
			"message": "二维码已过期",
		}))
		return
	}

	// 查询状态
	result, err := session.Client.CheckQRStatus(sign)
	if err != nil {
		c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
			"status":  pcscommand.QRStatusWaiting.String(),
			"message": "等待扫码",
		}))
		return
	}

	respData := gin.H{
		"status":  result.Status.String(),
		"message": result.Message,
	}

	// 如果已确认，返回临时 BDUSS
	if result.Status == pcscommand.QRStatusConfirmed && result.TempBDUSS != "" {
		respData["temp_bduss"] = result.TempBDUSS
	}

	c.JSON(http.StatusOK, model.SuccessResponse(respData))
}

// QRCodeLoginRequest 扫码登录请求
type QRCodeLoginRequest struct {
	Sign      string `json:"sign" binding:"required"`       // 二维码标识符
	TempBDUSS string `json:"temp_bduss" binding:"required"` // 临时 BDUSS
}

// QRCodeLogin 使用临时凭证完成登录
// @Summary 完成扫码登录
// @Description 使用临时 BDUSS 完成登录
// @Tags 账号管理
// @Accept json
// @Produce json
// @Param request body QRCodeLoginRequest true "登录请求"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/auth/qrcode/login [post]
func QRCodeLogin(c *gin.Context) {
	var req QRCodeLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	// 获取会话
	qrMutex.RLock()
	session, exists := qrSessions[req.Sign]
	qrMutex.RUnlock()

	var client *pcscommand.QRLoginClient
	if exists {
		client = session.Client
	} else {
		// 如果会话不存在，创建新客户端
		client = pcscommand.NewQRLoginClient()
	}

	// 交换正式凭证
	bduss, stoken, cookies, err := client.ExchangeBDUSS(req.TempBDUSS)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, "交换凭证失败: "+err.Error()))
		return
	}

	// 保存用户配置
	baidu, err := pcsconfig.Config.SetupUserByBDUSS(bduss, "", stoken, cookies)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, "保存用户配置失败: "+err.Error()))
		return
	}
	pcsconfig.Config.Save()

	// 清理会话
	qrMutex.Lock()
	delete(qrSessions, req.Sign)
	qrMutex.Unlock()

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"message": "登录成功",
		"uid":     baidu.UID,
		"name":    baidu.Name,
		"bduss":   bduss,
		"stoken":  stoken,
		"cookies": cookies,
	}))
}

// cleanExpiredSessions 清理过期的二维码会话
func cleanExpiredSessions() {
	qrMutex.Lock()
	defer qrMutex.Unlock()

	now := time.Now()
	for sign, session := range qrSessions {
		if now.After(session.ExpireAt) {
			delete(qrSessions, sign)
		}
	}
}
