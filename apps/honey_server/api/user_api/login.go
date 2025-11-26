package user_api

// File: honey_server/api/user_api/login.go
// Description: 用户登录API接口

import (
	"ThreatTrapMatrix/apps/honey_server/global"
	"ThreatTrapMatrix/apps/honey_server/middleware"
	"ThreatTrapMatrix/apps/honey_server/models"
	"ThreatTrapMatrix/apps/honey_server/service/log_service"
	"ThreatTrapMatrix/apps/honey_server/utils/captcha"
	"ThreatTrapMatrix/apps/honey_server/utils/jwts"
	"ThreatTrapMatrix/apps/honey_server/utils/pwd"
	"ThreatTrapMatrix/apps/honey_server/utils/response"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// LoginRequest 用户登录请求参数结构体
type LoginRequest struct {
	Username    string `json:"username" binding:"required" label:"用户名"` // 用户名（必填）
	Password    string `json:"password" binding:"required" label:"密码"`   // 密码（必填）
	CaptchaID   string `json:"captchaID" binding:"required"`               // 验证码ID（必填）
	CaptchaCode string `json:"captchaCode" binding:"required"`             // 验证码内容（必填）
}

// LoginView 用户登录接口处理函数
func (UserApi) LoginView(c *gin.Context) {
	// 获取绑定的登录请求参数
	cr := middleware.GetBind[LoginRequest](c)
	// 创建用户登录日志服务
	loginLog := log_service.NewLoginLog(c)
	// 校验验证码参数是否完整
	if cr.CaptchaID == "" || cr.CaptchaCode == "" {
		loginLog.FailLog(cr.Username, "", "未输入图片验证码")
		response.FailWithMsg("请输入图片验证码", c)
		return
	}

	// 验证图片验证码有效性
	if !captcha.CaptchaStore.Verify(cr.CaptchaID, cr.CaptchaCode, true) {
		loginLog.FailLog(cr.Username, "", "图片验证码验证失败")
		response.FailWithMsg("图片验证码验证失败", c)
		return
	}

	// 根据用户名查询用户信息
	var user models.UserModel
	err := global.DB.Take(&user, "username = ?", cr.Username).Error
	if err != nil {
		loginLog.FailLog(cr.Username, cr.Password, "用户名不存在")
		response.FailWithMsg("用户名或密码错误", c)
		return
	}

	// 验证密码是否匹配
	if !pwd.CompareHashAndPassword(user.Password, cr.Password) {
		loginLog.FailLog(cr.Username, cr.Password, "密码错误")
		response.FailWithMsg("用户名或密码错误", c)
		return
	}

	// 生成JWT Token
	token, err := jwts.GetToken(jwts.ClaimsUserInfo{
		UserID: user.ID,
		Role:   user.Role,
	})
	if err != nil {
		logrus.Errorf("生成token失败 %s", err)
		response.FailWithMsg("登录失败", c)
		return
	}

	// 更新用户最后登录时间
	now := time.Now().Format(time.DateTime)
	global.DB.Model(&user).Update("last_login_date", now)

	// 登录成功，记录登录日志
	loginLog.SuccessLog(user.ID, cr.Username)
	// 返回登录成功结果（包含Token）
	response.OkWithData(token, c)
}
