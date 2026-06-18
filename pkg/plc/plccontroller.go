/**
@Time : 2026/03/13 09:02
@Author: FangYao( 方少、)
@Description: plc控制器
@Email: fy20030315@163.com
*/

package plc

import (
	"go-nvr/pkg/common"
	"go-nvr/pkg/config"
	"sync"
	"time"
)

const (
	ForwardStop = "forward_stop" // 前向停止信号
	ForwardSlow = "forward_slow" // 前向慢速信号
	BackStop    = "back_stop"    // 后向停止信号
	BackSlow    = "back_slow"    // 后向慢速信号
)

type CameraData struct {
	Camera       string   // 摄像头唯一标识ID/名称
	ActionLevel  int      // 动作等级：0-无动作 1-慢速 2-停止
	Direction    string   // 目标方向：front-前向 back-后向 inside-双向
	Timestamp    float64  // 检测触发时间戳(秒)
	FrameTimeTs  float64  // 视频帧时间戳
	ObjectID     string   // 检测目标唯一ID
	ObjType      string   // 检测目标类型
	CurrentZones []string // 目标所在区域列表
	EventType    string   // 事件类型
	EndTime      float64  // 事件结束时间戳
	IsTrueTarget bool     // 是否为有效真实目标
	Score        float64  // 目标检测置信度
}

type PlcController struct {
	plc          *PLCClient
	plcConfig    *config.PlcConfig
	plcLock      sync.Mutex
	connected    bool
	lastWritten  *sync.Map // 线程安全 map
	HoldTime     float64
	actionHolds  map[string]*sync.Map
	stopChan     chan struct{}
	heartbeatDur time.Duration
}

func NewPlcController(cfg *config.PlcConfig) *PlcController {
	// 初始化 actionHolds
	ah := map[string]*sync.Map{
		ForwardStop: new(sync.Map),
		ForwardSlow: new(sync.Map),
		BackStop:    new(sync.Map),
		BackSlow:    new(sync.Map),
	}

	// 初始化 lastWritten
	lw := new(sync.Map)
	for _, tag := range []string{"M16", "M17", "M18", "M19", "M20", "M21", "M22"} {
		lw.Store(tag, false)
	}

	return &PlcController{
		plcConfig:    cfg,
		lastWritten:  lw,
		HoldTime:     5.0,
		actionHolds:  ah,
		stopChan:     make(chan struct{}),
		heartbeatDur: 5 * time.Second,
	}
}

// ConnectPLC 尝试连接PLC
func (c *PlcController) ConnectPLC() bool {
	c.plcLock.Lock()
	defer c.plcLock.Unlock()
	client, err := NewPLCS7Client(c.plcConfig)
	if err != nil {
		c.connected = false
		return false
	}
	c.plc = client
	c.connected = true
	return true
}

// WriteBit 安全写入M区点位
func (c *PlcController) WriteBit(byteAddr int, bit int, value bool, tag string) bool {
	if !c.connected || c.plc == nil {
		common.Log.Infof("[PLC] %s 写入失败: 未连接", tag)
		return false
	}
	c.plcLock.Lock()
	defer c.plcLock.Unlock()

	// 读取 lastWritten
	lastVal, _ := c.lastWritten.Load(tag)
	if lastVal.(bool) != value {
		if tag != "M20" {
			common.Log.Infof("[PLC] 准备写入%s: 目标值=%v", tag, value)
		}
		err := c.plc.WriteBit(byteAddr, bit, value)
		if err != nil {
			c.connected = false
			common.Log.Errorf("[PLC] %s 写入失败: %v", tag, err)
			return false
		}
		c.lastWritten.Store(tag, value)
		if tag != "M20" {
			common.Log.Infof("[PLC] %s 写入成功: 当前值=%v", tag, value)
		}
	}
	return true
}

// ResetAllSignals 将M16-M22全部复位
func (c *PlcController) ResetAllSignals() {
	common.Log.Infof("[PLC] 开始复位所有PLC信号(M16-M22)")
	for _, p := range []struct {
		byteAddr int
		tag      string
	}{
		{16, "M16"}, {17, "M17"}, {18, "M18"}, {19, "M19"}, {21, "M21"}, {22, "M22"},
	} {
		c.WriteBit(p.byteAddr, 0, false, p.tag)
		time.Sleep(20 * time.Millisecond)
	}
	common.Log.Infof("[PLC] 所有PLC信号复位完成")
}

// StartupSequence 闪烁M21三次
func (c *PlcController) StartupSequence() {
	common.Log.Infof("[PLC] 开始执行启动闪烁序列(M21)")
	c.ResetAllSignals()
	for i := 0; i < 3; i++ {
		c.WriteBit(21, 0, true, "M21")
		time.Sleep(500 * time.Millisecond)
		c.WriteBit(21, 0, false, "M21")
		time.Sleep(500 * time.Millisecond)
	}
	common.Log.Infof("[PLC] PLC启动序列执行完成")
}

// HeartbeatLoop 心跳线程，M20.0
func (c *PlcController) HeartbeatLoop() {
	ticker := time.NewTicker(c.heartbeatDur)
	defer ticker.Stop()
	for {
		select {
		case <-c.stopChan:
			return
		default:
			if c.connected {
				c.WriteBit(20, 0, true, "M20")
				time.Sleep(500 * time.Millisecond)
				c.WriteBit(20, 0, false, "M20")
			}
			time.Sleep(4 * time.Second)
		}
	}
}

// ResetExpiredSignals 清理过期信号并写入PLC
func (c *PlcController) ResetExpiredSignals() {
	now := float64(time.Now().UnixNano()) / 1e9

	// 清理过期信号
	for action, cameras := range c.actionHolds {
		cameras.Range(func(camKey, tsValue interface{}) bool {
			cam := camKey.(string)
			ts := tsValue.(float64)
			if now-ts > c.HoldTime {
				cameras.Delete(cam)
				common.Log.Infof("[PLC] 信号过期清理并自动复位: 动作=%s, 摄像头=%s", action, cam)
			}
			return true
		})
	}

	// 聚合信号
	m16, m17, m18, m19 := false, false, false, false
	c.actionHolds[ForwardStop].Range(func(_, _ interface{}) bool { m16 = true; return false })
	c.actionHolds[BackStop].Range(func(_, _ interface{}) bool { m17 = true; return false })
	c.actionHolds[ForwardSlow].Range(func(_, _ interface{}) bool { m18 = true; return false })
	c.actionHolds[BackSlow].Range(func(_, _ interface{}) bool { m19 = true; return false })

	if m16 {
		m18 = false
	}
	if m17 {
		m19 = false
	}

	m21 := m16 || m17 || m18 || m19
	m22 := m16 || m17

	// 判断是否有变化
	lastM16, _ := c.lastWritten.Load("M16")
	lastM17, _ := c.lastWritten.Load("M17")
	lastM18, _ := c.lastWritten.Load("M18")
	lastM19, _ := c.lastWritten.Load("M19")
	lastM21, _ := c.lastWritten.Load("M21")
	lastM22, _ := c.lastWritten.Load("M22")

	changed := lastM16.(bool) != m16 ||
		lastM17.(bool) != m17 ||
		lastM18.(bool) != m18 ||
		lastM19.(bool) != m19 ||
		lastM21.(bool) != m21 ||
		lastM22.(bool) != m22

	if !changed {
		return
	}

	if c.connected {

		common.Log.Infof("[PLC] 信号状态已变更，开始写入PLC")
		//M16：前进停止
		//M17: 后退停止
		//M18: 前进减速
		//M19: 后退减速
		//M20：心跳
		//M21: 光学报警
		//M22: 声学报警
		c.WriteBit(16, 0, m16, "M16")
		c.WriteBit(17, 0, m17, "M17")
		c.WriteBit(18, 0, m18, "M18")
		c.WriteBit(19, 0, m19, "M19")
		c.WriteBit(21, 0, m21, "M21")
		c.WriteBit(22, 0, m22, "M22")

		common.Log.Infof("[PLC] 状态更新完成 M16=%v M17=%v M18=%v M19=%v M21=%v M22=%v",
			m16, m17, m18, m19, m21, m22)
	} else {
		common.Log.Infof("[PLC] PLC未连接，状态未写入")
	}
}

// Run 循环调用 ResetExpiredSignals
func (c *PlcController) Run(queueChan <-chan CameraData) {
	if c.ConnectPLC() {
		c.StartupSequence()
	} else {
		common.Log.Errorf("PLC 初次连接失败，进入重连循环")
	}

	go c.HeartbeatLoop()

	ticker := time.NewTicker(50 * time.Millisecond)
	defer ticker.Stop()
	lastReconnect := time.Now()

	for {
		select {
		case <-c.stopChan:
			c.ResetAllSignals()
			if c.plc != nil {
				c.plc.Close()
			}
			return

		case data := <-queueChan:
			common.Log.Infof("[PLC] 接收到数据: 摄像头=[%s], 方向=%s, 动作等级=%d",
				data.Camera, data.Direction, data.ActionLevel)
			var targets []string
			switch data.Direction {
			case "front":
				if data.ActionLevel == 2 {
					targets = []string{ForwardStop}
					common.Log.Infof("[PLC] 触发前向停止信号 | 摄像头:%s", data.Camera)
				} else if data.ActionLevel == 1 {
					targets = []string{ForwardSlow}
					common.Log.Infof("[PLC] 触发前向减速信号 | 摄像头:%s", data.Camera)
				}
			case "back":
				if data.ActionLevel == 2 {
					targets = []string{BackStop}
					common.Log.Infof("[PLC] 触发后向停止信号 | 摄像头:%s", data.Camera)
				} else if data.ActionLevel == 1 {
					targets = []string{BackSlow}
					common.Log.Infof("[PLC] 触发后向减速信号 | 摄像头:%s", data.Camera)
				}
			case "inside":
				if data.ActionLevel == 2 {
					targets = []string{ForwardStop, BackStop}
					common.Log.Infof("[PLC] 触发双向停止信号 | 摄像头:%s", data.Camera)
				} else if data.ActionLevel == 1 {
					targets = []string{ForwardSlow, BackSlow}
					common.Log.Infof("[PLC] 触发双向减速信号 | 摄像头:%s", data.Camera)
				}
			}
			now := float64(time.Now().UnixNano()) / 1e9
			for _, t := range targets {
				c.actionHolds[t].Store(data.Camera, now)
			}

		case <-ticker.C:
			c.ResetExpiredSignals()

			if !c.connected && time.Since(lastReconnect) > 5*time.Second {
				common.Log.Infof("[PLC] 尝试重连...")
				if c.ConnectPLC() {
					common.Log.Infof("[PLC] 重连成功")
					c.StartupSequence()
				}
				lastReconnect = time.Now()
			}
		}
	}
}

// Stop 停止PLC控制
func (c *PlcController) Stop() {
	common.Log.Infof("plc关闭成功")
	close(c.stopChan)
}
