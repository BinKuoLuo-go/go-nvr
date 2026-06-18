/**
@Time : 2026/03/13 10:55
@Author: FangYao( 方少、)
@Description:  plc 管理
@Email: fy20030315@163.com
*/

package plc

import (
	"go-nvr/pkg/config"
	"sync"
)

var (
	plcController *PlcController
	plcQueue      chan CameraData
	plcOnce       sync.Once
)

// 初始化PLC
func initPLC() {
	plcController = NewPlcController(config.Conf.Plc)

	plcQueue = make(chan CameraData, 20)

	go plcController.Run(plcQueue)
}

// 程序启动时调用
func StartPLC() {
	plcOnce.Do(initPLC)
}

// 程序关闭时调用
func StopPLC() {
	if plcController != nil {
		plcController.Stop()
	}
}

func GetPLC() (*PlcController, chan CameraData) {
	if !config.Conf.System.IsStartOPlc {
		return nil, nil
	}
	return plcController, plcQueue
}
