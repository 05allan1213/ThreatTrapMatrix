package user_api

// File: honey_server/api/user_api/list.go
// Description: 用户列表查询API接口

import (
	"honey_server/internal/middleware"
	models2 "honey_server/internal/models"
	"honey_server/internal/service/common_service"
	"honey_server/internal/utils/response"

	"github.com/gin-gonic/gin"
)

// UserListRequest 用户列表查询请求参数结构体
// 包含分页信息和用户名筛选条件
type UserListRequest struct {
	models2.PageInfo        // 嵌入分页信息结构体（页码、页大小）
	Username         string `form:"username"` // 用户名筛选条件（支持模糊查询）
}

// UserListView 用户列表查询接口处理函数
func (UserApi) UserListView(c *gin.Context) {
	// 获取绑定的列表查询请求参数
	cr := middleware.GetBind[UserListRequest](c)

	// 调用通用查询服务获取用户列表及总数
	// 支持用户名模糊查询、分页及按创建时间倒序排序
	list, count, _ := common_service.QueryList(models2.UserModel{Username: cr.Username}, common_service.QueryListRequest{
		Likes:    []string{"username"}, // 用户名字段支持模糊查询
		PageInfo: cr.PageInfo,          // 分页参数
		Sort:     "created_at desc",    // 排序规则：按创建时间倒序
	})

	// 返回列表数据及总数（统一响应格式）
	response.OkWithList(list, count, c)
}
