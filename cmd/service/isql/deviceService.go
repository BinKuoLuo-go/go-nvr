/**
@Time : 2026/01/16 10:47
@Author: FangYao( 方少、)
@Description: 设备相关数据库操作
@Email: fy20030315@163.com
*/

package isql

import (
	"fmt"
	"go-nvr/cmd/service/isql/utils"
	"go-nvr/model"
	"go-nvr/model/request"
	"go-nvr/pkg/common"
	"strings"
	"time"
)

type DeviceServiceIsql struct{}

// SaveZones 保存流的 ROI 数据到数据库
func (d *DeviceServiceIsql) SaveZones(stream string, zones []model.ROIZone) error {
	// 删除原先的 ROI
	if err := common.DB.Where("stream_name = ?", stream).Delete(&model.ROIZone{}).Error; err != nil {
		return err
	}

	// 保存新的 ROI
	for _, z := range zones {
		zone := model.ROIZone{
			StreamName: stream,
			Names:      z.Names,
			Color:      z.Color,
			Points:     z.Points,
			Labels:     z.Labels,
			Signals:    z.Signals,
		}

		// 插入数据
		if err := common.DB.Create(&zone).Error; err != nil {
			return err
		}
	}

	return nil
}

// GetZones 获取指定流的 ROI 数据
func (d *DeviceServiceIsql) GetZones(stream string) (*model.ROIZone, error) {
	var zone model.ROIZone
	err := common.DB.
		Where("stream_name = ?", stream).
		First(&zone).Error

	if err != nil {
		return nil, err
	}
	return &zone, nil
}

// SaveSnapshot 保存快照记录
func (d *DeviceServiceIsql) SaveSnapshot(snapshot *model.Snapshot) error {
	return common.DB.Create(snapshot).Error
}

// GetSnapshot 获取快照记录
func (d *DeviceServiceIsql) GetSnapshot(req *request.SnapshotReq) ([]*model.Snapshot, int64, error) {
	var list []*model.Snapshot
	var total int64
	db := common.DB.Model(&model.Snapshot{}).Order("record_at DESC")

	streamName := strings.TrimSpace(req.StreamName)
	label := strings.TrimSpace(req.Label)
	if streamName != "" {
		// 根据流名称查询
		db = db.Where("stream_name = ?", streamName)
	}
	if label != "" {
		// 根据检测标签查询
		db = db.Where("label = ?", label)
	}
	// 增加时间过滤
	if req.StartMs > 0 {
		startTime := time.UnixMilli(req.StartMs)
		db = db.Where("record_at >= ?", startTime)
	}
	if req.EndMs > 0 {
		endTime := time.UnixMilli(req.EndMs)
		db = db.Where("record_at <= ?", endTime)
	}

	// 统计总数
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("统计快照总数失败: %w", err)
	}

	// 分页查询
	pageReq := utils.NewPageOption(req.PageNum, req.PageSize)
	err := db.Offset(pageReq.PageNum).Limit(pageReq.PageSize).Find(&list).Debug().Error
	return list, total, err
}
