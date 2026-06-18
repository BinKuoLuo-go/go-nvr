/**
@Time : 2026/01/19 11:04
@Author: FangYao( 方少、)
@Description: go2rtc api反向代理
@Email: fy20030315@163.com
*/

package controller

import (
	"github.com/gin-gonic/gin"
	"go-nvr/cmd/logic"
)

type ProxyController struct{}

// WebRTCProxy rtsp转webrtc
func (pc *ProxyController) WebRTCProxy(c *gin.Context) {
	logic.Proxy.WebRTCProxy(c)
}

// LogsProxy 日志代理
func (pc *ProxyController) LogsProxy(c *gin.Context) {
	logic.Proxy.LogsProxy(c)
}

// StreamsProxy streams 代理
func (pc *ProxyController) StreamsProxy(c *gin.Context) {
	logic.Proxy.StreamsProxy(c)
}

// InfoProxy go2rtc 基本信息
func (pc *ProxyController) InfoProxy(c *gin.Context) {
	logic.Proxy.InfoProxy(c)
}

// ConfigProxy 配置文件代理
func (pc *ProxyController) ConfigProxy(c *gin.Context) {
	logic.Proxy.ConfigProxy(c)
}

// StreamsDotProxy streams.dot 代理
func (pc *ProxyController) StreamsDotProxy(c *gin.Context) {
	logic.Proxy.StreamsDotProxy(c)
}

// DeviceDiscoveryProxy 统一设备发现代理
func (pc *ProxyController) DeviceDiscoveryProxy(c *gin.Context) {
	logic.Proxy.DeviceDiscoveryProxy(c)
}

// ControlProxy 退出或重启 go2rtc
func (pc *ProxyController) ControlProxy(c *gin.Context) {
	logic.Proxy.ControlProxy(c)
}

// StreamProxy 代理各种流接口
func (pc *ProxyController) StreamProxy(c *gin.Context) {
	logic.Proxy.StreamProxy(c)
}
