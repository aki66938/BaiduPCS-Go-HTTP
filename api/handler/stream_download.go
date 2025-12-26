package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/qjfoidnh/BaiduPCS-Go/api/model"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcscommand"
)

const (
	// BaiduPanUserAgent 百度网盘客户端 User-Agent，用于绕过防盗链
	BaiduPanUserAgent = "netdisk;2.2.51.6;netdisk;10.0.63;PC;android-android"
)

// StreamDownload 流式代理下载
// @Summary 流式下载文件
// @Description 代理下载网盘文件，解决浏览器防盗链问题。后端使用正确的 User-Agent 请求百度服务器，然后流式转发给浏览器。
// @Tags 上传下载
// @Produce octet-stream
// @Param path query string true "网盘文件路径，如 /视频/电影.mp4"
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

	// 1. 获取下载直链
	pcs := pcscommand.GetBaiduPCS()

	// 获取文件信息
	fileInfo, err := pcs.FilesDirectoriesMeta(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, fmt.Sprintf("获取文件信息失败: %v", err)))
		return
	}

	// 检查是否为目录
	if fileInfo.Isdir {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "不支持下载目录，请指定文件路径"))
		return
	}

	// 获取下载链接
	info, pcsErr := pcs.LocateDownload(path)
	if pcsErr != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, fmt.Sprintf("获取下载链接失败: %v", pcsErr)))
		return
	}

	// 获取直链 URL
	downloadURL := info.SingleURL(true) // true for HTTPS
	if downloadURL == nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, "未获取到下载链接"))
		return
	}

	// 2. 使用 PCS 客户端发起请求（携带 BDUSS 等认证 Cookies）
	pcsClient := pcs.GetClient()
	proxyReq, reqErr := http.NewRequest("GET", downloadURL.String(), nil)
	if reqErr != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, fmt.Sprintf("创建请求失败: %v", reqErr)))
		return
	}

	// 关键：设置百度网盘需要的 User-Agent
	proxyReq.Header.Set("User-Agent", BaiduPanUserAgent)

	// 支持断点续传（如果前端请求有 Range 头）
	if rangeHeader := c.GetHeader("Range"); rangeHeader != "" {
		proxyReq.Header.Set("Range", rangeHeader)
	}

	// 3. 发起请求（使用 PCS 客户端，自动携带 Cookies）
	resp, respErr := pcsClient.Client.Do(proxyReq)
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
