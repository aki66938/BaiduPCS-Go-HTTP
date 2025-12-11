package model

// Response 统一响应结构
type Response struct {
	Code    int         `json:"code"`    // 状态码：0-成功，非0-失败
	Message string      `json:"message"` // 响应消息
	Data    interface{} `json:"data,omitempty"` // 响应数据
}

// SuccessResponse 成功响应
func SuccessResponse(data interface{}) Response {
	return Response{
		Code:    0,
		Message: "success",
		Data:    data,
	}
}

// ErrorResponse 错误响应
func ErrorResponse(code int, message string) Response {
	return Response{
		Code:    code,
		Message: message,
	}
}

// PageData 分页数据
type PageData struct {
	Total int         `json:"total"` // 总数
	Items interface{} `json:"items"` // 数据列表
}
