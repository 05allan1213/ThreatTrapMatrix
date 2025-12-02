package suricata_service

// File: honey_node/service/suricata_service/enter.go
// Description: Suricata告警日志处理模块，负责实时监听Suricata的Eve日志文件，解析告警数据并输出关键信息

import (
	"encoding/json"
	"honey_node/internal/global"
	"honey_node/internal/service/mq_service"
	"io"
	"strconv"
	"time"

	"github.com/hpcloud/tail"
	"github.com/sirupsen/logrus"
)

// AlertType Suricata Eve日志的告警数据结构体，与日志JSON格式一一对应
type AlertType struct {
	Timestamp string `json:"timestamp"`  // 告警发生时间戳
	FlowId    int64  `json:"flow_id"`    // 网络流唯一标识ID
	InIface   string `json:"in_iface"`   // 接收告警流量的网卡接口
	EventType string `json:"event_type"` // 事件类型（仅"alert"类型为告警事件）
	SrcIp     string `json:"src_ip"`     // 源IP地址
	SrcPort   int    `json:"src_port"`   // 源端口
	DestIp    string `json:"dest_ip"`    // 目标IP地址
	DestPort  int    `json:"dest_port"`  // 目标端口
	Proto     string `json:"proto"`      // 网络协议（如TCP、UDP）
	PktSrc    string `json:"pkt_src"`    // 数据包来源
	TxId      int    `json:"tx_id"`      // 事务ID
	Alert     struct { // 告警核心信息
		Action      string `json:"action"`       // 告警响应动作（如allow、block）
		Gid         int    `json:"gid"`          // 规则组ID
		SignatureId int    `json:"signature_id"` // 告警规则ID
		Rev         int    `json:"rev"`          // 规则版本号
		Signature   string `json:"signature"`    // 告警规则描述信息
		Category    string `json:"category"`     // 告警分类（如恶意软件、SQL注入）
		Severity    int    `json:"severity"`     // 告警严重级别（数值越高越严重）
		Metadata    struct { // 告警元数据
			Level []string `json:"level"` // 告警级别描述（如critical、high）
		} `json:"metadata"`
	} `json:"alert"` // 告警核心信息
	Http struct { // HTTP相关告警详情
		Hostname         string `json:"hostname"`           // HTTP请求目标主机名
		Url              string `json:"url"`                // HTTP请求URL
		HttpUserAgent    string `json:"http_user_agent"`    // HTTP请求User-Agent头
		HttpContentType  string `json:"http_content_type"`  // HTTP响应内容类型
		HttpMethod       string `json:"http_method"`        // HTTP请求方法（如GET、POST）
		Protocol         string `json:"protocol"`           // HTTP协议版本（如HTTP/1.1）
		Status           int    `json:"status"`             // HTTP响应状态码
		Length           int    `json:"length"`             // 响应数据长度
		HttpResponseBody string `json:"http_response_body"` // HTTP响应体内容
	} `json:"http"`                     // HTTP相关告警详情（仅HTTP协议流量有值）
	AppProto  string `json:"app_proto"` // 应用层协议（如http、ssh）
	Direction string `json:"direction"` // 流量方向（如toserver、toclient）
	Flow      struct { // 网络流统计信息
		PktsToserver  int    `json:"pkts_toserver"`  // 发送到服务端的数据包数量
		PktsToclient  int    `json:"pkts_toclient"`  // 发送到客户端的数据包数量
		BytesToserver int    `json:"bytes_toserver"` // 发送到服务端的字节数
		BytesToclient int    `json:"bytes_toclient"` // 发送到客户端的字节数
		Start         string `json:"start"`          // 流建立时间
		SrcIp         string `json:"src_ip"`         // 流源IP（同顶层SrcIp）
		DestIp        string `json:"dest_ip"`        // 流目标IP（同顶层DestIp）
		SrcPort       int    `json:"src_port"`       // 流源端口（同顶层SrcPort）
		DestPort      int    `json:"dest_port"`      // 流目标端口（同顶层DestPort）
	} `json:"flow"`                 // 网络流统计信息
	Payload string `json:"payload"` // 告警相关数据包负载内容
	Stream  int    `json:"stream"`  // 流序号
}

// Run 启动Suricata告警日志监听服务，实时解析Eve日志并输出告警关键信息
func Run() {
	cfg := global.Config.System
	// 初始化日志尾追器，从Eve日志文件末尾开始实时监听新日志
	t, err := tail.TailFile(cfg.EvePath, tail.Config{
		Follow: true, // 持续跟随日志文件新增内容
		Location: &tail.SeekInfo{
			Offset: 0,
			Whence: io.SeekEnd, // 初始位置设为文件末尾，避免重复解析历史日志
		},
	})
	if err != nil {
		logrus.Fatalf("suricata路径错误 %s", err)
	}
	logrus.Infof("开始监听suricata告警日志")

	// 循环读取日志行，处理每条新增日志
	for line := range t.Lines {
		var alert AlertType
		// 将日志JSON字符串解析为告警结构体
		err = json.Unmarshal([]byte(line.Text), &alert)
		if err != nil {
			logrus.Errorf("解析suricata告警记录失败 %s %s", err, line.Text)
			continue
		}
		// 仅处理事件类型为"alert"的告警日志，过滤其他类型事件（如stats、flow）
		if alert.EventType != "alert" {
			continue
		}
		// 输出告警关键信息：规则描述、源IP、目标IP:端口
		logrus.Infof("%s %s => %s:%d", alert.Alert.Signature, alert.SrcIp, alert.DestIp, alert.DestPort)

		// 解析告警级别：从metadata.level中提取，处理多级别及转换异常场景
		var level int8
		levelList := alert.Alert.Metadata.Level
		if len(levelList) > 0 {
			// 存在多个级别时记录日志，便于排查数据异常
			if len(levelList) > 1 {
				logrus.Infof("存在level多个的情况 %v", levelList)
			}
			// 将第一个级别字符串转换为整数（业务约定取首个级别）
			l, err := strconv.Atoi(levelList[0])
			if err != nil {
				logrus.Errorf("level转换失败 %s", err)
				level = 1 // 转换失败时设置默认级别为1
			} else {
				level = int8(l) // 转换为int8类型适配消息结构体
			}
		}

		// 时间格式转换：将Suricata日志的UTC时间格式转为业务标准格式
		layout := "2006-01-02T15:04:05.999999Z0700" // Suricata日志原始时间格式（带时区）
		ti, err := time.Parse(layout, alert.Timestamp)
		if err != nil {
			logrus.Errorf("解析时间时出错: %s", err)
			return // 时间解析失败则终止当前告警处理，避免发送无效数据
		}
		timeStamp := ti.Format(time.DateTime) // 转换为业务标准格式：2006-01-02 15:04:05

		// 构造标准告警消息结构体，调用MQ服务发送至告警队列
		mq_service.SendAlertMsg(mq_service.AlertMsgType{
			NodeUid:          cfg.Uid,                     // 节点唯一标识（从配置读取）
			SrcIp:            alert.SrcIp,                 // 攻击源IP（取自Suricata告警数据）
			SrcPort:          alert.SrcPort,               // 攻击源端口（取自Suricata告警数据）
			DestIp:           alert.DestIp,                // 攻击目标IP（取自Suricata告警数据）
			DestPort:         alert.DestPort,              // 攻击目标端口（取自Suricata告警数据）
			Signature:        alert.Alert.Signature,       // 告警规则描述（取自Suricata告警数据）
			HttpResponseBody: alert.Http.HttpResponseBody, // HTTP响应体（仅HTTP类告警有值）
			Payload:          alert.Payload,               // 数据包负载内容（取自Suricata告警数据）
			Level:            level,                       // 告警级别（已处理后的整数级别）
			Timestamp:        timeStamp,                   // 标准化后的告警时间
		})
	}
}
