package common_service

// File: matrix_server/service/common_service/query_list.go
// Description: 通用查询服务模块，提供支持分页、模糊查询、排序等功能的通用列表查询能力

import (
	"matrix_server/internal/core"
	"matrix_server/internal/models"
	"fmt"

	"gorm.io/gorm"
)

// QueryListRequest 通用查询请求参数结构体
// 封装分页、排序、模糊查询、预加载等通用查询条件
type QueryListRequest struct {
	Debug    bool            // 调试模式开关（开启时打印SQL）
	Likes    []string        // 支持模糊查询的字段列表
	Where    *gorm.DB        // 自定义Where条件
	Preload  []string        // 需要预加载的关联字段列表
	Sort     string          // 排序规则
	PageInfo models.PageInfo // 分页信息（页码、页大小、搜索关键词）
}

// QueryList 通用列表查询函数（泛型实现）
func QueryList[T any](model T, req QueryListRequest) (list []T, count int64, err error) {
	// 获取数据库连接实例
	db := core.GetDB()

	// 调试模式：开启SQL日志打印
	if req.Debug {
		db = db.Debug()
	}

	// 预加载关联字段
	for _, s := range req.Preload {
		db = db.Preload(s)
	}

	// 字段精确匹配查询（基于传入的model实例）
	db = db.Where(model)

	// 应用自定义Where条件（高级查询）
	if req.Where != nil {
		db = db.Where(req.Where)
	}

	// 模糊查询处理（基于PageInfo.Key和Likes字段列表）
	if req.PageInfo.Key != "" {
		like := core.GetDB().Where("")
		for _, column := range req.Likes {
			like.Or(fmt.Sprintf("%s like ?", column), fmt.Sprintf("%%%s%%", req.PageInfo.Key))
		}
		db = db.Where(like)
	}

	// 分页参数处理（设置默认值）
	if req.PageInfo.Limit <= 0 {
		req.PageInfo.Limit = 10 // 默认每页10条
	}
	if req.PageInfo.Page <= 0 {
		req.PageInfo.Page = 1 // 默认第1页
	}
	offset := (req.PageInfo.Page - 1) * req.PageInfo.Limit // 计算偏移量

	// 执行分页查询（带排序）
	err = db.Offset(offset).Limit(req.PageInfo.Limit).Order(req.Sort).Find(&list).Error
	// 查询总记录数（用于分页计算）
	err = db.Count(&count).Error

	return
}
