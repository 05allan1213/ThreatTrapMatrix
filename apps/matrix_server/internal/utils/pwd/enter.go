package pwd

// File: matrix_server/utils/pwd/enter.go
// Description: 密码安全处理模块，基于bcrypt算法实现密码加密与校验功能

import "golang.org/x/crypto/bcrypt"

// GenerateFromPassword 使用bcrypt算法对明文密码进行加密
func GenerateFromPassword(password string) (string, error) {
	// 使用默认加密成本（Cost）生成哈希密码
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedPassword), nil
}

// CompareHashAndPassword 校验明文密码与哈希密码是否匹配
func CompareHashAndPassword(hashedPassword string, password string) bool {
	// 对比哈希密码与明文密码的一致性
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}
