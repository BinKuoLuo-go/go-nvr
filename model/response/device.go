/**
@Time : 2026/01/16 09:36
@Author: FangYao( 方少、)
@Description:
@Email: fy20030315@163.com
*/

package response

// 单个 ROI 区域
type FrontendROIZone struct {
	Name    string    `json:"name"`
	Points  []float32 `json:"points"`
	Color   string    `json:"color"`
	Labels  []string  `json:"labels"`
	Signals []string  `json:"signals"`
}

// ROI 返回结构
type FrontendROIResp struct {
	Stream string            `json:"stream"`
	Zones  []FrontendROIZone `json:"zones"`
}
