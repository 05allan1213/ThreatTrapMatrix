package core

// File: honey_node/core/manf.go
// Description: OUI数据库管理模块，实现基于MAC地址前缀（OUI）的厂商信息查询，支持IEEE标准OUI数据加载与快速查询

import (
	"bufio"
	"bytes"
	_ "embed"
	"fmt"
	"regexp"
	"strings"

	"github.com/sirupsen/logrus"
)

// OUIDatabase OUI数据库结构体，存储OUI前缀到厂商名称的映射关系
type OUIDatabase struct {
	vendors map[string]string // 键：6位OUI字符串（大写无分隔符），值：厂商名称
}

// NewOUIDatabase 创建空的OUI数据库实例
func NewOUIDatabase() *OUIDatabase {
	return &OUIDatabase{
		vendors: make(map[string]string),
	}
}

// LoadFromIEEE 从IEEE标准格式的oui.txt文件加载OUI数据
func (db *OUIDatabase) LoadFromIEEE(reader *bufio.Reader) error {
	// 正则表达式匹配IEEE oui.txt格式行：OUI(十六进制) + 厂商名称
	// 匹配示例：00-1B-44   (hex)  Intel Corporation
	re := regexp.MustCompile(`^([0-9A-Fa-f]{2}[-:][0-9A-Fa-f]{2}[-:][0-9A-Fa-f]{2})\s+\(hex\)\s+(.*)$`)

	scanner := bufio.NewScanner(reader)
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue // 跳过空行
		}

		// 匹配OUI记录行
		matches := re.FindStringSubmatch(line)
		if len(matches) != 3 {
			continue // 忽略注释行、标题行等非记录行
		}

		// 标准化OUI格式：去除分隔符（-/:），转为大写
		ouiRaw := matches[1]
		oui := strings.ToUpper(strings.ReplaceAll(ouiRaw, "-", ""))
		oui = strings.ReplaceAll(oui, ":", "") // 兼容冒号分隔的OUI格式

		// 提取并清理厂商名称
		vendor := strings.TrimSpace(matches[2])

		// 存入映射表
		db.vendors[oui] = vendor
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("扫描文件错误: %v", err)
	}

	fmt.Printf("成功加载OUI数据库，共加载 %d 条记录\n", len(db.vendors))
	return nil
}

// LookupVendor 通过MAC地址查询对应的厂商名称
func (db *OUIDatabase) LookupVendor(mac string) (string, bool) {
	// 标准化MAC地址：去除所有分隔符（:/-/.），转为大写
	mac = strings.ToUpper(
		strings.ReplaceAll(
			strings.ReplaceAll(
				strings.ReplaceAll(mac, ":", ""),
				"-", ""),
			".", ""),
	)

	// 提取OUI部分（前6位）
	if len(mac) < 6 {
		return "", false // MAC地址格式无效
	}
	oui := mac[:6]

	vendor, exists := db.vendors[oui]
	return vendor, exists
}

//go:embed oui.txt
var oui []byte

// 全局OUI数据库实例
var manufDB *OUIDatabase

// init 初始化函数，程序启动时加载OUI数据库
func init() {
	manufDB = NewOUIDatabase()
	var err error
	// 从嵌入的oui.txt数据加载OUI信息
	err = manufDB.LoadFromIEEE(bufio.NewReader(bytes.NewReader(oui)))
	if err != nil {
		logrus.Fatalf("加载OUI数据库失败: %v", err)
		return
	}
}

// ManufQuery 对外提供的MAC地址厂商查询接口
func ManufQuery(mac string) (string, bool) {
	return manufDB.LookupVendor(mac)
}
