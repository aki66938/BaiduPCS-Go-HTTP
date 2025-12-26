package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/qjfoidnh/BaiduPCS-Go/api/model"
)

const (
	// BaiduPanUserAgent 百度网盘客户端 User-Agent，用于绕过防盗链
	BaiduPanUserAgent = "netdisk;2.2.51.6;netdisk;10.0.63;PC;android-android"
	// PCSDownloadAPI 百度 PCS 下载 API
	PCSDownloadAPI = "https://pcs.baidu.com/rest/2.0/pcs/file"
	// BaiduPanAppID 百度网盘 App ID
	BaiduPanAppID = "266719"
)

// StreamDownload 流式代理下载
// @Summary 流式下载文件
// @Description 代理下载网盘文件，解决浏览器防盗链问题。后端使用正确的 User-Agent 请求百度服务器，然后流式转发给浏览器。需要传入百度网盘的 Cookie（至少包含 BDUSS）。
// @Tags 上传下载
// @Produce octet-stream
// @Param path query string true "网盘文件路径，如 /视频/电影.mp4"
// @Param cookie query string true "百度网盘 Cookie（至少包含 BDUSS）"
// @Success 200 {file} binary "文件流"
// @Failure 400 {object} model.Response "参数错误"
// @Failure 500 {object} model.Response "服务器错误"
// @Router /api/stream-download [get]
func StreamDownload(c *gin.Context) {
	path := c.Query("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "path 参数必填"))
		return
	}

	cookie := c.Query("cookie")
	if cookie == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "cookie 参数必填，需包含 BDUSS"))
		return
	}

	// 1. 构建 PCS 下载 URL（与 curl 测试相同的方式）
	downloadURL := fmt.Sprintf("%s?app_id=%s&method=download&path=%s",
		PCSDownloadAPI, BaiduPanAppID, url.QueryEscape(path))

	// 2. 创建 HTTP 请求
	client := &http.Client{
		// 允许跟随重定向
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// 重定向时保留 User-Agent 和 Cookie
			req.Header.Set("User-Agent", BaiduPanUserAgent)
			req.Header.Set("Cookie", cookie)
			return nil
		},
	}

	proxyReq, reqErr := http.NewRequest("GET", downloadURL, nil)
	if reqErr != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, fmt.Sprintf("创建请求失败: %v", reqErr)))
		return
	}

	// 关键：设置百度网盘需要的 User-Agent 和 Cookie
	proxyReq.Header.Set("User-Agent", BaiduPanUserAgent)
	proxyReq.Header.Set("Cookie", cookie)

	// 支持断点续传（如果前端请求有 Range 头）
	if rangeHeader := c.GetHeader("Range"); rangeHeader != "" {
		proxyReq.Header.Set("Range", rangeHeader)
	}

	// 3. 发起请求
	resp, respErr := client.Do(proxyReq)
	if respErr != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, fmt.Sprintf("请求百度服务器失败: %v", respErr)))
		return
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode >= 400 {
		c.JSON(resp.StatusCode, model.ErrorResponse(resp.StatusCode, fmt.Sprintf("百度服务器返回错误: %d", resp.StatusCode)))
		return
	}

	// 4. 设置响应头
	filename := filepath.Base(path)
	// URL 编码文件名以支持中文 (RFC 5987)
	encodedFilename := url.PathEscape(filename)

	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename*=UTF-8''%s", encodedFilename))
	c.Header("Content-Type", resp.Header.Get("Content-Type"))
	if contentLength := resp.Header.Get("Content-Length"); contentLength != "" {
		c.Header("Content-Length", contentLength)
	}
	if acceptRanges := resp.Header.Get("Accept-Ranges"); acceptRanges != "" {
		c.Header("Accept-Ranges", acceptRanges)
	}
	if contentRange := resp.Header.Get("Content-Range"); contentRange != "" {
		c.Header("Content-Range", contentRange)
	}

	// 5. 流式转发
	c.Status(resp.StatusCode)

	// 使用缓冲区流式传输，避免内存问题
	buf := make([]byte, 32*1024) // 32KB 缓冲区
	_, _ = io.CopyBuffer(c.Writer, resp.Body, buf)
}
