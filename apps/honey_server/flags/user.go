package flags

// File: honey_server/flags/user.go
// Description: 用户命令行操作模块，支持通过JSON参数或交互式方式创建用户及查询用户列表

import (
	"encoding/json"
	"fmt"
	"os"

	"ThreatTrapMatrix/apps/honey_server/global"
	"ThreatTrapMatrix/apps/honey_server/models"
	"ThreatTrapMatrix/apps/honey_server/utils/pwd"

	"github.com/sirupsen/logrus"
	"golang.org/x/crypto/ssh/terminal"
)

// User 命令行用户操作处理器结构体，封装用户创建与列表查询功能
type User struct{}

// UserInfoRequest 用户信息详情请求结构体，用于接收创建用户的参数
type UserInfoRequest struct {
	Role     int8   `json:"role"`     // 用户角色（1-管理员，2-普通用户）
	Username string `json:"username"` // 用户名
	Password string `json:"password"` // 用户密码
}

// Create 创建用户，支持JSON参数传入或交互式输入两种方式
func (User) Create(value string) {
	var userInfo UserInfoRequest

	// 判断是否通过JSON参数传入用户信息
	if value != "" {
		// 解析JSON参数到用户信息结构体
		err := json.Unmarshal([]byte(value), &userInfo)
		if err != nil {
			logrus.Errorf("用户信息错误 %s", err)
			return
		}
	} else {
		// 交互式输入用户信息流程
		// 选择用户角色
		fmt.Println("请选择角色： 1 管理员 2 普通用户")
		_, err := fmt.Scanln(&userInfo.Role)
		if err != nil {
			fmt.Println("输入错误", err)
			return
		}
		// 校验角色输入合法性
		if !(userInfo.Role == 1 || userInfo.Role == 2) {
			fmt.Println("用户角色输入错误")
			return
		}

		// 输入用户名
		fmt.Println("请输入用户名")
		fmt.Scanln(&userInfo.Username)

		// 输入密码（隐藏输入）
		fmt.Println("请输入密码")
		password, err := terminal.ReadPassword(int(os.Stdin.Fd())) // 安全读取密码（不回显）
		if err != nil {
			fmt.Println("读取密码时出错:", err)
			return
		}

		// 确认密码
		fmt.Println("请再次输入密码")
		rePassword, err := terminal.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			fmt.Println("读取密码时出错:", err)
			return
		}

		// 校验两次密码一致性
		if string(password) != string(rePassword) {
			fmt.Println("两次密码不一致")
			return
		}
		userInfo.Password = string(password)
	}

	// 检查用户名是否已存在
	var u models.UserModel
	err := global.DB.Take(&u, "username = ?", userInfo.Username).Error
	if err == nil {
		fmt.Println("用户名已存在")
		return
	}

	// 密码加密并创建用户记录
	hashPwd, _ := pwd.GenerateFromPassword(userInfo.Password)
	err = global.DB.Create(&models.UserModel{
		Username: userInfo.Username,
		Password: hashPwd,
		Role:     userInfo.Role,
	}).Error
	if err != nil {
		logrus.Errorf("用户创建失败 %s", err)
		return
	}
	logrus.Infof("用户创建成功")
}

// List 查询并展示最近创建的10条用户信息列表
func (User) List() {
	var userList []models.UserModel
	// 查询最近创建的10条用户记录（按创建时间倒序）
	global.DB.Order("created_at desc").Limit(10).Find(&userList)

	// 格式化输出用户信息
	for _, model := range userList {
		fmt.Printf("用户id：%d  用户名：%s 用户角色：%d 创建时间：%s\n",
			model.ID,
			model.Username,
			model.Role,
			model.CreatedAt.Format("2006-01-02 15:04:05"),
		)
	}
}
