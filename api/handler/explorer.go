package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/qjfoidnh/BaiduPCS-Go/api/model"
	"github.com/qjfoidnh/BaiduPCS-Go/baidupcs"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcscommand"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcsconfig"
)

// ListFiles 列出文件
// @Summary 列出文件和目录
// @Description 列出指定路径下的文件和目录
// @Tags 文件管理
// @Accept json
// @Produce json
// @Param request body model.ListRequest true "列表请求"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Router /api/ls [post]
func ListFiles(c *gin.Context) {
	var req model.ListRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	if req.Path == "" {
		req.Path = "."
	}

	activeUser := pcsconfig.Config.ActiveUser()
	if activeUser == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "未登录"))
		return
	}

	targetPath, err := matchPath(req.Path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	orderOpt := &baidupcs.OrderOptions{
		By:    baidupcs.OrderByName,
		Order: baidupcs.OrderAsc,
	}
	if req.Order == "time" {
		orderOpt.By = baidupcs.OrderByTime
	} else if req.Order == "size" {
		orderOpt.By = baidupcs.OrderBySize
	}
	if req.Desc {
		orderOpt.Order = baidupcs.OrderDesc
	}

	pcs := pcscommand.GetBaiduPCS()
	files, err := pcs.FilesDirectoriesList(targetPath, orderOpt)

	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	var fileInfos []model.FileInfo
	for _, f := range files {
		fileInfos = append(fileInfos, model.FileInfo{
			Path:     f.Path,
			Filename: f.Filename,
			IsDir:    f.Isdir,
			Size:     f.Size,
			MD5:      f.MD5,
			MTime:    f.Mtime,
		})
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"path":  targetPath,
		"files": fileInfos,
	}))
}

// Meta 获取元数据
// Meta 获取元数据
// @Summary 获取文件元数据
// @Description 获取指定文件或目录的详细信息
// @Tags 文件管理
// @Accept json
// @Produce json
// @Param request body model.MetaRequest true "元数据请求"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/meta [post]
func Meta(c *gin.Context) {
	var req model.MetaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	paths, err := matchPaths(req.Paths...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	pcs := pcscommand.GetBaiduPCS()
	var fileInfos []model.FileInfo

	for _, p := range paths {
		f, err := pcs.FilesDirectoriesMeta(p)
		if err != nil {
			c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
			return
		}
		fileInfos = append(fileInfos, model.FileInfo{
			Path:     f.Path,
			Filename: f.Filename,
			IsDir:    f.Isdir,
			Size:     f.Size,
			MD5:      f.MD5,
			MTime:    f.Mtime,
		})
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"files": fileInfos,
	}))
}

// Search 搜索文件
// Search 搜索文件
// @Summary 搜索文件
// @Description 在指定目录下搜索文件
// @Tags 文件管理
// @Accept json
// @Produce json
// @Param keyword query string true "搜索关键字"
// @Param path query string false "搜索路径 (默认根目录)"
// @Param recurse query bool false "是否递归"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/search [get]
func Search(c *gin.Context) {
	var req model.SearchRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	if req.Path == "" {
		req.Path = "."
	}

	targetPath, err := matchPath(req.Path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	pcs := pcscommand.GetBaiduPCS()
	files, err := pcs.Search(targetPath, req.Keyword, req.Recurse)

	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	var fileInfos []model.FileInfo
	for _, f := range files {
		fileInfos = append(fileInfos, model.FileInfo{
			Path:     f.Path,
			Filename: f.Filename,
			Size:     f.Size,
			IsDir:    f.Isdir,
			MTime:    f.Mtime,
			MD5:      f.MD5,
		})
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"files": fileInfos,
	}))
}

// Pwd 获取当前工作目录
// Pwd 获取当前工作目录
// @Summary 获取当前工作目录
// @Description 获取当前用户的工作目录路径
// @Tags 工作目录
// @Accept json
// @Produce json
// @Success 200 {object} model.Response
// @Failure 401 {object} model.Response
// @Router /api/pwd [get]
func Pwd(c *gin.Context) {
	activeUser := pcsconfig.Config.ActiveUser()
	if activeUser == nil {
		c.JSON(http.StatusUnauthorized, model.ErrorResponse(401, "未登录"))
		return
	}
	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"path": activeUser.Workdir,
	}))
}

// Cd 切换工作目录
// Cd 切换工作目录
// @Summary 切换工作目录
// @Description 改变当前用户的工作目录
// @Tags 工作目录
// @Accept application/x-www-form-urlencoded
// @Produce json
// @Param path formData string true "目标路径"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/cd [post]
func Cd(c *gin.Context) {
	path := c.PostForm("path")
	if path == "" {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "path 参数不能为空"))
		return
	}

	targetPath, err := matchPath(path)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	pcs := pcscommand.GetBaiduPCS()
	f, err := pcs.FilesDirectoriesMeta(targetPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, "目录不存在"))
		return
	}
	if !f.Isdir {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "目标不是目录"))
		return
	}

	user := pcsconfig.Config.ActiveUser()
	user.Workdir = targetPath

	// 保存配置
	err = pcsconfig.Config.Save()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, "保存配置失败"))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"path": targetPath,
	}))
}
