package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/qjfoidnh/BaiduPCS-Go/api/model"
	"github.com/qjfoidnh/BaiduPCS-Go/baidupcs"
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
// @Description 代理下载网盘文件，解决浏览器防盗链问题。后端使用正确的 User-Agent 请求百度服务器，然后流式转发给浏览器。
// @Tags 上传下载
// @Produce octet-stream
// @Param path query string true "网盘文件路径，如 /视频/电影.mp4"
// @Param cookie query string true "百度网盘 Cookie"
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
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "cookie 参数必填"))
		return
	}

	appID, _ := strconv.Atoi(BaiduPanAppID)
	// 1. 初始化 PCS 客户端
	pcs := baidupcs.NewPCSWithCookieStr(appID, cookie)

	// 设置为 Web 浏览器 UA
	webUA := "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/131.0.0.0 Safari/537.36"
	pcs.SetPanUserAgent(webUA)
	pcs.SetPCSUserAgent(webUA) // 尝试统一 UA

	// 2. 获取文件元数据 (为了拿到 fs_id) -> Not strictly needed for manual PCS link but useful for verification
	// We skip verification to avoid extra API call if possible.

	// 3. 直接构造 PCS 下载链接
	// method=download usually works if we provide cookie to CDN
	pcsURL := &url.URL{
		Scheme: "https",
		Host:   "pcs.baidu.com",
		Path:   "/rest/2.0/pcs/file",
	}
	q := pcsURL.Query()
	q.Set("method", "download")
	q.Set("path", path)
	q.Set("app_id", "250528") // Use Pan AppID for download
	pcsURL.RawQuery = q.Encode()

	bestDlink := pcsURL.String()

	// Switch to Netdisk UA (Attempt to bypass CDN 403)
	// Some sources say CDN requires "netdisk;" UA
	webUA = baidupcs.NetdiskUA

	// 4. 创建代理请求
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			// 重定向时只保留 UA，Cookie 通常不需要发给 CDN，或者按需发送
			req.Header.Set("User-Agent", webUA)
			req.Header.Set("Cookie", cookie) // CDN 需要 Cookie
			return nil
		},
	}

	proxyReq, reqErr := http.NewRequest("GET", bestDlink, nil)
	if reqErr != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, fmt.Sprintf("创建请求失败: %v", reqErr)))
		return
	}

	// 设置 UA
	proxyReq.Header.Set("User-Agent", webUA)
	proxyReq.Header.Set("Cookie", cookie)

	// 支持断点续传
	if rangeHeader := c.GetHeader("Range"); rangeHeader != "" {
		proxyReq.Header.Set("Range", rangeHeader)
	}

	// 5. 发起请求
	resp, respErr := client.Do(proxyReq)
	if respErr != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, fmt.Sprintf("下载请求失败: %v", respErr)))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		c.JSON(resp.StatusCode, model.ErrorResponse(resp.StatusCode, fmt.Sprintf("CDN返回错误: %d", resp.StatusCode)))
		return
	}

	// 6. 转发响应
	filename := filepath.Base(path)
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

	c.Status(resp.StatusCode)
	buf := make([]byte, 32*1024)
	_, _ = io.CopyBuffer(c.Writer, resp.Body, buf)
}
