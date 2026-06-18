/**
@Time : 2026/03/08 10:05
@Author: FangYao( 方少、)
@Description: 海康设备完整信息响应
@Email: fy20030315@163.com
*/

package response

// HikDeviceFullInfoResp 海康设备完整信息
type HikDeviceFullInfoResp struct {
	IP       string `json:"ip"`       // 设备IP
	Port     int    `json:"port"`     // 端口
	Username string `json:"username"` // 用户名
	// 设备基础信息
	DeviceType    string `json:"deviceType"`    // 设备类型(摄像头/硬盘录像机)
	DeviceName    string `json:"deviceName"`    // 设备名称
	Model         string `json:"model"`         // 设备型号
	MacAddress    string `json:"macAddress"`    // MAC地址
	SerialNumber  string `json:"serialNumber"`  // 设备序列号
	Firmware      string `json:"firmware"`      // 固件版本
	TelecontrolID string `json:"telecontrolId"` // 序列号
	SystemContact string `json:"systemContact"` // 厂商
	// 系统状态
	CPU    string `json:"cpu"`    // CPU占用率(%)
	Memory string `json:"memory"` // 内存占用率(%)
	Uptime int64  `json:"uptime"` // 运行时间(小时)
	// 摄像头专属信息
	ChannelName string `json:"channelName"` // 通道名称
	CodecType   string `json:"codecType"`   // 主码流类型
	BitRate     string `json:"bitRate"`     // 码率(Kbps)
	Resolution  string `json:"resolution"`  // 分辨率(宽x高)
	// 异常信息
	Remark string `json:"remark"` // 备注/错误信息
}
