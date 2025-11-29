package core

// File: image_server/core/ip_addr.go
// Description: IP地址解析核心模块，基于ip2region数据库实现IP地址到地理位置的解析

import (
	_ "embed"
	"fmt"
	"image_server/internal/utils/ip"
	"strings"

	"github.com/lionsoul2014/ip2region/binding/golang/xdb"
	"github.com/sirupsen/logrus"
)

// searcher 全局ip2region数据库搜索器实例
var searcher *xdb.Searcher

// addrDB 嵌入的ip2region数据库文件内容（二进制）
//
//go:embed ip2region.xdb
var addrDB []byte

// InitIPDB 初始化IP地址数据库，加载嵌入式的ip2region.xdb文件
func InitIPDB() {
	// 从内存缓冲区创建searcher实例
	_searcher, err := xdb.NewWithBuffer(xdb.IPv4, addrDB)
	if err != nil {
		logrus.Fatalf("ip地址数据库加载失败 %s", err)
		return
	}
	searcher = _searcher
}

// GetIpAddr 根据IP地址解析对应的地理位置信息
func GetIpAddr(_ip string) (addr string) {
	// 内网IP直接返回"内网"
	if ip.HasLocalIPAddr(_ip) {
		return "内网"
	}

	// ip2region数据库未初始化
	if searcher == nil {
		logrus.Error("ip地址数据库未初始化，无法解析IP")
		return "数据库未初始化"
	}

	// 从ip2region数据库中查询IP对应的地理位置信息
	region, err := searcher.SearchByStr(_ip)
	if err != nil {
		logrus.Warnf("错误的ip地址 %s", err)
		return "异常地址"
	}

	// 分割查询结果（格式：国家|区域|省份|城市|运营商）
	_addrList := strings.Split(region, "|")
	if len(_addrList) != 5 {
		logrus.Warnf("异常的ip地址 %s", _ip)
		return "未知地址"
	}

	// 提取各部分地理信息
	country := _addrList[0]  // 国家
	province := _addrList[2] // 省份
	city := _addrList[3]     // 城市

	// 按优先级格式化地理位置描述
	if province != "0" && city != "0" {
		return fmt.Sprintf("%s·%s", province, city)
	}
	if country != "0" && province != "0" {
		return fmt.Sprintf("%s·%s", country, province)
	}
	if country != "0" {
		return country
	}
	return region
}
