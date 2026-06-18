/**
@Time : 2026/01/27 15:50
@Author: FangYao( 方少、)
@Description:
@Email: fy20030315@163.com
*/

package request

type FrontendROIReq struct {
	Stream string            `json:"stream"` // stream字段
	Zones  []FrontendROIZone `json:"zones"`  // zones数组
}

// 区域信息结构体
type FrontendROIZone struct {
	Name    string    `json:"name"`    // 区域名称
	Points  []float32 `json:"points"`  // 坐标点数组，float32适配前端的小数数值
	Color   string    `json:"color"`   // 颜色值
	Labels  []string  `json:"labels"`  // 检测标签数组
	Signals []string  `json:"signals"` // 信号标签数组
}

// GetROIReq 前端根据流名请求载荷
type GetROIReq struct {
	StreamName string `json:"stream_name" form:"stream_name"`
}
