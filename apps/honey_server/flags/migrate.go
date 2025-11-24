package flags

// File: honey_server/flags/migrate.go
// Description: 负责执行GORM自动迁移以创建或更新数据表结构

import (
	"ThreatTrapMatrix/apps/honey_server/global"
	"ThreatTrapMatrix/apps/honey_server/models"

	"github.com/sirupsen/logrus"
)

// Migrate 执行数据库表结构自动迁移
func Migrate() {
	// 自动迁移指定的模型结构体到数据库，生成或更新数据表结构
	err := global.DB.AutoMigrate(
		&models.HoneyIpModel{},
		&models.HoneyPortModel{},
		&models.HostModel{},
		&models.HostTemplateModel{},
		&models.ImageModel{},
		&models.LogModel{},
		&models.MatrixTemplateModel{},
		&models.NetModel{},
		&models.NodeModel{},
		&models.NodeNetworkModel{},
		&models.ServiceModel{},
		&models.UserModel{},
	)
	if err != nil {
		logrus.Fatalf("表结构迁移失败 %s", err)
	}
	logrus.Infof("表结构迁移成功")
}
