package handler

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/gin-gonic/gin"
	"github.com/qjfoidnh/BaiduPCS-Go/api/model"
)

// XpanListFiles 获取文件列表
// @Summary 获取百度网盘文件列表
// @Description 使用 AccessToken 通过 xpan API 获取指定目录的文件列表
// @Tags 文件管理
// @Accept json
// @Produce json
// @Param access_token query string true "百度 AccessToken"
// @Param dir query string false "目录路径" default("/")
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Router /api/xpan/files [get]
func XpanListFiles(c *gin.Context) {
	accessToken := c.Query("access_token")
	if accessToken == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "缺少 access_token 参数"))
		return
	}

	dir := c.DefaultQuery("dir", "/")

	// 调用百度 xpan API
	apiURL := fmt.Sprintf("https://pan.baidu.com/rest/2.0/xpan/file?method=list&dir=%s&access_token=%s",
		url.QueryEscape(dir), accessToken)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, "创建请求失败: "+err.Error()))
		return
	}

	req.Header.Set("User-Agent", "pan.baidu.com")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, "请求失败: "+err.Error()))
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, "读取响应失败: "+err.Error()))
		return
	}

	// 解析百度返回的 JSON
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, "解析响应失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(result))
}

// XpanFileMetadata 获取文件元数据（含下载链接）
// @Summary 获取文件元数据
// @Description 使用 AccessToken 通过 xpan API 获取文件详细信息，包含 dlink 下载链接
// @Tags 文件管理
// @Accept json
// @Produce json
// @Param access_token query string true "百度 AccessToken"
// @Param fs_id query string true "文件 fs_id"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Router /api/xpan/file/meta [get]
func XpanFileMetadata(c *gin.Context) {
	accessToken := c.Query("access_token")
	if accessToken == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "缺少 access_token 参数"))
		return
	}

	fsID := c.Query("fs_id")
	if fsID == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "缺少 fs_id 参数"))
		return
	}

	// 调用百度 xpan API
	apiURL := fmt.Sprintf("https://pan.baidu.com/rest/2.0/xpan/multimedia?method=filemetas&dlink=1&fsids=%%5B%s%%5D&access_token=%s",
		fsID, accessToken)

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, "创建请求失败: "+err.Error()))
		return
	}

	req.Header.Set("User-Agent", "pan.baidu.com")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, "请求失败: "+err.Error()))
		return
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, "读取响应失败: "+err.Error()))
		return
	}

	// 解析百度返回的 JSON
	var result map[string]interface{}
	if err := json.Unmarshal(body, &result); err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, "解析响应失败: "+err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(result))
}
