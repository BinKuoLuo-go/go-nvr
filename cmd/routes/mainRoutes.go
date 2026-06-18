/*
*
@Time : 2025/12/14 18:40
@Author: FangYao( 方少、)
@Description:  主路由层
@Email: fy20030315@163.com
*/
package routes

import (
	_ "fmt"
	"github.com/gin-gonic/gin"
	"go-nvr/pkg/common"
	"go-nvr/pkg/config"
	"go-nvr/pkg/middleware"
	"go-nvr/static"
	"net/http"
	_ "path/filepath"
	_ "time"
)

func InitRoutes() *gin.Engine {
	// 设置模式
	gin.SetMode(config.Conf.System.Mode)

	// 创建带有默认中间件的路由:
	r := gin.Default()

	// 创建不带中间件的路由:
	// r := gin.New()
	// r.Use(gin.Recovery())
	r.Static("/snapshots", config.Conf.System.SnapshotRootPath)

	// 注册静态文件服务中间件
	r.Use(middleware.Serve("/", middleware.EmbedFolder(static.Static, "dist")))
	r.NoRoute(func(c *gin.Context) {
		// 从嵌入的文件系统中读取dist目录下的index.html
		data, err := static.Static.ReadFile("dist/index.html")
		if err != nil {
			// 若读取失败（如文件不存在），返回500错误
			_ = c.AbortWithError(http.StatusInternalServerError, err)
			return
		}
		// 成功读取则返回index.html内容，类型为text/html
		c.Data(http.StatusOK, "text/html; charset=utf-8", data)
	})

	// 启用全局跨域中间件
	r.Use(middleware.CORSMiddleware())

	apiGroup := r.Group("/" + config.Conf.System.UrlPathPrefix)
	// 注册代理路由
	InitProxyRoutes(apiGroup)
	InitDeviceRoutes(apiGroup)
	common.Log.Info("初始化路由完成！")
	return r
}
