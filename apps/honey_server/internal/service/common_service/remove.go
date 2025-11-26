package common_service

// File: honey_server/service/common_service/remove.go
// Description: 通用删除服务模块，提供支持条件过滤、ID列表批量删除的通用删除能力

import (
	"ThreatTrapMatrix/apps/honey_server/internal/global"

	"github.com/sirupsen/logrus"
	"gorm.io/gorm"
)

// RemoveRequest 通用删除请求参数结构体
// 封装删除操作所需的条件、日志、调试等配置
type RemoveRequest struct {
	Debug    bool          // 调试模式开关（开启时打印SQL）
	Where    *gorm.DB      // 自定义Where条件
	IDList   []uint        // 需要删除的记录ID列表
	Log      *logrus.Entry // 日志实例（用于记录操作日志）
	Msg      string        // 操作描述信息（用于日志说明）
	Unscoped bool          // 是否使用物理删除
}

// Remove 通用删除函数（泛型实现）
func Remove[T any](model T, req RemoveRequest) (successCount int64, err error) {
	// 获取数据库连接实例（查询用）
	db := global.DB
	// 获取数据库连接实例（删除操作用）
	deleteDB := global.DB

	// 调试模式：开启SQL日志打印
	if req.Debug {
		db = db.Debug()
		deleteDB = deleteDB.Debug()
	}

	// 物理删除
	if req.Unscoped {
		req.Log.Infof("启用物理删除")
		deleteDB = deleteDB.Unscoped()
	}

	// 应用自定义Where条件（高级过滤）
	if req.Where != nil {
		db = db.Where(req.Where)
	}

	// 字段精确匹配过滤（基于传入的model实例）
	db = db.Where(model)

	// 根据ID列表过滤待删除记录
	if len(req.IDList) > 0 {
		req.Log.Infof("删除 %s idList %v", req.Msg, req.IDList)
		db = db.Where("id in ?", req.IDList)
	}

	// 查询待删除的记录列表
	var list []T
	db.Find(&list)

	// 无匹配记录时直接返回
	if len(list) <= 0 {
		req.Log.Infof("没查到")
		return
	}

	// 执行批量删除操作
	result := deleteDB.Delete(&list)
	if result.Error != nil {
		req.Log.Errorf("删除失败 %s", result.Error)
		err = result.Error
		return
	}

	// 记录成功删除的记录数并打印日志
	successCount = result.RowsAffected
	req.Log.Infof("删除 %s 成功, 成功%d个", req.Msg, successCount)
	return
}
