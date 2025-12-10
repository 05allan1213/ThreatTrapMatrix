package node_version_api

// File: honey_server/api/node_version_api/node_download.go
// Description: 节点下载脚本生成API接口

import (
	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/models"
	"honey_server/internal/utils/response"
	"net/url"

	"github.com/gin-gonic/gin"
)

// NodeDownloadRequest 节点下载脚本请求结构体
type NodeDownloadRequest struct {
	Version string `form:"version"` // 镜像版本Tag
	Log     bool   `form:"log"`     // 日志输出开关：控制脚本执行时是否打印详细日志
}

// NodeDownloadView 节点下载脚本生成接口
func (NodeVersionApi) NodeDownloadView(c *gin.Context) {
	// 绑定并解析前端提交的下载脚本请求参数（版本Tag+日志开关）
	cr := middleware.GetBind[NodeDownloadRequest](c)

	// 根据版本Tag查询对应的节点镜像记录，校验镜像是否存在
	var model models.NodeVersionModel
	err := global.DB.Take(&model, "tag = ?", cr.Version).Error
	if err != nil {
		response.FailWithMsg("节点镜像不存在", c)
		return
	}

	// 设置脚本下载响应头，保证浏览器识别为文件下载并指定文件名
	c.Header("Content-Type", "application/octet-stream")                                    // 响应类型：二进制流，适配脚本文件下载
	c.Header("Content-Disposition", "attachment; filename="+url.QueryEscape("download.sh")) // 下载文件名：download.sh（转义特殊字符）
	c.Header("Content-Transfer-Encoding", "binary")                                         // 传输编码：二进制，保证脚本内容完整传输

	// 构建节点下载脚本内容
	shellContent := "echo 'hello world'"

	// 返回脚本内容作为响应体
	c.Data(0, "text", []byte(shellContent))
}
