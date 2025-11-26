package captcha

import (
	"github.com/mojocn/base64Captcha"
)

// CaptchaStore 用于存储验证码的全局变量,使用 base64Captcha 提供的默认内存存储实现
var CaptchaStore = base64Captcha.DefaultMemStore
