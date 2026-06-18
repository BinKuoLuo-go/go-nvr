/*
@Time : 2026/04/21 11:19
@Author: FangYao( 方少、)
@Description: 定时清理图片、录像、数据库过期记录
@Email: fy20030315@163.com
*/
package task

import (
	"go-nvr/model"
	"go-nvr/pkg/common"
	"go-nvr/pkg/config"
	"os"
	"path/filepath"
	"time"

	"github.com/robfig/cron/v3"
)

// InitCronTask 初始化并启动定时清理任务
func InitCronTask() {
	// 初始化cron
	c := cron.New(cron.WithSeconds())
	// 加载配置
	cronExpr := config.Conf.System.Task.CronExpression
	if cronExpr == "" {
		cronExpr = "0 0 0 * * *" // 秒 分 时 日 月 周 默认每天0点执行
	}

	// 添加定时任务
	_, err := c.AddFunc(cronExpr, CleanAllExpiredData)
	if err != nil {
		common.Log.Errorf("定时任务初始化失败: %v", err)
		return
	}

	// 启动定时任务
	c.Start()
	common.Log.Infof("定时清理任务启动成功 | 执行规则: %s | 快照保留%d天 | 录像保留%d天",
		cronExpr,
		config.Conf.System.Task.SnapshotRetentionDays,
		config.Conf.System.Task.RecordRetentionDays)
}

// CleanAllExpiredData 总清理任务：清理文件+数据库
func CleanAllExpiredData() {
	common.Log.Info("===== 开始执行定时清理任务 =====")

	// 清理过期快照文件
	CleanExpiredSnapshot()
	// 清理过期录像文件
	CleanExpiredRecord()
	// 清理数据库过期快照记录
	CleanExpiredDBSnapshot()

	common.Log.Info("===== 定时清理任务执行完成 =====")
}

// CleanExpiredSnapshot 清理过期快照图片
func CleanExpiredSnapshot() {
	rootPath := config.Conf.System.SnapshotRootPath
	retentionDays := config.Conf.System.Task.SnapshotRetentionDays
	if rootPath == "" || retentionDays <= 0 {
		common.Log.Info("快照清理配置无效，跳过")
		return
	}

	// 计算过期时间
	expireTime := time.Now().AddDate(0, 0, -retentionDays)
	common.Log.Infof("开始清理快照: 路径=%s | 过期时间=%s", rootPath, expireTime.Format("2006-01-02 15:04:05"))

	// 递归遍历目录删除过期文件
	err := filepath.Walk(rootPath, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// 跳过目录，只删除文件
		if f.IsDir() {
			return nil
		}
		// 删除修改时间早于过期时间的文件
		if f.ModTime().Before(expireTime) {
			_ = os.Remove(path) // 直接删除
		}
		return nil
	})

	if err != nil {
		common.Log.Errorf("快照清理失败: %v", err)
	} else {
		common.Log.Info("快照清理完成")
	}
}

// CleanExpiredRecord 清理过期告警录像
func CleanExpiredRecord() {
	// 未启用录制则跳过
	if !config.Conf.System.EnableRecording {
		common.Log.Info("录制功能未启用，跳过录像清理")
		return
	}

	rootPath := config.Conf.System.RecordingRootPath
	retentionDays := config.Conf.System.Task.RecordRetentionDays
	if rootPath == "" || retentionDays <= 0 {
		common.Log.Info("录像清理配置无效，跳过")
		return
	}

	expireTime := time.Now().AddDate(0, 0, -retentionDays)
	common.Log.Infof("开始清理录像: 路径=%s | 过期时间=%s", rootPath, expireTime.Format("2006-01-02 15:04:05"))

	err := filepath.Walk(rootPath, func(path string, f os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if f.IsDir() {
			return nil
		}
		if f.ModTime().Before(expireTime) {
			_ = os.Remove(path) // 直接删除
		}
		return nil
	})

	if err != nil {
		common.Log.Errorf("录像清理失败: %v", err)
	} else {
		common.Log.Info("录像清理完成")
	}
}

// CleanExpiredDBSnapshot 清理数据库中过期的快照记录
func CleanExpiredDBSnapshot() {
	retentionDays := config.Conf.System.Task.SnapshotRetentionDays
	if retentionDays <= 0 {
		common.Log.Info("数据库快照清理配置无效，跳过")
		return
	}

	expireTime := time.Now().AddDate(0, 0, -retentionDays)
	common.Log.Infof("开始清理数据库快照记录 | 过期时间=%s", expireTime.Format("2006-01-02 15:04:05"))

	// 删除过期记录
	result := common.DB.Where("record_at < ?", expireTime).Delete(&model.Snapshot{})
	if result.Error != nil {
		common.Log.Errorf("数据库快照清理失败: %v", result.Error)
	} else {
		common.Log.Infof("数据库快照清理完成 | 删除记录数: %d", result.RowsAffected)
	}
}
