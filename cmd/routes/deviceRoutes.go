/**
@Time : 2026/01/20 10:06
@Author: FangYao( 方少、)
@Description:
@Email: fy20030315@163.com
*/

package routes

import (
	"github.com/gin-gonic/gin"
	"go-nvr/cmd/controller"
)

// 注册设备路由
func InitDeviceRoutes(r *gin.RouterGroup) gin.IRoutes {
	base := r.Group("/device")
	{
		base.GET("/ws/detect", controller.Device.DetectionWSHandler)
		base.GET("/snapshot", controller.Device.GetLastSnapshot)
		base.POST("/zonesSave", controller.Device.SaveZones)
		base.GET("/getZones", controller.Device.GetZones)
		base.POST("/hik/info", controller.Device.GetHikDeviceInfo)
		base.POST("/restart", controller.Device.RestartHandler)
		base.GET("/getConfig", controller.Device.GetConfig)    // 获取配置
		base.POST("/saveConfig", controller.Device.SaveConfig) // 保存配置
		base.POST("/alertSnapshot", controller.Device.GetSnapshot)
		base.POST("/ping", controller.Device.Ping)
	}
	return r
}
