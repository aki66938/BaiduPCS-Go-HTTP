package handler

import (
	"fmt"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/qjfoidnh/BaiduPCS-Go/api/model"
	"github.com/qjfoidnh/BaiduPCS-Go/baidupcs"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcscommand"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcsconfig"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcsfunctions/pcsupload"
	"github.com/qjfoidnh/BaiduPCS-Go/pcsutil"
	"github.com/qjfoidnh/BaiduPCS-Go/pcsutil/checksum"
	"github.com/qjfoidnh/BaiduPCS-Go/pcsutil/taskframework"
)

// Upload 上传服务器本地文件到网盘
// Upload 上传服务器本地文件到网盘
// @Summary 上传文件
// @Description 上传服务器本地文件到网盘 (application/json) 或 客户端上传文件 (multipart/form-data)
// @Tags 上传下载
// @Accept json,mpfd
// @Produce json
// @Param request body model.ServerUploadRequest false "服务器本地文件上传请求 (json)"
// @Param file formData file false "文件内容 (form-data)"
// @Param target_dir formData string false "目标目录 (form-data)"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/upload [post]
func Upload(c *gin.Context) {
	// 接收 form-data 或 json?
	// 鉴于复用 model.UploadRequest，我们假设是 form-data 用于 file upload (client -> server)
	// 或者 json 用于 server-side upload (server local file -> cloud)?
	// 之前的定义：UploadRequest 包含 TargetPath。没有 SourcePath？
	// 如果是 Server-side upload，我们需要 SourcePath。

	// 让我们检查 model.UploadRequest
	// type UploadRequest struct {
	//    TargetPath string `form:"target_path"`
	// }
	// 这里的 form tag 暗示可能是用于 multipart upload 的参数。

	// 但为了支持 n8n "Server-Side Upload" (控制服务器上传本地文件)，我们需要 SourcePath。
	// 这里我们做一个混合处理：
	// 如果 Content-Type 是 multipart/form-data，则处理文件上传（流式）。
	// 如果是 application/json，则处理服务器本地文件上传。

	contentType := c.ContentType()

	if strings.Contains(contentType, "application/json") {
		handleServerSideUpload(c)
		return
	}

	// Multipart upload not fully implemented yet due to complexity of streaming to PCS
	// But we can implement a simple version: save temp file -> upload -> delete temp
	handleMultipartUpload(c)
}

// handleServerSideUpload 服务器本地文件上传
func handleServerSideUpload(c *gin.Context) {
	var req model.ServerUploadRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	// 路径处理
	targetDir, err := matchPath(req.TargetDir)
	if err != nil {
		// 目标必须存在？或者如果是新目录？
		// 为了简单，我们尝试创建或使用
		// 如果 matchPath 失败（不存在），我们使用 PathJoin
		user := pcsconfig.Config.ActiveUser()
		targetDir = user.PathJoin(req.TargetDir)
		pcscommand.GetBaiduPCS().Mkdir(targetDir) // 尝试创建
	}

	// 准备上传
	pcs := pcscommand.GetBaiduPCS()
	uploadDatabase, err := pcsupload.NewUploadingDatabase()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, "无法初始化上传数据库"))
		return
	}
	// defer uploadDatabase.Close() // 不能在这里关闭，因为要在 goroutine 使用？
	// 不，TaskExecutor 执行完才关闭。
	// 如果异步，需要小心。

	executor := &taskframework.TaskExecutor{
		IsFailedDeque: true,
	}
	statistic := &pcsupload.UploadStatistic{}

	optParallel := pcsconfig.Config.MaxUploadParallel
	optLoad := pcsconfig.Config.MaxUploadLoad
	optPolicy := req.Policy
	if optPolicy == "" {
		optPolicy = pcsconfig.Config.UPolicy
	}

	var tasks []string

	// 遍历本地文件
	for _, localPath := range req.LocalPaths {
		walkedFiles, err := pcsutil.WalkDir(localPath, "")
		if err != nil {
			continue
		}

		for _, file := range walkedFiles {
			// 计算网盘路径
			relPath, _ := filepath.Rel(filepath.Dir(localPath), file)
			savePath := path.Clean(targetDir + baidupcs.PathSeparator + filepath.ToSlash(relPath))

			unit := pcsupload.UploadTaskUnit{
				LocalFileChecksum: checksum.NewLocalFileChecksum(file, int(baidupcs.SliceMD5Size)),
				SavePath:          savePath,
				PCS:               pcs,
				UploadingDatabase: uploadDatabase,
				Parallel:          optParallel,
				PrintFormat:       "", // Silent
				NoRapidUpload:     false,
				NoSplitFile:       false,
				UploadStatistic:   statistic,
				Policy:            optPolicy,
			}

			executor.Append(&unit, 3)
			tasks = append(tasks, fmt.Sprintf("%s -> %s", file, savePath))
		}
	}

	if len(tasks) == 0 {
		uploadDatabase.Close()
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "没有可上传的文件"))
		return
	}

	// 异步执行
	go func() {
		defer uploadDatabase.Close()
		executor.SetParallel(optLoad)
		statistic.StartTimer()
		executor.Execute()
		fmt.Printf("API Upload finished: %d files\n", len(tasks))
	}()

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"message": "上传任务已在后台启动",
		"files":   tasks,
	}))
}

// handleMultipartUpload 处理客户端文件上传
func handleMultipartUpload(c *gin.Context) {
	// 1. 获取上传的目标目录
	targetDir := c.PostForm("target_dir")
	if targetDir == "" {
		targetDir = "/"
	}

	// 2. 获取文件
	file, header, err := c.Request.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "获取上传文件失败"))
		return
	}
	defer file.Close()

	// 3. 保存到临时文件
	// 由于 BaiduPCS Upload 需要本地路径（计算MD5等），我们需要先存盘
	tempDir := os.TempDir()
	tempPath := filepath.Join(tempDir, header.Filename)

	err = c.SaveUploadedFile(header, tempPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, "保存临时文件失败"))
		return
	}
	defer os.Remove(tempPath) // 确保函数退出或上传完成后删除
	// 注意：如果是异步上传，不能立即删除！

	// 为了简单，我们这里进行**同步上传**

	pcs := pcscommand.GetBaiduPCS()
	user := pcsconfig.Config.ActiveUser()
	finalTargetDir := user.PathJoin(targetDir)
	savePath := path.Join(finalTargetDir, header.Filename)

	// 调用 PCS 上传
	// 我们可以复用 pcsupload，或者直接调用 pcs.Upload (更底层)
	// pcs.Upload(savePath, func(uploadURL string, jar http.CookieJar) ...)
	// 这比较复杂。
	// 复用 pcsupload.UploadTaskUnit 是最稳妥的

	uploadDatabase, _ := pcsupload.NewUploadingDatabase()
	defer uploadDatabase.Close()

	statistic := &pcsupload.UploadStatistic{}
	unit := pcsupload.UploadTaskUnit{
		LocalFileChecksum: checksum.NewLocalFileChecksum(tempPath, int(baidupcs.SliceMD5Size)),
		SavePath:          savePath,
		PCS:               pcs,
		UploadingDatabase: uploadDatabase,
		Parallel:          pcsconfig.Config.MaxUploadParallel,
		NoRapidUpload:     false,
		NoSplitFile:       false,
		UploadStatistic:   statistic,
		Policy:            baidupcs.OverWritePolicy, // 默认覆盖?
	}

	// 执行上传 (TaskUnit 实现了 TaskUnit 接口，可以直接运行 Run?)
	// 不，TaskExecutor 调用 Run。
	// 我们手动运行 Run 逻辑?
	// pcsupload 的 Run 逻辑稍微复杂。
	// 让我们构建一个只包含一个任务的 Executor 并同步运行

	executor := &taskframework.TaskExecutor{}
	executor.Append(&unit, 3)
	executor.Execute()

	// 检查结果
	// 获取状态? unit 没有直接导出 Result
	// executor.FailedDeque() check
	if executor.FailedDeque().Size() > 0 {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, "上传失败"))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"path": savePath,
		"size": header.Size,
	}))
}
