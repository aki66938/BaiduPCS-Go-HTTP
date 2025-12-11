package handler

import (
	"net/http"
	"path"

	"github.com/gin-gonic/gin"
	"github.com/qjfoidnh/BaiduPCS-Go/api/model"
	"github.com/qjfoidnh/BaiduPCS-Go/baidupcs"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcscommand"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcsconfig"
)

// MakeDir 创建目录
// MakeDir 创建目录
// @Summary 创建目录
// @Description 在网盘中创建新目录
// @Tags 文件管理
// @Accept json
// @Produce json
// @Param request body model.MkdirRequest true "创建目录请求"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/mkdir [post]
func MakeDir(c *gin.Context) {
	var req model.MkdirRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	user := pcsconfig.Config.ActiveUser()
	targetPath := user.PathJoin(req.Path)

	pcs := pcscommand.GetBaiduPCS()
	err := pcs.Mkdir(targetPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"path": targetPath,
	}))
}

// Remove 删除文件
// Remove 删除文件
// @Summary 删除文件
// @Description 删除网盘中的文件或目录
// @Tags 文件管理
// @Accept json
// @Produce json
// @Param request body model.DeleteRequest true "删除请求"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/rm [post]
func Remove(c *gin.Context) {
	var req model.DeleteRequest
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
	err = pcs.Remove(paths...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"deleted": paths,
	}))
}

// handleCopyMove 处理复制或移动逻辑 (内部使用)
func handleCopyMove(c *gin.Context, op string, fromPaths []string, toPath string) {
	sources, err := matchPaths(fromPaths...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	user := pcsconfig.Config.ActiveUser()
	dest := user.PathJoin(toPath)
	pcs := pcscommand.GetBaiduPCS()

	destInfo, err := pcs.FilesDirectoriesMeta(dest)
	// 如果目标存在且是目录，则将所有源文件移动/复制到该目录下
	if err == nil && destInfo.Isdir {
		var cj []*baidupcs.CpMvJSON
		for _, s := range sources {
			cj = append(cj, &baidupcs.CpMvJSON{
				From: s,
				To:   path.Join(dest, path.Base(s)),
			})
		}

		if op == "copy" {
			err = pcs.Copy(cj...)
		} else {
			err = pcs.Move(cj...)
		}
	} else {
		// 目标不存在或不是目录
		if len(sources) == 1 {
			// 单个文件，视为重命名
			if op == "copy" {
				err = pcs.Copy(&baidupcs.CpMvJSON{From: sources[0], To: dest})
			} else {
				err = pcs.Rename(sources[0], dest)
			}
		} else {
			// 多个文件但目标不是目录，报错
			c.JSON(http.StatusBadRequest, model.ErrorResponse(400, "目标路径不存在或不是目录，无法批量操作"))
			return
		}
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"sources": sources,
		"dest":    dest,
	}))
}

// Move 移动/重命名文件
// Move 移动/重命名文件
// @Summary 移动/重命名文件
// @Description 移动文件或重命名文件
// @Tags 文件管理
// @Accept json
// @Produce json
// @Param request body model.MoveRequest true "移动/重命名请求"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/mv [post]
func Move(c *gin.Context) {
	var req model.MoveRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}
	handleCopyMove(c, "move", req.FromPaths, req.ToPath)
}

// Copy 复制文件
// Copy 复制文件
// @Summary 复制文件
// @Description 复制文件或目录
// @Tags 文件管理
// @Accept json
// @Produce json
// @Param request body model.CopyRequest true "复制请求"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/cp [post]
func Copy(c *gin.Context) {
	var req model.CopyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}
	handleCopyMove(c, "copy", req.FromPaths, req.ToPath)
}
