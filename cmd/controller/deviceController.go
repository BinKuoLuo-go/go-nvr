/**
@Time : 2026/01/16 09:26
@Author: FangYao( 方少、)
@Description: 控制层
@Email: fy20030315@163.com
*/

package controller

import (
	"github.com/gin-gonic/gin"
	"go-nvr/cmd/logic"
	"go-nvr/model/request"
)

type DeviceController struct{}

// DetectionWSHandler WebSocket 推理接口
func (dc *DeviceController) DetectionWSHandler(c *gin.Context) {
	logic.Device.DetectionWSHandler(c)
}

// GetLastSnapshot ws获取快照
func (dc *DeviceController) GetLastSnapshot(c *gin.Context) {
	logic.Device.GetLastSnapshot(c)
}

// SaveZones 保存zones
func (dc *DeviceController) SaveZones(c *gin.Context) {
	req := new(request.FrontendROIReq)
	Run(c, req, func() (interface{}, interface{}) {
		return logic.Device.SaveZones(c, req)
	})
}

// GetZones 根据流名获取zones
func (dc *DeviceController) GetZones(c *gin.Context) {
	req := new(request.GetROIReq)
	Run(c, req, func() (interface{}, interface{}) {
		return logic.Device.GetZones(c, req)
	})
}

// GetHikDeviceInfo 获取海康设备信息
func (dc *DeviceController) GetHikDeviceInfo(c *gin.Context) {
	req := new(request.HikDeviceInfoReq)
	Run(c, req, func() (interface{}, interface{}) {
		return logic.Device.GetHikDeviceInfo(c, req)
	})
}

// RestartHandler 服务自我重启接口
func (dc *DeviceController) RestartHandler(c *gin.Context) {
	// 调用业务逻辑层
	logic.Device.RestartHandler(c)
}

// GetConfig 获取根目录下的 YAML 配置
func (cc *DeviceController) GetConfig(c *gin.Context) {
	logic.Device.ReadConfigHandler(c)
}

// SaveConfig 保存 YAML 配置
func (cc *DeviceController) SaveConfig(c *gin.Context) {
	logic.Device.SaveConfigHandler(c)
}

// GetSnapshot 告警界面获取快照数据
func (dc *DeviceController) GetSnapshot(c *gin.Context) {
	req := new(request.SnapshotReq)
	Run(c, req, func() (interface{}, interface{}) {
		return logic.Device.GetSnapshot(c, req)
	})
}

// 心跳
func (dc *DeviceController) Ping(c *gin.Context) {
	// 调用业务逻辑层
	logic.Device.Ping(c)
}
