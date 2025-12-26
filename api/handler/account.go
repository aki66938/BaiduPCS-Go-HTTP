package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/qjfoidnh/BaiduPCS-Go/api/model"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcscommand"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcsconfig"
)

// Who 获取当前账号信息
// Who 获取当前账号信息
// @Summary 获取当前账号信息
// @Description 获取当前登录用户的详细信息
// @Tags 账号管理
// @Accept json
// @Produce json
// @Success 200 {object} model.Response
// @Failure 401 {object} model.Response
// @Router /api/account/who [get]
func Who(c *gin.Context) {
	activeUser := pcsconfig.Config.ActiveUser()
	if activeUser == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "未登录"))
		return
	}

	// 构造或获取 cookies
	cookies := activeUser.COOKIES
	if cookies == "" {
		// 如果是通过扫码登录等方式，可能没有完整的 cookies 字符串
		// 手动拼接至少包含 BDUSS 和 STOKEN
		cookies = "BDUSS=" + activeUser.BDUSS
		if activeUser.STOKEN != "" {
			cookies += "; STOKEN=" + activeUser.STOKEN
		}
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"uid":     activeUser.UID,
		"name":    activeUser.Name,
		"sex":     activeUser.Sex,
		"age":     activeUser.Age,
		"bduss":   activeUser.BDUSS,
		"workdir": activeUser.Workdir,
		"cookies": cookies, // 新增：返回完整的 cookies 字符串供前端使用
	}))
}

// Quota 获取网盘配额
// Quota 获取网盘配额
// @Summary 获取网盘配额
// @Description 获取当前账号的空间配额和使用情况
// @Tags 账号管理
// @Accept json
// @Produce json
// @Success 200 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/account/quota [get]
func Quota(c *gin.Context) {
	pcs := pcscommand.GetBaiduPCS()
	// QuotaInfo 返回 (quota, used int64, pcsError pcserror.Error)
	quota, used, err := pcs.QuotaInfo()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	ratio := 0.0
	if quota > 0 {
		ratio = float64(used) / float64(quota) * 100
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"total": quota,
		"used":  used,
		"ratio": ratio,
	}))
}

// UserList 列出所有已登录账号
// UserList 列出所有已登录账号
// @Summary 列出所有已登录账号
// @Description 获取本服务中保存的所有登录用户列表
// @Tags 账号管理
// @Accept json
// @Produce json
// @Success 200 {object} model.Response
// @Router /api/account/list [get]
func UserList(c *gin.Context) {
	users := pcsconfig.Config.BaiduUserList

	// 为了安全，不返回敏感信息如 BDUSS
	var simpleUsers []map[string]interface{}
	for _, u := range users {
		simpleUsers = append(simpleUsers, map[string]interface{}{
			"uid":  u.UID,
			"name": u.Name,
			"age":  u.Age,
			"sex":  u.Sex,
		})
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"users": simpleUsers,
	}))
}

// Switch 切换账号
// Switch 切换账号
// @Summary 切换账号
// @Description 切换当前活跃的百度网盘账号
// @Tags 账号管理
// @Accept json
// @Produce json
// @Param request body model.UserSwitchRequest true "切换用户请求"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/account/switch [post]
func Switch(c *gin.Context) {
	var req model.UserSwitchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	// 切换用户
	// SwitchUser(user *BaiduBase) (switchedUser *Baidu, err error)
	targetUser := &pcsconfig.BaiduBase{
		UID: req.UID,
	}
	_, err := pcsconfig.Config.SwitchUser(targetUser)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, "切换失败: "+err.Error()))
		return
	}

	err = pcsconfig.Config.Save()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, "保存配置失败"))
		return
	}

	activeUser := pcsconfig.Config.ActiveUser()
	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"message": "切换成功",
		"uid":     activeUser.UID,
		"name":    activeUser.Name,
	}))
}
