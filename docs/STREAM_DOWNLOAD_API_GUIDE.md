# BaiduPCS-Go-HTTP 流式下载代理接口实现指南

## 背景

DriveHub 前端需要通过浏览器下载百度网盘文件，但百度网盘直链有防盗链机制：
- 必须使用特定 User-Agent: `netdisk;2.2.51.6;netdisk;10.0.63;PC;android-android`
- 浏览器无法自定义请求头下载文件

**解决方案**：后端作为代理，用正确的 User-Agent 请求百度服务器，然后流式转发给浏览器。

---

## 新增接口规范

### 接口定义

```
GET /api/stream-download
```

### 请求参数

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| path | string | 是 | 网盘文件路径，如 `/视频/电影.mp4` |

### 响应

- 成功：直接返回文件流（二进制数据）
- 失败：返回 JSON 错误信息

### 响应头

```
Content-Type: application/octet-stream (或实际文件类型)
Content-Disposition: attachment; filename="文件名.ext"
Content-Length: 文件大小
```

---

## Go 代码实现

### 1. Handler 文件

创建 `handler/stream_download.go`:

```go
package handler

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

// StreamDownloadRequest 流式下载请求
type StreamDownloadRequest struct {
	Path string `form:"path" binding:"required"`
}

// StreamDownloadHandler 流式代理下载
// @Summary 流式下载文件
// @Description 代理下载网盘文件，解决浏览器防盗链问题
// @Tags 上传下载
// @Produce octet-stream
// @Param path query string true "文件路径"
// @Success 200 {file} binary "文件流"
// @Failure 400 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/stream-download [get]
func StreamDownloadHandler(c *gin.Context) {
	var req StreamDownloadRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    400,
			"message": "path 参数必填",
		})
		return
	}

	// 1. 获取下载直链
	// 注意：这里需要调用你现有的 locate 逻辑
	urls, err := getDownloadURLs(req.Path) // 需要你实现或复用现有代码
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": fmt.Sprintf("获取下载链接失败: %v", err),
		})
		return
	}

	if len(urls) == 0 {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": "未获取到下载链接",
		})
		return
	}

	// 2. 使用正确的 User-Agent 请求百度服务器
	downloadURL := urls[0]
	
	client := &http.Client{}
	proxyReq, err := http.NewRequest("GET", downloadURL, nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": fmt.Sprintf("创建请求失败: %v", err),
		})
		return
	}

	// 关键：设置百度网盘需要的 User-Agent
	proxyReq.Header.Set("User-Agent", "netdisk;2.2.51.6;netdisk;10.0.63;PC;android-android")
	
	// 支持断点续传（如果前端请求有 Range 头）
	if rangeHeader := c.GetHeader("Range"); rangeHeader != "" {
		proxyReq.Header.Set("Range", rangeHeader)
	}

	// 3. 发起请求
	resp, err := client.Do(proxyReq)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": fmt.Sprintf("请求百度服务器失败: %v", err),
		})
		return
	}
	defer resp.Body.Close()

	// 检查状态码
	if resp.StatusCode >= 400 {
		c.JSON(resp.StatusCode, gin.H{
			"code":    resp.StatusCode,
			"message": "百度服务器返回错误",
		})
		return
	}

	// 4. 设置响应头
	filename := filepath.Base(req.Path)
	// URL 编码文件名以支持中文
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
	io.CopyBuffer(c.Writer, resp.Body, buf)
}

// getDownloadURLs 获取下载直链
// TODO: 复用你现有的 locate 实现
func getDownloadURLs(path string) ([]string, error) {
	// 这里需要调用 BaiduPCS-Go 的 locate 功能
	// 参考现有的 LocateHandler 实现
	
	// 示例返回
	return nil, fmt.Errorf("需要实现此方法")
}
```

### 2. 注册路由

在路由配置文件中添加：

```go
// router.go 或 main.go
api := r.Group("/api")
{
    // ... 现有路由 ...
    
    // 新增流式下载
    api.GET("/stream-download", handler.StreamDownloadHandler)
}
```

### 3. 复用现有 locate 逻辑

查找你项目中现有的 locate 实现，通常在处理 `/api/locate` 请求的地方，
将获取 URL 的核心逻辑提取出来供 `getDownloadURLs` 调用。

---

## 前端调用方式

```typescript
// 方式1：直接打开新窗口下载
const downloadUrl = `/api/stream-download?path=${encodeURIComponent(filePath)}`;
window.open(downloadUrl, '_blank');

// 方式2：创建 a 标签下载
const link = document.createElement('a');
link.href = `/api/stream-download?path=${encodeURIComponent(filePath)}`;
link.download = filename;
document.body.appendChild(link);
link.click();
document.body.removeChild(link);
```

---

## 测试验证

### 使用 curl 测试

```bash
# 直接下载
curl -o test.mp4 "http://localhost:5299/api/stream-download?path=/test/video.mp4"

# 检查响应头
curl -I "http://localhost:5299/api/stream-download?path=/test/video.mp4"
```

### 预期响应头

```
HTTP/1.1 200 OK
Content-Type: video/mp4
Content-Disposition: attachment; filename*=UTF-8''video.mp4
Content-Length: 1234567890
```

---

## 注意事项

1. **大文件处理**：使用 `io.CopyBuffer` 流式传输，避免将整个文件加载到内存
2. **断点续传**：转发 `Range` 和 `Content-Range` 头支持断点续传
3. **中文文件名**：使用 `filename*=UTF-8''` 格式编码
4. **错误处理**：确保各种错误情况都有合适的响应

---

## 更新后通知

后端接口实现完成后，请告诉我：
1. 接口的最终路径（如果不是 `/api/stream-download`）
2. 我将更新前端代码以使用新接口
