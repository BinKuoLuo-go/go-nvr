/**
@Time : 2026/03/10 14:00
@Author: FangYao( 方少、)
@Description: 快照表
@Email: fy20030315@163.com
*/

package model

import "time"

type Snapshot struct {
	// 主键ID
	ID uint `gorm:"primaryKey;autoIncrement;comment:主键ID" json:"id"`
	// 流名称
	StreamName string `gorm:"size:32;index;comment:流名称" json:"stream_name"`
	// 开始时间
	RecordAt time.Time `gorm:"type:datetime;comment:开始时间" json:"record_at"`
	// 标签类型
	Label string `gorm:"size:64;index;comment:检测标签" json:"label"`
	// 置信度
	Score float64 `gorm:"type:decimal(10,6);comment:置信度得分" json:"score"`
	// 图片路径
	ImagePath string `gorm:"size:255;comment:图片存储路径" json:"image_path"`
	// 模型名称
	Model string `gorm:"size:64;comment:使用模型" json:"model"`
	// 创建/更新时间
	CreatedAt time.Time `gorm:"type:datetime;comment:创建时间" json:"created_at"`
	UpdatedAt time.Time `gorm:"type:datetime;comment:更新时间" json:"updated_at"`
}
