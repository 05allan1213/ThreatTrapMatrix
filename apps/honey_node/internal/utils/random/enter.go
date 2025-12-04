package random

// File: honey_node/utils/random/enter.go
// Description: 提供随机字符串生成功能，包含基础随机字符串生成方法及基于内存映射去重的唯一随机字符串生成方法

import "math/rand"

// letters 随机字符串生成的字符集
// 包含大小写字母和数字，用于生成基础随机字符串
var letters = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789")

// maps 内存映射表，用于存储已生成的唯一随机字符串
// 键为已生成的字符串，值为空结构体，用于RandStrV2去重
var maps = map[string]struct{}{}

// RandStr 生成指定长度的随机字符串
func RandStr(n int) string {
	b := make([]rune, n)
	for i := range b {
		// 从letters字符集中随机选取一个字符赋值
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// RandStrV2 生成指定长度的唯一随机字符串（基于内存映射去重）
func RandStrV2(n int) string {
	// 调用基础方法生成随机字符串
	str := RandStr(n)

	// 检查字符串是否已存在于内存映射表中
	_, ok := maps[str]
	if ok {
		// 已存在则递归重新生成
		return RandStrV2(n)
	}

	// 未存在则存入映射表，确保后续不会重复生成
	maps[str] = struct{}{}
	return str
}
