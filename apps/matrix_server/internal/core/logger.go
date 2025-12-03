package core

// File: matrix_server/core/logger.go
// Description: 日志模块，自定义logrus日志格式与钩子，实现日志按日期分割、分级存储及彩色输出

import (
	"matrix_server/internal/global"
	"bytes"
	"fmt"
	"os"
	"path"
	"sync"
	"time"

	"github.com/sirupsen/logrus"
)

// MyLog 自定义日志格式化器，实现logrus.Formatter接口
type MyLog struct{}

// 日志级别对应的终端输出颜色代码
const (
	red    = 31 // 错误级别颜色
	yellow = 33 // 警告级别颜色
	blue   = 36 // 信息级别颜色
	gray   = 37 // 调试/跟踪级别颜色
)

// Format 自定义日志格式化实现，添加颜色、调用信息和时间格式
func (MyLog) Format(entry *logrus.Entry) ([]byte, error) {
	// 根据日志级别设置终端输出颜色
	var levelColor int
	switch entry.Level {
	case logrus.DebugLevel, logrus.TraceLevel:
		levelColor = gray
	case logrus.WarnLevel:
		levelColor = yellow
	case logrus.ErrorLevel, logrus.FatalLevel, logrus.PanicLevel:
		levelColor = red
	default:
		levelColor = blue
	}

	// 初始化缓冲区，优先使用entry自带的Buffer
	var b *bytes.Buffer
	if entry.Buffer != nil {
		b = entry.Buffer
	} else {
		b = &bytes.Buffer{}
	}

	// 格式化日志时间戳
	timestamp := entry.Time.Format("2006-01-02 15:04:05")

	// 若包含调用信息，则拼接完整日志内容
	if entry.HasCaller() {
		// 获取调用函数名和文件位置（仅保留文件名和行号）
		funcVal := entry.Caller.Function
		fileVal := fmt.Sprintf("%s:%d", path.Base(entry.Caller.File), entry.Caller.Line)

		// 自定义日志输出格式：应用名、时间戳、日志级别、调用信息、日志内容
		appName := entry.Data["appName"]
		if appName == nil {
			appName = global.Config.Logger.AppName
		}
		fmt.Fprintf(b, "%s [%s] \x1b[%dm[%s]\x1b[0m %s %s %s\n", appName, timestamp, levelColor, entry.Level, fileVal, funcVal, entry.Message)
	}

	return b.Bytes(), nil
}

// MyHook 自定义日志钩子，实现日志按日期分割、普通日志与错误日志分离存储
type MyHook struct {
	file     *os.File   // 普通日志文件句柄
	errFile  *os.File   // 错误日志文件句柄
	fileDate string     // 当前日志文件对应的日期
	logPath  string     // 日志存储根路径
	mu       sync.Mutex // 确保日志写入线程安全的互斥锁
}

// Fire 日志钩子核心方法，处理日志写入逻辑
func (hook *MyHook) Fire(entry *logrus.Entry) error {
	hook.mu.Lock()
	defer hook.mu.Unlock()

	// 获取当前日志条目对应的日期
	timer := entry.Time.Format("2006-01-02")
	// 格式化日志条目为字符串
	line, err := entry.String()
	if err != nil {
		return fmt.Errorf("failed to format log entry: %v", err)
	}

	// 若日期变化，执行日志文件轮换
	if hook.fileDate != timer {
		if err := hook.rotateFiles(timer); err != nil {
			return err
		}
	}

	// 写入普通日志文件
	if _, err := hook.file.Write([]byte(line)); err != nil {
		return fmt.Errorf("failed to write to log file: %v", err)
	}

	// 错误级别及以上日志同时写入错误日志文件
	if entry.Level <= logrus.ErrorLevel {
		if _, err := hook.errFile.Write([]byte(line)); err != nil {
			return fmt.Errorf("failed to write to error log file: %v", err)
		}
	}

	return nil
}

// rotateFiles 日志文件轮换，按日期创建新的日志文件
func (hook *MyHook) rotateFiles(timer string) error {
	// 关闭旧的日志文件句柄
	if hook.file != nil {
		if err := hook.file.Close(); err != nil {
			return fmt.Errorf("failed to close log file: %v", err)
		}
	}
	if hook.errFile != nil {
		if err := hook.errFile.Close(); err != nil {
			return fmt.Errorf("failed to close error log file: %v", err)
		}
	}

	// 创建当日的日志目录（不存在则创建）
	dirName := fmt.Sprintf("%s/%s", hook.logPath, timer)
	if err := os.MkdirAll(dirName, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create log directory: %v", err)
	}

	// 定义普通日志和错误日志的文件路径
	infoFilename := fmt.Sprintf("%s/info.log", dirName)
	errFilename := fmt.Sprintf("%s/err.log", dirName)

	var err error
	// 打开普通日志文件（追加模式，不存在则创建）
	hook.file, err = os.OpenFile(infoFilename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open log file: %v", err)
	}

	// 打开错误日志文件（追加模式，不存在则创建）
	hook.errFile, err = os.OpenFile(errFilename, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o600)
	if err != nil {
		return fmt.Errorf("failed to open error log file: %v", err)
	}

	// 更新当前日志文件日期
	hook.fileDate = timer
	return nil
}

// Levels 指定钩子处理的日志级别范围（所有级别）
func (hook *MyHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

// GetLogger 初始化日志实例，配置日志级别、格式和钩子
func GetLogger() *logrus.Entry {
	logger := logrus.New()
	l := global.Config.Logger

	// 解析配置的日志级别，无效则默认Info级别
	level, err := logrus.ParseLevel(l.Level)
	if err != nil {
		logrus.Warnf("日志级别配置错误 自动修改为 info")
		level = logrus.InfoLevel
	}

	// 设置日志级别
	logger.SetLevel(level)
	// 添加自定义日志钩子（处理文件分割）
	logger.AddHook(&MyHook{logPath: "logs"})

	// 根据配置选择日志格式（JSON或自定义彩色格式）
	if l.Format == "json" {
		logger.SetFormatter(&logrus.JSONFormatter{
			TimestampFormat: time.DateTime,
		})
	} else {
		logger.SetFormatter(&MyLog{})
	}

	// 启用调用信息追踪
	logger.SetReportCaller(true)
	// 添加应用名字段并返回日志实例
	return logger.WithField("appName", l.AppName)
}

// SetLogDefault 设置默认日志配置
func SetLogDefault() {
	l := global.Config.Logger
	logrus.SetFormatter(&MyLog{})
	logrus.SetReportCaller(true)
	logrus.WithField("appName", l.AppName)
}
