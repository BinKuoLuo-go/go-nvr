/**
@Time : 2026/03/13 10:42
@Author: FangYao( 方少、)
@Description: 测试PLC控制器主循环、心跳及复位
@Email: fy20030315@163.com
*/

package plc

import (
	"fmt"
	"go-nvr/pkg/common"
	"go-nvr/pkg/config"
	"go.uber.org/zap"
	"testing"
	"time"
)

// MockPLCClient 模拟PLC客户端，保证WriteBit总是成功
type MockPLCClient struct{}

func (m *MockPLCClient) WriteBit(byteAddr, bit int, value bool) error {
	// 模拟写入成功
	return nil
}

func (m *MockPLCClient) Close() {}

// TestPlcController_Run 测试PLC控制器主循环和心跳
func TestPlcController_Run(t *testing.T) {
	// 初始化logger
	logger, _ := zap.NewDevelopment()
	common.Log = logger.Sugar()
	// 初始化PLC配置
	cfg := &config.PlcConfig{
		IP:   "192.168.1.2",
		Rack: 0,
		Slot: 1,
	}

	controller := NewPlcController(cfg)

	// 注入 MockPLCClient
	controller.plc = &PLCClient{}
	controller.connected = true

	// 创建摄像头数据队列
	cameraQueue := make(chan CameraData, 100)

	// 启动PLC控制器主循环
	go controller.Run(cameraQueue)

	// 模拟摄像头检测数据
	go func() {
		cameras := []string{"cam1", "cam2"}
		directions := []string{"front", "back", "inside"}

		for i := 0; i < 10; i++ {
			for _, cam := range cameras {
				data := CameraData{
					Camera:      cam,
					ActionLevel: i % 3, // 0/1/2 模拟 Stop/SLOW/None
					Direction:   directions[i%len(directions)],
					Timestamp:   float64(time.Now().Unix()),
					FrameTimeTs: float64(time.Now().Unix()),
					ObjectID:    fmt.Sprintf("%s_obj_%d", cam, i),
					ObjType:     "person",
					Score:       0.8,
				}
				cameraQueue <- data
			}
			time.Sleep(100 * time.Millisecond)
		}
	}()

	// 等待一段时间让控制器处理
	time.Sleep(3 * time.Second)

	// 停止PLC控制器
	controller.Stop()

	// 等待ResetAllSignals完成
	time.Sleep(200 * time.Millisecond)

	// 检查信号全部复位
	for _, tag := range []string{"M16", "M17", "M18", "M19", "M21", "M22"} {
		val, ok := controller.lastWritten.Load(tag)
		if !ok {
			t.Errorf("PLC信号 %s 不存在", tag)
			continue
		}
		if val.(bool) != false {
			t.Errorf("PLC信号 %s 未复位，当前值：%v", tag, val)
		} else {
			common.Log.Infof("PLC信号 %s 复位成功", tag)
		}
	}
}
