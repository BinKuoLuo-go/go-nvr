/**
@Time : 2026/01/27 15:50
@Author: FangYao( 方少、)
@Description: 区域表
@Email: fy20030315@163.com
*/

package model

import "gorm.io/datatypes"

// ROI 区域模型表
type ROIZone struct {
	ID         uint64         `json:"id"`          // 区域 ID
	StreamName string         `json:"stream_name"` // 视频流名称
	Names      datatypes.JSON `json:"names"`       // 区域名称数组
	Color      datatypes.JSON `json:"color"`       // 区域颜色
	// 区域点位数组
	Points datatypes.JSON `json:"points" ` // 存储 Points 的数组
	// 检测标签
	Labels datatypes.JSON `json:"labels"` // 存储 Labels 的数组
	// 信号标签
	Signals datatypes.JSON `json:"signals"`
}
