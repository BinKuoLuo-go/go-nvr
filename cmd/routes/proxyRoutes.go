/**
@Time : 2026/01/19 11:10
@Author: FangYao( 方少、)
@Description: go2rtc代理路由
@Email: fy20030315@163.com
*/

package routes

import (
	"github.com/gin-gonic/gin"
	"go-nvr/cmd/controller"
)

// 注册代理路由
func InitProxyRoutes(r *gin.RouterGroup) gin.IRoutes {
	base := r.Group("/proxy")
	{
		base.GET("/info", controller.Proxy.InfoProxy)
		base.GET("/streams.dot", controller.Proxy.StreamsDotProxy)
		base.Any("/webrtc", controller.Proxy.WebRTCProxy)
		base.Any("/log", controller.Proxy.LogsProxy)
		base.Any("/streams", controller.Proxy.StreamsProxy)
		base.Any("/config", controller.Proxy.ConfigProxy)
		base.Any("/device", controller.Proxy.DeviceDiscoveryProxy)
		base.Any("/control", controller.Proxy.ControlProxy)

		// 流代理
		base.GET("/stream.mp4", controller.Proxy.StreamProxy)
		base.GET("/stream.m3u8", controller.Proxy.StreamProxy)
		base.GET("/hls/playlist.m3u8", controller.Proxy.StreamProxy)
		base.GET("/hls/segment.ts", controller.Proxy.StreamProxy)
		base.GET("/hls/init.mp4", controller.Proxy.StreamProxy)
		base.GET("/hls/segment.m4s", controller.Proxy.StreamProxy)
		base.GET("/stream.mjpeg", controller.Proxy.StreamProxy)
		base.GET("/stream.ascii", controller.Proxy.StreamProxy)
		base.GET("/stream.y4m", controller.Proxy.StreamProxy)
		base.GET("/stream.ts", controller.Proxy.StreamProxy)
		base.GET("/stream.aac", controller.Proxy.StreamProxy)
		base.GET("/stream.flv", controller.Proxy.StreamProxy)
		base.GET("/frame.jpeg", controller.Proxy.StreamProxy)
		base.GET("/frame.mp4", controller.Proxy.StreamProxy)

	}
	return r
}
