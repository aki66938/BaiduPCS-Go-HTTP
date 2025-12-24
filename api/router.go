package api

import (
	"github.com/gin-gonic/gin"
	"github.com/qjfoidnh/BaiduPCS-Go/api/handler"
	"github.com/qjfoidnh/BaiduPCS-Go/api/middleware"
	_ "github.com/qjfoidnh/BaiduPCS-Go/docs"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"
)

// SetupRouter 设置路由
func SetupRouter(username, password string, enableAuth bool) *gin.Engine {
	// 设置 Gin 模式
	gin.SetMode(gin.ReleaseMode)

	r := gin.New()

	// 使用中间件
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS())

	// Swagger 文档
	r.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler))

	// API 路由组
	api := r.Group("/api")

	// 如果启用认证，添加 Basic Auth 中间件
	if enableAuth {
		api.Use(middleware.BasicAuth(username, password))
	}

	{
		// 文件管理接口
		api.POST("/ls", handler.ListFiles)  // 列出文件
		api.POST("/mkdir", handler.MakeDir) // 创建目录
		api.POST("/rm", handler.Remove)     // 删除文件
		api.POST("/mv", handler.Move)       // 移动/重命名
		api.POST("/cp", handler.Copy)       // 复制文件
		api.POST("/meta", handler.Meta)     // 获取元数据
		api.GET("/search", handler.Search)  // 搜索文件

		// 工作目录管理
		api.GET("/pwd", handler.Pwd) // 获取当前目录
		api.POST("/cd", handler.Cd)  // 切换目录

		// 上传下载接口
		api.POST("/upload", handler.Upload)     // 上传文件
		api.POST("/download", handler.Download) // 下载文件
		api.POST("/locate", handler.Locate)     // 获取直链

		recycle := api.Group("/recycle")
		{
			recycle.GET("/list", handler.RecycleList)        // 列出回收站
			recycle.POST("/restore", handler.RecycleRestore) // 恢复文件
			recycle.POST("/delete", handler.RecycleDelete)   // 彻底删除
			recycle.POST("/clear", handler.RecycleClear)     // 清空回收站
		}

		// 分享管理接口
		share := api.Group("/share")
		{
			share.POST("/set", handler.ShareSet)       // 创建分享
			share.GET("/list", handler.ShareList)      // 列出分享
			share.POST("/cancel", handler.ShareCancel) // 取消分享
		}

		// 转存接口
		api.POST("/transfer", handler.Transfer) // 转存分享链接

		// 离线下载接口
		cloud := api.Group("/cloud")
		{
			cloud.POST("/add", handler.CloudDlAdd)       // 添加离线任务
			cloud.POST("/query", handler.CloudDlQuery)   // 查询离线任务
			cloud.GET("/list", handler.CloudDlList)      // 列出离线任务
			cloud.POST("/cancel", handler.CloudDlCancel) // 取消离线任务
			cloud.POST("/delete", handler.CloudDlDelete) // 删除离线任务
			cloud.POST("/clear", handler.CloudDlClear)   // 清空离线任务
		}

		// 账号管理接口
		auth := api.Group("/auth")
		{
			auth.POST("/login", handler.Login)   // 登录
			auth.POST("/logout", handler.Logout) // 登出
			// 扫码登录
			auth.POST("/qrcode", handler.QRCodeGet)          // 获取二维码
			auth.GET("/qrcode/status", handler.QRCodeStatus) // 查询扫码状态
			auth.POST("/qrcode/login", handler.QRCodeLogin)  // 完成扫码登录
		}

		account := api.Group("/account")
		{
			account.GET("/who", handler.Who)        // 当前账号信息
			account.GET("/quota", handler.Quota)    // 账号配额
			account.GET("/list", handler.UserList)  // 账号列表
			account.POST("/switch", handler.Switch) // 切换账号
		}

		// 配置管理接口
		config := api.Group("/config")
		{
			config.GET("", handler.ConfigGet)      // 获取配置
			config.POST("/set", handler.ConfigSet) // 设置配置
		}

		// 健康检查
		api.GET("/health", handler.Health)
	}

	return r
}
