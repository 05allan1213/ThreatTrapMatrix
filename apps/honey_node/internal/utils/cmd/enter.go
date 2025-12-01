package cmd

// File: honey_node/utils/cmd/enter.go
// Description: 系统命令执行工具包，提供不同场景下的Shell命令执行功能，支持普通执行、带返回结果执行及指定路径执行

import (
	"bytes"
	"os/exec"

	"github.com/sirupsen/logrus"
)

// Cmd 执行Shell命令，仅返回执行错误（忽略输出结果）
func Cmd(command string) (err error) {
	// 记录要执行的命令（日志追踪）
	logrus.Infof("执行命令 %s", command)
	// 使用sh -c执行命令（支持复杂Shell语法）
	cmd := exec.Command("sh", "-c", command)

	// 捕获命令标准输出（用于日志记录）
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	// 执行命令并返回结果
	err = cmd.Run()
	if err != nil {
		return err
	}

	// 记录命令输出结果
	logrus.Infof("命令输出 %s", stdout.String())
	return nil
}

// Command 执行Shell命令，返回命令输出结果和执行错误
func Command(command string) (msg string, err error) {
	// 记录要执行的命令（日志追踪）
	logrus.Infof("执行命令 %s", command)
	// 使用sh -c执行命令（支持复杂Shell语法）
	cmd := exec.Command("sh", "-c", command)

	// 捕获命令标准输出（用于返回结果）
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	// 执行命令并返回结果
	err = cmd.Run()
	if err != nil {
		return "", err
	}

	// 记录命令输出结果
	logrus.Infof("命令输出 %s", stdout.String())
	return stdout.String(), nil
}

// PathCmd 指定可执行文件路径执行Shell命令，仅返回执行错误
func PathCmd(path string, command string) (err error) {
	// 记录要执行的命令（日志追踪）
	logrus.Infof("执行命令 %s", command)
	// 使用sh -c执行命令（支持复杂Shell语法）
	cmd := exec.Command("sh", "-c", command)

	// 捕获命令标准输出（用于日志记录）
	var stdout bytes.Buffer
	cmd.Stdout = &stdout

	// 设置命令的可执行文件路径（覆盖默认PATH）
	cmd.Path = path

	// 执行命令并返回结果
	err = cmd.Run()
	if err != nil {
		return err
	}

	// 记录命令输出结果
	logrus.Infof("命令输出 %s", stdout.String())
	return nil
}
