package main

import "github.com/gin-gonic/gin"

func main() {
	r := gin.Default()
	r.LoadHTMLFiles("电力平台后台管理系统.html")
	r.Static("static", "电力平台后台管理系统_files")
	r.Static("fonts", "fonts")
	r.GET("", func(c *gin.Context) {
		c.HTML(200, "电力平台后台管理系统.html", nil)
	})

	r.Run(":8081")
}
