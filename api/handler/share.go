package handler

import (
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/qjfoidnh/BaiduPCS-Go/api/model"
	"github.com/qjfoidnh/BaiduPCS-Go/baidupcs"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcscommand"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcsconfig"
)

// ShareSet 创建分享
// ShareSet 创建分享
// @Summary 创建分享
// @Description 创建文件的分享链接
// @Tags 分享管理
// @Accept json
// @Produce json
// @Param request body model.ShareSetRequest true "创建分享请求"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/share/set [post]
func ShareSet(c *gin.Context) {
	var req model.ShareSetRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	pcspaths, err := matchPaths(req.Paths...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	option := &baidupcs.ShareOption{
		Password: req.Password,
		Period:   req.Period,
	}

	pcs := pcscommand.GetBaiduPCS()
	shared, err := pcs.ShareSet(pcspaths, option)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"share_id": shared.ShareID,
		"link":     shared.Link,
		"password": shared.Pwd,
	}))
}

// ShareList 列出分享
// ShareList 列出分享
// @Summary 列出分享
// @Description 列出当前账户的所有分享
// @Tags 分享管理
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Success 200 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/share/list [get]
func ShareList(c *gin.Context) {
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pcs := pcscommand.GetBaiduPCS()
	records, err := pcs.ShareList(page)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	// 补充获取密码逻辑
	for _, record := range records {
		if record.Public == 0 && record.ExpireType != -1 {
			info, err := pcs.ShareSURLInfo(record.ShareID)
			if err == nil {
				record.Passwd = strings.TrimSpace(info.Pwd)
			}
		}
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"page":    page,
		"records": records,
	}))
}

// ShareCancel 取消分享
// ShareCancel 取消分享
// @Summary 取消分享
// @Description 取消指定的分享链接
// @Tags 分享管理
// @Accept json
// @Produce json
// @Param request body model.ShareCancelRequest true "取消分享请求"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/share/cancel [post]
func ShareCancel(c *gin.Context) {
	var req model.ShareCancelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	if len(req.ShareIDs) == 0 {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "没有指定 share_ids"))
		return
	}

	pcs := pcscommand.GetBaiduPCS()
	err := pcs.ShareCancel(req.ShareIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"cancelled": req.ShareIDs,
	}))
}

// Transfer 转存分享链接
// 复刻 RunShareTransfer 逻辑
// Transfer 转存分享链接
// @Summary 转存分享链接
// @Description 将他人的分享链接转存到自己网盘
// @Tags 转存管理
// @Accept json
// @Produce json
// @Param request body model.TransferRequest true "转存请求"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/transfer [post]
func Transfer(c *gin.Context) {
	var req model.TransferRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	// 解析链接
	parsedURL, err := url.Parse(req.ShareURL)
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "链接格式非法"))
		return
	}

	queryParams := parsedURL.Query()
	extractCode := queryParams.Get("pwd")
	if req.ExtractCode != "" {
		extractCode = req.ExtractCode
	}

	// 检查秒传
	if strings.Contains(req.ShareURL, "bdlink=") || !strings.Contains(req.ShareURL, "pan.baidu.com/") {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "秒传已不再被支持"))
		return
	}

	// 获取 surl
	featureStr := path.Base(strings.TrimSuffix(parsedURL.Path, "/"))
	if featureStr == "init" {
		featureStr = "1" + queryParams.Get("surl")
	}
	if len(featureStr) > 23 || featureStr[0:1] != "1" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "链接地址非法"))
		return
	}
	if len(extractCode) != 4 {
		// 尝试无密码访问或提示缺少密码？
	}

	pcs := pcscommand.GetBaiduPCS()

	// 1. 访问页面获取 tokens
	tokens := pcs.AccessSharePage(featureStr, true)
	if tokens["ErrMsg"] != "0" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, tokens["ErrMsg"]))
		return
	}

	// 2. 验证提取码
	verifyUrl := pcs.GenerateShareQueryURL("verify", map[string]string{
		"shareid":    tokens["shareid"],
		"time":       strconv.Itoa(int(time.Now().UnixMilli())),
		"clienttype": "1",
		"uk":         tokens["share_uk"],
	}).String()
	res := pcs.PostShareQuery(verifyUrl, req.ShareURL, map[string]string{
		"pwd":       extractCode,
		"vcode":     "null",
		"vcode_str": "null",
		"bdstoken":  tokens["bdstoken"],
	})
	if res["ErrMsg"] != "0" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, res["ErrMsg"]))
		return
	}

	pcs.UpdatePCSCookies(true)

	// 3. 再次获取 tokens
	tokens = pcs.AccessSharePage(featureStr, false)
	if tokens["ErrMsg"] != "0" {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, tokens["ErrMsg"]))
		return
	}

	// 4. 获取分享文件信息
	featureMap := map[string]string{
		"bdstoken": tokens["bdstoken"],
		"root":     "1",
		"web":      "5",
		"app_id":   baidupcs.PanAppID,
		"shorturl": featureStr[1:],
		"channel":  "chunlei",
	}
	queryShareInfoUrl := pcs.GenerateShareQueryURL("list", featureMap).String()
	transMetas := pcs.ExtractShareInfo(queryShareInfoUrl, tokens["shareid"], tokens["share_uk"], tokens["bdstoken"])

	if transMetas["ErrMsg"] != "success" {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, transMetas["ErrMsg"]))
		return
	}

	// 5. 准备转存路径
	activeUser := pcsconfig.Config.ActiveUser()
	savePath := activeUser.Workdir
	transMetas["path"] = savePath

	if transMetas["item_num"] != "1" && req.Collect {
		transMetas["filename"] += "等文件"
		transMetas["path"] = path.Join(savePath, transMetas["filename"])
		pcs.Mkdir(transMetas["path"])
	}

	transMetas["referer"] = "https://pan.baidu.com/s/" + featureStr
	pcs.UpdatePCSCookies(true)

	// 6. 执行转存
	resp := pcs.GenerateRequestQuery("POST", transMetas)
	if resp["ErrNo"] != "0" {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, resp["ErrMsg"]))
		return
	}

	if req.Collect {
		resp["filename"] = transMetas["filename"]
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"message": "转存成功",
		"path":    path.Join(transMetas["path"], resp["filename"]),
		"info":    resp,
	}))
}
