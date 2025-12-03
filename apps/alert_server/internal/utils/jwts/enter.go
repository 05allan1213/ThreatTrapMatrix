package jwts

// File: alert_server/utils/jwt/enter.go
// Description: JWT工具模块，提供Token生成、解析及验证功能

import (
	"alert_server/internal/global"
	"errors"
	"time"

	"github.com/dgrijalva/jwt-go"
)

// ClaimsUserInfo Token中存储的用户信息结构体
type ClaimsUserInfo struct {
	UserID uint `json:"userID"` // 用户ID
	Role   int8 `json:"role"`   // 用户角色
}

// Claims JWT完整载荷结构体，包含用户信息和标准Claims
type Claims struct {
	ClaimsUserInfo     // 嵌入用户信息结构体
	jwt.StandardClaims // 嵌入JWT标准Claims
}

// GetToken 根据用户信息生成JWT Token
func GetToken(info ClaimsUserInfo) (string, error) {
	// 获取全局配置中的JWT配置项
	j := global.Config.Jwt
	// 构建JWT载荷
	cla := Claims{
		ClaimsUserInfo: info,
		StandardClaims: jwt.StandardClaims{
			ExpiresAt: time.Now().Add(time.Duration(j.Expires) * time.Second).Unix(), // Token过期时间（从配置读取有效期）
			Issuer:    j.Issuer,                                                      // Token签发人（从配置读取）
		},
	}
	// 使用HS256算法创建Token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, cla)
	// 使用配置中的密钥进行签名，生成最终Token字符串
	return token.SignedString([]byte(j.Secret))
}

// ParseToken 解析并验证JWT Token，返回载荷信息
func ParseToken(tokenString string) (*Claims, error) {
	// 获取全局配置中的JWT配置项
	j := global.Config.Jwt
	// 解析Token并验证签名
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		// 返回签名验证密钥
		return []byte(j.Secret), nil
	})
	if err != nil {
		return nil, err
	}
	// 类型断言获取自定义载荷
	claims, ok := token.Claims.(*Claims)
	// 验证Token有效性及签发人
	if ok && token.Valid {
		if claims.Issuer != j.Issuer {
			return nil, errors.New("invalid issuer") // 签发人不一致
		}
		return claims, nil
	}
	return nil, errors.New("invalid token") // Token无效
}
