package flags

// File: honey_server/flags/migrate.go
// Description: 负责执行GORM自动迁移以创建或更新数据表结构

import (
	"honey_server/internal/global"
	models2 "honey_server/internal/models"

	"github.com/sirupsen/logrus"
)

// Migrate 执行数据库表结构自动迁移
func Migrate() {
	// 自动迁移指定的模型结构体到数据库，生成或更新数据表结构
	err := global.DB.AutoMigrate(
		&models2.HoneyIpModel{},
		&models2.HoneyPortModel{},
		&models2.HostModel{},
		&models2.HostTemplateModel{},
		&models2.ImageModel{},
		&models2.LogModel{},
		&models2.MatrixTemplateModel{},
		&models2.NetModel{},
		&models2.NodeModel{},
		&models2.NodeNetworkModel{},
		&models2.ServiceModel{},
		&models2.UserModel{},
	)
	if err != nil {
		logrus.Fatalf("表结构迁移失败 %s", err)
	}
	logrus.Infof("表结构迁移成功")
}
