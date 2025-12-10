package site_api

// File: honey_server/api/site_api/enter.go
// Description: 站点配置模块API接口

import (
	"errors"
	"fmt"
	"honey_server/internal/config"
	"honey_server/internal/core"
	"honey_server/internal/global"
	"honey_server/internal/middleware"
	"honey_server/internal/utils/response"
	"os"

	"github.com/PuerkitoBio/goquery"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
)

// SiteApi 站点配置模块API接口结构体，封装站点信息查询、配置更新相关接口方法
type SiteApi struct {
}

// SiteInfoView 站点信息查询接口处理函数
func (SiteApi) SiteInfoView(c *gin.Context) {
	// 返回站点配置信息，data为全局配置中的站点配置结构体
	response.OkWithData(global.Config.Site, c)
}

// SiteUpdateView 站点配置更新接口处理函数
func (SiteApi) SiteUpdateView(c *gin.Context) {
	// 绑定并解析前端提交的站点配置参数
	cr := middleware.GetBind[config.Site](c)

	// 保留原HTML文件路径，避免配置更新时丢失路径信息
	path := global.Config.Site.Path
	global.Config.Site = cr
	global.Config.Site.Path = path

	// 根据新配置修改HTML文件的标题和图标
	err := SetHtml(global.Config.Site)
	if err != nil {
		logrus.Errorf("修改站点html文件失败 %s", err)
		response.FailWithMsg("修改站点html文件失败", c)
		return
	}

	// 将更新后的配置持久化到配置文件
	err = core.SetConfig(global.Config)
	if err != nil {
		response.FailWithMsg("保存站点配置文件失败", c)
		return
	}

	// 配置更新成功，返回提示信息
	response.OkWithMsg("站点配置修改成功", c)
}

// SetHtml 根据站点配置修改指定HTML文件的标题和图标
func SetHtml(p config.Site) error {
	// 若HTML文件路径为空，无需处理直接返回
	if p.Path == "" {
		return nil
	}

	// 打开指定路径的HTML文件
	f, err := os.Open(p.Path)
	if err != nil {
		return errors.New("文件不存在")
	}
	defer f.Close() // 延迟关闭文件句柄，避免资源泄漏

	// 使用goquery解析HTML文件内容
	doc, _ := goquery.NewDocumentFromReader(f)

	// 处理站点标题：存在title标签则修改文本，不存在则在head标签下新增
	if p.Title != "" {
		if doc.Find("title").Length() != 0 {
			doc.Find("title").SetText(p.Title)
		} else {
			doc.Find("head").AppendHtml(fmt.Sprintf("<title>%s</title>", p.Title))
		}
	}

	// 处理站点图标：存在link[rel="icon"]标签则修改href属性，不存在则在head标签下新增
	if p.Icon != "" {
		if doc.Find("link[rel=\"icon\"]").Length() != 0 {
			doc.Find("link[rel=\"icon\"]").SetAttr("href", p.Icon)
		} else {
			doc.Find("head").AppendHtml(fmt.Sprintf("<link rel=\"icon\" href=\"%s\">", p.Icon))
		}
	}

	// 将修改后的HTML内容转为字符串
	ret, _ := doc.Html()
	// 写入修改后的内容到原文件，权限设置为0666
	err = os.WriteFile(p.Path, []byte(ret), 0666)
	return err
}
