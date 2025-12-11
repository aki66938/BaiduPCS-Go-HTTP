package handler

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/qjfoidnh/BaiduPCS-Go/api/model"
	"github.com/qjfoidnh/BaiduPCS-Go/baidupcs"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcscommand"
	"github.com/qjfoidnh/BaiduPCS-Go/internal/pcsconfig"
)

// CloudDlAdd 添加离线下载任务
// CloudDlAdd 添加离线下载任务
// @Summary 添加离线下载
// @Description 添加新的离线下载任务
// @Tags 离线下载
// @Accept json
// @Produce json
// @Param request body model.CloudAddRequest true "添加任务请求"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/cloud/add [post]
func CloudDlAdd(c *gin.Context) {
	var req model.CloudAddRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	savePath := req.SavePath
	// 使用 matchPath 处理 savePath, 支持相对路径
	finalSavePath, err := matchPath(savePath)
	if err != nil {
		// 如果路径不存在，尝试作为相对路径拼接到工作目录
		user := pcsconfig.Config.ActiveUser()
		finalSavePath = user.PathJoin(savePath)
	}

	pcs := pcscommand.GetBaiduPCS()
	var taskIDs []int64
	var errors []string

	for _, url := range req.SourceURLs {
		taskID, err := pcs.CloudDlAddTask(url, finalSavePath+baidupcs.PathSeparator)
		if err != nil {
			errors = append(errors, fmt.Sprintf("%s: %s", url, err.Error()))
		} else {
			taskIDs = append(taskIDs, taskID)
		}
	}

	if len(taskIDs) == 0 && len(errors) > 0 {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, strings.Join(errors, "; ")))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"task_ids":  taskIDs,
		"errors":    errors,
		"save_path": finalSavePath,
	}))
}

// CloudDlQuery 查询离线下载任务
// CloudDlQuery 查询离线下载任务
// @Summary 查询离线任务
// @Description 查询指定 ID 的离线任务状态
// @Tags 离线下载
// @Accept json
// @Produce json
// @Param request body model.CloudQueryRequest true "查询任务请求"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/cloud/query [post]
func CloudDlQuery(c *gin.Context) {
	var req model.CloudQueryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	pcs := pcscommand.GetBaiduPCS()
	tasks, err := pcs.CloudDlQueryTask(req.TaskIDs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"tasks": tasks,
	}))
}

// CloudDlList 列出离线下载任务
// CloudDlList 列出离线下载任务
// @Summary 列出离线任务
// @Description 列出所有的离线下载任务
// @Tags 离线下载
// @Accept json
// @Produce json
// @Success 200 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/cloud/list [get]
func CloudDlList(c *gin.Context) {
	pcs := pcscommand.GetBaiduPCS()
	tasks, err := pcs.CloudDlListTask()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"tasks": tasks,
	}))
}

// CloudDlCancel 取消离线下载任务
// CloudDlCancel 取消离线下载任务
// @Summary 取消离线任务
// @Description 取消正在进行的离线下载任务
// @Tags 离线下载
// @Accept json
// @Produce json
// @Param request body model.CloudTaskIDsRequest true "取消任务请求"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/cloud/cancel [post]
func CloudDlCancel(c *gin.Context) {
	var req model.CloudTaskIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	pcs := pcscommand.GetBaiduPCS()
	var cancelled []int64
	for _, id := range req.TaskIDs {
		err := pcs.CloudDlCancelTask(id)
		if err == nil {
			cancelled = append(cancelled, id)
		}
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"cancelled": cancelled,
	}))
}

// CloudDlDelete 删除离线下载任务
// CloudDlDelete 删除离线下载任务
// @Summary 删除离线任务
// @Description 删除离线下载任务记录
// @Tags 离线下载
// @Accept json
// @Produce json
// @Param request body model.CloudTaskIDsRequest true "删除任务请求"
// @Success 200 {object} model.Response
// @Failure 400 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/cloud/delete [post]
func CloudDlDelete(c *gin.Context) {
	var req model.CloudTaskIDsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, model.ErrorResponse(400, err.Error()))
		return
	}

	pcs := pcscommand.GetBaiduPCS()
	var deleted []int64
	for _, id := range req.TaskIDs {
		err := pcs.CloudDlDeleteTask(id)
		if err == nil {
			deleted = append(deleted, id)
		}
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"deleted": deleted,
	}))
}

// CloudDlClear 清空离线下载任务
// CloudDlClear 清空离线下载任务
// @Summary 清空离线任务
// @Description 清空所有离线下载任务记录
// @Tags 离线下载
// @Accept json
// @Produce json
// @Success 200 {object} model.Response
// @Failure 500 {object} model.Response
// @Router /api/cloud/clear [post]
func CloudDlClear(c *gin.Context) {
	pcs := pcscommand.GetBaiduPCS()
	total, err := pcs.CloudDlClearTask()
	if err != nil {
		c.JSON(http.StatusInternalServerError, model.ErrorResponse(500, err.Error()))
		return
	}

	c.JSON(http.StatusOK, model.SuccessResponse(gin.H{
		"total": total,
	}))
}
