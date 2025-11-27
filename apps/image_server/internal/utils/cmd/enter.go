package cmd

// File: image_server/utils/cmd/enter.go
// Description: 提供系统命令执行工具方法，支持普通命令执行及指定工作路径的命令执行

import (
	"bytes"
	"os/exec"

	"github.com/sirupsen/logrus"
)

// Cmd 执行系统命令（默认工作路径）
func Cmd(command string) (err error) {
	// 记录待执行的命令日志
	logrus.Infof("执行命令 %s", command)
	// 创建shell命令执行实例（通过sh -c执行命令）
	cmd := exec.Command("sh", "-c", command)
	// 捕获命令标准输出
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	// 执行命令
	err = cmd.Run()
	if err != nil {
		return err
	}
	// 记录命令执行输出日志
	logrus.Infof("命令输出 %s", stdout.String())
	return nil
}

// PathCmd 指定工作路径执行系统命令
func PathCmd(path string, command string) (err error) {
	// 记录待执行的命令日志
	logrus.Infof("执行命令 %s", command)
	// 创建shell命令执行实例（通过sh -c执行命令）
	cmd := exec.Command("sh", "-c", command)
	// 捕获命令标准输出
	var stdout bytes.Buffer
	cmd.Stdout = &stdout
	// 设置命令执行的工作路径
	cmd.Path = path
	// 执行命令
	err = cmd.Run()
	if err != nil {
		return err
	}
	// 记录命令执行输出日志
	logrus.Infof("命令输出 %s", stdout.String())
	return nil
}
