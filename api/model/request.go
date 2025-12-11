package model

// UploadRequest 上传请求
type UploadRequest struct {
	TargetPath string `form:"target_path" binding:"required"` // 目标路径
	Policy     string `form:"policy"`                         // 上传策略：skip/overwrite/rsync
	NoRapid    bool   `form:"norapid"`                        // 是否跳过秒传
}

// UploadResponse 上传响应
type UploadResponse struct {
	Path     string `json:"path"`          // 文件路径
	Size     int64  `json:"size"`          // 文件大小
	MD5      string `json:"md5,omitempty"` // 文件MD5
	IsRapid  bool   `json:"is_rapid"`      // 是否秒传
	Verified bool   `json:"verified"`      // 是否已验证
}

// ListRequest 文件列表请求
type ListRequest struct {
	Path  string `json:"path" form:"path"`   // 路径
	Order string `json:"order" form:"order"` // 排序字段：name/time/size
	Desc  bool   `json:"desc" form:"desc"`   // 是否降序
}

// FileInfo 文件信息
type FileInfo struct {
	Path     string `json:"path"`          // 文件路径
	Filename string `json:"filename"`      // 文件名
	IsDir    bool   `json:"is_dir"`        // 是否目录
	Size     int64  `json:"size"`          // 文件大小
	MD5      string `json:"md5,omitempty"` // 文件MD5
	MTime    int64  `json:"mtime"`         // 修改时间
}

// DeleteRequest 删除请求
type DeleteRequest struct {
	Paths []string `json:"paths" binding:"required,min=1"` // 要删除的路径列表
}

// MkdirRequest 创建目录请求
type MkdirRequest struct {
	Path string `json:"path" binding:"required"` // 目录路径
}

// MoveRequest 移动/重命名请求
type MoveRequest struct {
	FromPaths []string `json:"from_paths" binding:"required,min=1"` // 源路径列表
	ToPath    string   `json:"to_path" binding:"required"`          // 目标路径
}

// CopyRequest 复制请求
type CopyRequest struct {
	FromPaths []string `json:"from_paths" binding:"required,min=1"` // 源路径列表
	ToPath    string   `json:"to_path" binding:"required"`          // 目标路径
}

// MetaRequest 元数据请求
type MetaRequest struct {
	Paths []string `json:"paths" binding:"required,min=1"` // 路径列表
}

// SearchRequest 搜索请求
type SearchRequest struct {
	Keyword string `form:"keyword" binding:"required"` // 搜索关键词
	Path    string `form:"path"`                       // 搜索路径
	Recurse bool   `form:"recurse"`                    // 是否递归搜索
}

// DownloadRequest 下载请求
type DownloadRequest struct {
	Paths  []string `json:"paths" binding:"required,min=1"` // 要下载的路径列表
	Save   bool     `json:"save"`                           // 是否保存到本地
	SaveTo string   `json:"save_to"`                        // 保存路径
}

// RecycleRestoreRequest 回收站恢复请求
type RecycleRestoreRequest struct {
	FsIDs []int64 `json:"fsids" binding:"required,min=1"` // 要恢复的文件ID列表
}

// RecycleDeleteRequest 回收站删除请求
type RecycleDeleteRequest struct {
	FsIDs []int64 `json:"fsids" binding:"required,min=1"` // 要删除的文件ID列表
}

// ShareSetRequest 创建分享请求
type ShareSetRequest struct {
	Paths    []string `json:"paths" binding:"required,min=1"` // 要分享的路径列表
	Password string   `json:"password"`                       // 提取码（可选）
	Period   int      `json:"period"`                         // 有效期（天数，0表示永久）
}

// ShareCancelRequest 取消分享请求
type ShareCancelRequest struct {
	ShareIDs []int64 `json:"share_ids" binding:"required,min=1"` // 要取消的分享ID列表
}

// TransferRequest 转存请求
type TransferRequest struct {
	ShareURL    string `json:"share_url" binding:"required"` // 分享链接
	ExtractCode string `json:"extract_code"`                 // 提取码（可选）
	Collect     bool   `json:"collect"`                      // 是否归档到同名目录
	Download    bool   `json:"download"`                     // 转存后是否自动下载
}

// CloudAddRequest 添加离线任务请求
type CloudAddRequest struct {
	SourceURLs []string `json:"source_urls" binding:"required,min=1"` // 下载源地址列表
	SavePath   string   `json:"save_path" binding:"required"`         // 保存路径
}

// CloudQueryRequest 查询离线任务请求
type CloudQueryRequest struct {
	TaskIDs []int64 `json:"task_ids" binding:"required,min=1"` // 任务ID列表
}

// CloudTaskIDsRequest 离线任务ID请求
type CloudTaskIDsRequest struct {
	TaskIDs []int64 `json:"task_ids" binding:"required,min=1"` // 任务ID列表
}

type LoginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	BDUSS    string `json:"bduss"`
	VCode    string `json:"vcode"`
	VCodeStr string `json:"vcodestr"`
}

type UserSwitchRequest struct {
	UID uint64 `json:"uid" binding:"required"`
}

// ServerUploadRequest 服务器本地文件上传请求
type ServerUploadRequest struct {
	LocalPaths []string `json:"local_paths" binding:"required"`
	TargetDir  string   `json:"target_dir" binding:"required"`
	Policy     string   `json:"policy"`
}
