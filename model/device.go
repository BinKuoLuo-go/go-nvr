/**
@Time : 2026/01/16 09:43
@Author: FangYao( 方少、)
@Description:
@Email: fy20030315@163.com
*/

package model

import (
	"gorm.io/datatypes"
	"time"
)

// Device 系统设备表
type Device struct {
	ID           string         `gorm:"primaryKey;type:varchar(64)" json:"id"` // 主键 ID
	Type         string         `gorm:"type:varchar(32)" json:"type"`          // 设备类型
	DeviceID     string         `gorm:"type:varchar(64)" json:"device_id"`     // 设备 ID
	Name         string         `gorm:"type:varchar(128)" json:"name"`         // 设备名称
	IP           string         `gorm:"type:varchar(64)" json:"ip"`            // 设备 IP
	Port         int            `gorm:"type:int" json:"port"`                  // 设备端口
	IsOnline     bool           `gorm:"type:boolean" json:"is_online"`         // 在线状态
	RegisteredAt time.Time      `gorm:"type:datetime" json:"registered_at"`    // 注册时间
	Username     string         `gorm:"type:varchar(64)" json:"username"`      // 登录用户名
	Password     string         `gorm:"type:varchar(64)" json:"password"`      // 登录密码
	IPAddress    string         `gorm:"type:varchar(128)" json:"ip_address"`   // IP地址 IP:Port
	Ext          datatypes.JSON `gorm:"type:json" json:"ext"`                  // 设备详细信息
	CreatedAt    time.Time      `json:"created_at"`                            // 创建时间
	UpdatedAt    time.Time      `json:"updated_at"`                            // 更新时间
}

// DeviceExt 设备详细信息
type DeviceExt struct {
	StreamName string     `json:"streamName"`       // 流名称
	Model      string     `json:"model"`            // 模型
	Zones      []ZoneItem `json:"zones,omitempty" ` // 区域信息
	EnabledAI  bool       `json:"enabled_ai"`       // 是否启用推理
}

// ZoneItem 坐标/标签/颜色字段
type ZoneItem struct {
	Name        string    `json:"name"`        // 区域名称
	Coordinates []float64 `json:"coordinates"` // 区域坐标点(浮点数组)
	Color       string    `json:"color"`       // 区域展示颜色
	Labels      []string  `json:"labels"`      // 识别标签列表
}

// Event
type Event struct {
	ID         int64     `gorm:"primaryKey" json:"id"`                                                                        // ID
	StreamName string    `gorm:"column:stream_name;notNull;default:'';comment:流名称（" json:"stream_name"`                       // ：流名称（标识对应的视频流/数据流）
	Label      string    `gorm:"column:label;notNull;default:'';comment:检测标签 (person, car 等)" json:"label"`                   // 检测标签 (person, car 等)
	Score      float32   `gorm:"column:score;notNull;default:0;comment:置信度 (0.0-1.0)" json:"score"`                           // 置信度 (0.0-1.0)
	ImagePath  string    `gorm:"column:image_path;notNull;default:'';comment:图片相对路径 (cid/年月日时分秒_随机6位.jpg)" json:"image_path"` // 图片相对路径 (cid/年月日时分秒_随机6位.jpg)
	Model      string    `gorm:"column:model;notNull;default:'';comment:分析模型名称" json:"model"`                                 // 分析模型名称
	CreatedAt  time.Time `gorm:"column:created_at;notNull;default:CURRENT_TIMESTAMP;comment:创建时间" json:"created_at"`          // 创建时间
	UpdatedAt  time.Time `gorm:"column:updated_at;notNull;default:CURRENT_TIMESTAMP;comment:更新时间" json:"updated_at"`          // 更新时间
}
