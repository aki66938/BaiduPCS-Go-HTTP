package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/qjfoidnh/BaiduPCS-Go/api/model"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcscommand"
)

// RecycleList 列出回收站
// RecycleList 列出回收站
// @Summary 列出回收站
// @Description 列出回收站中的文件
// @Tags 回收站
// @Accept json
// @Produce json
// @Param page query int false "页码"
// @Success 200 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/recycle/list [get]
func RecycleList(c *gin.Context) {
	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pcs := pcscommand.GetBaiduPCS()
	files, err := pcs.RecycleList(page)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"page":  page,
		"files": files,
	}))
}

// RecycleRestore 恢复回收站文件
// RecycleRestore 恢复回收站文件
// @Summary 恢复文件
// @Description 恢复回收站中的文件
// @Tags 回收站
// @Accept json
// @Produce json
// @Param request body model.RecycleRestoreRequest true "恢复请求"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/recycle/restore [post]
func RecycleRestore(c *gin.Context) {
	var req model.RecycleRestoreRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	pcs := pcscommand.GetBaiduPCS()
	_, err := pcs.RecycleRestore(req.FsIDs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"restored": req.FsIDs,
	}))
}

// RecycleDelete 彻底删除回收站文件
// RecycleDelete 彻底删除回收站文件
// @Summary 彻底删除
// @Description 彻底删除回收站中的文件（不可恢复）
// @Tags 回收站
// @Accept json
// @Produce json
// @Param request body model.RecycleDeleteRequest true "删除请求"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/recycle/delete [post]
func RecycleDelete(c *gin.Context) {
	var req model.RecycleDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	pcs := pcscommand.GetBaiduPCS()
	err := pcs.RecycleDelete(req.FsIDs...)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"deleted": req.FsIDs,
	}))
}

// RecycleClear 清空回收站
// RecycleClear 清空回收站
// @Summary 清空回收站
// @Description 清空回收站中所有文件
// @Tags 回收站
// @Accept json
// @Produce json
// @Success 200 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/recycle/clear [post]
func RecycleClear(c *gin.Context) {
	pcs := pcscommand.GetBaiduPCS()
	num, err := pcs.RecycleClear()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"message": "回收站已清空",
		"count":   num,
	}))
}
