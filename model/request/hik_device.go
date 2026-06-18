/**
@Time : 2026/03/08 10:00
@Author: FangYao( 方少、)
@Description: 海康设备信息请求参数
@Email: fy20030315@163.com
*/

package request

// HikDeviceInfoReq 海康设备信息获取请求参数
type HikDeviceInfoReq struct {
	DeviceIP string `json:"deviceIp" binding:"required" label:"设备IP"`
	Port     int    `json:"port" binding:"required,min=1,max=65535" label:"设备端口"`
	Username string `json:"username" binding:"required" label:"设备账号"`
	Password string `json:"password" binding:"required" label:"设备密码"`
}
