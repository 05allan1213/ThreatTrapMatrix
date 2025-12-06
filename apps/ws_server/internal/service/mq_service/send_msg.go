package mq_service

// wsMsgType WebSocket消息类型
type wsMsgType struct {
	Type   int8 `json:"type"`   // 消息类型
	NetID  uint `json:"netID"`  // 子网ID
	NodeID uint `json:"nodeID"` // 节点ID
}

/*
1 部署  前端调可用ip列表接口 参数是 NetID
2 进度  参数是 NetID
3 告警  前端根据自己所在的页面，去请求对应的接口
4 节点
5 节点详情 参数是 NodeID
6 网络列表
6 主机列表
7 用户
8 镜像列表
9 虚拟服务列表
10 主机模板
11 矩阵模板
*/
