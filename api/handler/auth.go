package handler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	baidulogin "github.com/qjfoidnh/Baidu-Login"
	"github.com/qjfoidnh/BaiduPCS-Go/api/model"
	"github.com/qjfoidnh/BaiduPCS-Go/baidupcs"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcsconfig"
)

// Login 登录账号
// Login 登录账号
// @Summary 登录账号
// @Description 使用 BDUSS 或 用户名/密码 登录百度网盘
// @Tags 账号管理
// @Accept json
// @Produce json
// @Param request body model.LoginRequest true "登录请求"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 401 {object} model.Response
// @Router /api/auth/login [post]
func Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	// 1. BDUSS 登录
	if req.BDUSS != "" {
		bduss := req.BDUSS

		// NewPCS 需要 appID (int) 和 bduss (string)
		appID := pcsconfig.Config.AppID
		pcs := baidupcs.NewPCS(appID, bduss)

		uk, err := pcs.UK()
		if err != nil {
			c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "BDUSS 无效"))
			return
		}

		// 保存用户配置 (bduss, ptoken, stoken, cookies)
		// 此时只有 BDUSS
		// SetupUserByBDUSS returns (*Baidu, error) (standard error)
		// But err above is pcserror.Error from pcs.UK()
		_, sysErr := pcsconfig.Config.SetupUserByBDUSS(bduss, "", "", "")
		if sysErr != nil {
			c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, "保存配置失败: "+sysErr.Error()))
			return
		}
		pcsconfig.Config.Save()

		c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
			"message": "登录成功",
			"uid":     uk,
		}))
		return
	}

	// 2. 用户名密码登录
	if req.Username != "" && req.Password != "" {
		bc := baidulogin.NewBaiduClinet()
		lj := bc.BaiduLogin(req.Username, req.Password, req.VCode, req.VCodeStr)

		switch lj.ErrInfo.No {
		case "0": // 成功
			// SetupUserByBDUSS (bduss, ptoken, stoken, cookies)
			_, err := pcsconfig.Config.SetupUserByBDUSS(lj.Data.BDUSS, lj.Data.PToken, lj.Data.SToken, lj.Data.CookieString)
			if err != nil {
				c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, "保存用户失败: "+err.Error()))
				return
			}
			pcsconfig.Config.Save()

			activeUser := pcsconfig.Config.ActiveUser()

			c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
				"message": "登录成功",
				"uid":     activeUser.UID,
			}))
			return

		case "400023", "400101": // 需要验证手机/邮箱
			// API 暂时不支持复杂的二次验证流程
			c.JSON(http.StatusForbidden, model.ErrorResponse(403, "需要手机或邮箱验证，请使用 BDUSS 登录或命令行登录"))
			return

		case "500001", "500002": // 需要验证码
			verifyImgURL := "https://wappass.baidu.com/cgi-bin/genimage?" + lj.Data.CodeString
			c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
				"status":   "need_captcha",
				"message":  lj.ErrInfo.Msg,
				"vcodestr": lj.Data.CodeString,
				"img_url":  verifyImgURL,
			}))
			return

		default:
			c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, fmt.Sprintf("登录失败: %s (%s)", lj.ErrInfo.Msg, lj.ErrInfo.No)))
			return
		}
	}

	c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "请提供 BDUSS 或 用户名/密码"))
}

// Logout 退出账号
// Logout 退出账号
// @Summary 退出当前账号
// @Description 退出当前登录的百度网盘账号
// @Tags 账号管理
// @Accept json
// @Produce json
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Router /api/auth/logout [post]
func Logout(c *gin.Context) {
	activeUser := pcsconfig.Config.ActiveUser()
	if activeUser.UID == 0 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "当前未登录"))
		return
	}

	// DeleteUser 返回 (deletedUser *Baidu, err error)
	_, err := pcsconfig.Config.DeleteUser(&pcsconfig.BaiduBase{
		UID: activeUser.UID,
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	pcsconfig.Config.Save()

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"message": "已退出登录",
		"uid":     activeUser.UID,
	}))
}
