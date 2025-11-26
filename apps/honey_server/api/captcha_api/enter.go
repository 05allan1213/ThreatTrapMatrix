package captcha_api

// File: captcha_api.go
// Description: 验证码接口模块，提供图片验证码生成接口及相关数据结构定义

import (
	"ThreatTrapMatrix/apps/honey_server/utils/captcha"
	"ThreatTrapMatrix/apps/honey_server/utils/response"

	"github.com/gin-gonic/gin"
	"github.com/mojocn/base64Captcha"
	"github.com/sirupsen/logrus"
)

// CaptchaApi 验证码接口处理结构体
type CaptchaApi struct{}

// GenerateResponse 验证码生成接口的响应结构体
type GenerateResponse struct {
	CaptchaID string `json:"captchaID"` // 验证码唯一标识ID
	Captcha   string `json:"captcha"`   // 验证码图片Base64编码字符串
}

// GenerateView 生成图片验证码的接口处理函数
func (CaptchaApi) GenerateView(c *gin.Context) {
	// 配置图片验证码参数：尺寸、干扰线、字符长度及来源等
	driver := base64Captcha.DriverString{
		Width:           200,          // 验证码图片宽度
		Height:          60,           // 验证码图片高度
		NoiseCount:      2,            // 验证码图片干扰点数量
		ShowLineOptions: 4,            // 验证码图片干扰线样式
		Length:          4,            // 验证码字符长度
		Source:          "0123456789", // 验证码字符来源（数字）
	}

	// 创建验证码实例，使用自定义存储
	cp := base64Captcha.NewCaptcha(&driver, captcha.CaptchaStore)
	// 生成验证码ID、Base64图片、字符值及错误信息
	id, b64s, _, err := cp.Generate()
	if err != nil {
		logrus.Errorf("图片验证码生成失败 %s", err)
		response.FailWithMsg("图片验证码生成失败", c)
		return
	}

	// 返回验证码ID和Base64图片数据
	response.OkWithData(GenerateResponse{
		CaptchaID: id,
		Captcha:   b64s,
	}, c)
}
