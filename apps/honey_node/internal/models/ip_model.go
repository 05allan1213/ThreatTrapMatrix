package models

// IpModel 诱捕ip模型
type IpModel struct {
	Model
	Ip       string `json:"ip"`       // 诱捕ip
	Mask     int8   `json:"mask"`     // 子网掩码
	LinkName string `json:"linkName"` // 自己的接口名称
	Network  string `json:"network"`  // 基于哪个网卡创建的
	Mac      string `json:"mac"`      // mac地址
}
