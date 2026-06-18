/**
@Time : 2026/01/16 09:30
@Author: FangYao( 方少、)
@Description: RTSP/视频流 -> FFmpeg 解码 ->推理 -> 截图 -> 告警录像 -> Ws通知
@Email: fy20030315@163.com
*/

package logic

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"go-nvr/cmd/logic/utils"
	"go-nvr/cmd/service/isql"
	"go-nvr/model"
	"go-nvr/model/request"
	"go-nvr/model/response"
	"go-nvr/pkg/common"
	"go-nvr/pkg/config"
	"go-nvr/pkg/ffmpeg"
	"go-nvr/pkg/hikisapi"
	"go-nvr/pkg/plc"
	"go-nvr/pkg/plugins"
	"gorm.io/gorm"
	"image"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"syscall"
	"time"
)

type DeviceLogic struct{}

// WebSocket
var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool { return true },
}

// LastSnapshot 截图索引
type LastSnapshot struct {
	Path      string
	Timestamp int64
}

// DetectFrame 帧数据结构体（绑定图片+帧序号）
type DetectFrame struct {
	Img     image.Image // 视频帧
	FrameID int         // 帧ID
	OrigW   int         // 视频原始宽度
	OrigH   int         // 视频原始高度
}

var lastSnapshots sync.Map // key = src

// plc 保存每个摄像头 PLC 信号状态
type SignalState struct {
	LastSent    bool    // 上次是否发送
	LastTimeSec float64 // 上次发送时间戳(秒)
}

// signalStateMap
var signalStateMap sync.Map // key = src + signal, value = *SignalState

// DetectionWSHandler 推理，图片保存，录像，plc信号传递
func (dl *DeviceLogic) DetectionWSHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("ws连接错误:", err)
		return
	}
	defer func() {
		if err := conn.Close(); err != nil {
			log.Println("WS关闭错误:", err)
		}
	}()

	src := c.Query("src")
	videoPath := utils.GetGo2RTCStreamURL(src)

	if !config.Conf.System.IsAnalysis {
		log.Printf("[%s] 模型分析已禁用", src)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}

	// 初始化 MediaMTX Path
	if config.Conf.System.EnableRecording {
		if err := utils.EnsureMediaMTXPath(src); err != nil {
			log.Printf("MediaMTX Path 初始化失败[%s]: %v", src, err)
		}
	}

	recStatus := utils.GetRecordingStatus(src)

	roiList, err := utils.LoadROIsFromDB(src)
	if err != nil {
		log.Println("加载ROI失败:", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	//defer cancel()

	// WS断开监听
	go func() {
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				log.Printf("[%s] WS客户端断开连接", src)
				cancel()

				recStatus.Mu.Lock()
				if recStatus.IsRecording {
					_ = utils.StopRecording(src)
					recStatus.IsRecording = false
				}
				recStatus.Mu.Unlock()

				return
			}
		}
	}()

	if config.Conf.System.EnableRecording {
		go utils.StartRecordingTimeoutCheck(ctx, src, 5*1000)
	}
	detectChan := make(chan DetectFrame, 16)

	var wg sync.WaitGroup

	// 推理处理线程
	wg.Add(1)

	go func() {

		defer wg.Done()

		for {

			select {

			case <-ctx.Done():
				return

			case frameData, ok := <-detectChan:

				if !ok {
					return
				}

				rgba := frameData.Img.(*image.RGBA)
				currentFrameID := frameData.FrameID // 帧ID

				origW := frameData.OrigW // 原始宽度
				origH := frameData.OrigH // 原始高度

				//clone := ffmpeg.GetRGBAClone(rgba.Rect)
				//copy(clone.Pix, rgba.Pix)
				//
				//ffmpeg.PutRGBA(rgba)

				var clone *image.RGBA
				// GPU 不拷贝 / CPU 必拷贝
				if config.Conf.Onnx.UseCuda {
					// GPU 推理：直接用原始帧，无拷贝，省 CPU
					clone = rgba
				} else {
					// CPU 推理：必须克隆拷贝，保证解码不阻塞
					clone = ffmpeg.GetRGBAClone(rgba.Rect)
					copy(clone.Pix, rgba.Pix)
					ffmpeg.PutRGBA(rgba) // 立刻归还原始帧
				}

				//// YOLO推理
				//boxes, err := common.DetectImageWithYoloFromImage(config.Conf.Onnx.DefaultModel, clone)
				//if err != nil {
				//	ffmpeg.PutRGBAClone(clone)
				//	common.Log.Infof("推理失败:", err)
				//	continue
				//}

				// 双模式推理
				var boxes []common.BoundingBox
				mode := config.Conf.Onnx.ROIDetectMode
				if mode == "" {
					mode = "filter"
				}
				imgW := clone.Bounds().Dx()
				imgH := clone.Bounds().Dy()

				if mode == "crop" && len(roiList) > 0 {
					// 先裁剪再推理
					boxes = make([]common.BoundingBox, 0)
					for _, roi := range roiList {
						roiRect := utils.PolygonToMinRect(roi.Points, imgW, imgH)
						if roiRect.Empty() {
							continue
						}
						cropImg := utils.CropImage(clone, roiRect)
						if cropImg == nil {
							continue
						}

						// 标记：是否有有效目标
						hasValidTarget := false

						cropBoxes, errTmp := common.DetectImageWithYoloFromImage(
							config.Conf.Onnx.DefaultModel, cropImg)
						if errTmp != nil {
							continue
						}
						// 坐标映射
						for _, box := range cropBoxes {
							mappedBox := utils.MapBoxToOriginal(box, roiRect.Min.X, roiRect.Min.Y)
							if utils.IsBoxInROI(mappedBox, roi.Points, imgW, imgH) &&
								utils.ContainsLabel(mappedBox, roi.Labels) {
								boxes = append(boxes, mappedBox)
								hasValidTarget = true
							}
						}
						if hasValidTarget {
							utils.SaveCroppedSnapshotOriginal(src, cropImg)
						}
					}
				} else {
					// 先推理再过滤
					boxes, err = common.DetectImageWithYoloFromImage(config.Conf.Onnx.DefaultModel, clone)
					if err != nil {
						ffmpeg.PutRGBAClone(clone)
						common.Log.Infof("推理失败:", err)
						continue
					}
				}

				filteredBoxes := make([]common.BoundingBox, 0)

				// 命中的ROI
				hitROI := make(map[string]utils.ROI)

				signalSet := make(map[string]struct{})

				for _, box := range boxes {

					// 没有ROI配置
					if len(roiList) == 0 {
						if utils.ContainsLabel(box, []string{"person"}) {
							filteredBoxes = append(filteredBoxes, box)
						}

						continue
					}

					for _, roi := range roiList {

						// 标签不匹配
						if !utils.ContainsLabel(box, roi.Labels) {
							continue
						}

						// 不在ROI区域
						if !utils.IsBoxInROI(
							box,
							roi.Points,
							clone.Bounds().Dx(),
							clone.Bounds().Dy(),
						) {
							continue
						}

						filteredBoxes = append(filteredBoxes, box)

						// 记录命中的ROI
						hitROI[roi.Name] = roi

						// 收集PLC信号
						for _, s := range roi.Signals {
							signalSet[s] = struct{}{}
						}

						break
					}
				}

				// 置信度过滤
				validBox := make([]common.BoundingBox, 0)

				for _, box := range filteredBoxes {
					if box.Confidence >= config.Conf.Onnx.Confidence {
						validBox = append(validBox, box)
					}
				}

				if len(validBox) == 0 {
					if !config.Conf.Onnx.UseCuda {
						ffmpeg.PutRGBAClone(clone)
					} else {
						ffmpeg.PutRGBA(clone)
					}
					continue
				}

				detectType := validBox[0].Label

				// 更新录像状态
				recStatus.Mu.Lock()

				recStatus.LastDetectTime = time.Now().UnixMilli()

				if config.Conf.System.EnableRecording && !recStatus.IsRecording {

					if err := utils.StartRecording(src); err != nil {
						common.Log.Error("启动录制失败:", err)
					} else {
						recStatus.IsRecording = true
						common.Log.Infof("[%s] 检测到%d目标，启动录制", src, len(validBox))
					}
				}

				recStatus.Mu.Unlock()

				// 只绘制命中的 ROI
				drawROIs := make([]utils.ROI, 0, len(hitROI))

				for _, r := range hitROI {
					drawROIs = append(drawROIs, r)
				}

				// 保存原图（不入库）
				utils.SaveSnapshotOriginal(src, clone, origW, origH)

				// 保存带ROI截图
				snapPath := utils.SaveSnapshotWithROI(src, clone, validBox, drawROIs)

				relPath, _ := filepath.Rel(config.Conf.System.SnapshotRootPath, snapPath)
				relPath = filepath.ToSlash(relPath)

				snapURL := "/snapshots/" + relPath

				lastSnapshots.Store(src, LastSnapshot{
					Path:      snapURL,
					Timestamp: time.Now().UnixMilli(),
				})

				// 异步入库
				//go func(path string, boxes []common.BoundingBox) {
				//
				//	for _, box := range boxes {
				//
				//		utils.SaveSnapshotToDB(
				//			src,
				//			box.Label,
				//			float64(box.Confidence),
				//			path,
				//			config.Conf.Onnx.DefaultModel,
				//		)
				//	}
				//
				//}(snapURL, validBox)

				// ffmpeg.PutRGBAClone(clone)

				// 手动释放帧
				if !config.Conf.Onnx.UseCuda {
					ffmpeg.PutRGBAClone(clone)
				} else {
					ffmpeg.PutRGBA(clone)
				}

				// PLC信号
				signals := make([]string, 0, len(signalSet))

				for s := range signalSet {
					signals = append(signals, s)
				}

				if len(signals) > 0 {
					SendSignalsToPLC(src, signals)
				}

				go func(path string, plcSignals []string, confidence float64) {

					for _, sig := range plcSignals {
						utils.SaveSnapshotToDB(
							src,
							sig,        // 存入PLC信号（forward_stop/back_slow等）
							confidence, // 目标置信度
							path,
							config.Conf.Onnx.DefaultModel,
						)
					}

				}(snapURL, signals, float64(validBox[0].Confidence))

				// WS推送
				_ = conn.WriteJSON(gin.H{
					"type":      detectType,
					"frame_id":  currentFrameID, // 帧序号
					"timestamp": time.Now().UnixMilli(),
					"count":     len(validBox),
					"boxes":     validBox, // 检测框数据
					"snapUrl":   snapURL,
					"signals":   signals,
				})
			}
		}
	}()

	// 视频读取 + 自动重连
	for {

		select {

		case <-ctx.Done():

			recStatus.Mu.Lock()

			if recStatus.IsRecording {
				_ = utils.StopRecording(src)
				recStatus.IsRecording = false
			}

			recStatus.Mu.Unlock()

			wg.Wait()

			return

		default:

		}

		frames, proc, err := ffmpeg.ReadVideoStream(ctx, videoPath)

		if err != nil {
			log.Println("打开视频流失败:", err)
			time.Sleep(time.Second)
			continue
		}

		func() {

			defer proc.Stop()

			frameID := 0

			for {

				select {

				case <-ctx.Done():
					return

				case img, ok := <-frames:

					if !ok {
						log.Println("视频流断开，准备重连...")
						return
					}

					// 抽帧检测
					if frameID%config.Conf.Onnx.FrameCount == 0 {

						select {

						case detectChan <- DetectFrame{
							Img:     img.Image,
							FrameID: frameID,
							OrigW:   img.OrigW,
							OrigH:   img.OrigH,
						}:

						default:
							ffmpeg.PutRGBA(img.Image.(*image.RGBA))
						}

					} else {

						ffmpeg.PutRGBA(img.Image.(*image.RGBA))

					}

					frameID++
				}
			}

		}()

		time.Sleep(time.Second)
	}
}

// SendSignalsToPLC 处理信号并发送到plc管理器
func SendSignalsToPLC(src string, signals []string) {
	plcCtrl, queue := plc.GetPLC()
	if plcCtrl == nil || queue == nil {
		return
	}

	now := float64(time.Now().UnixNano()) / 1e9

	// 构建信号集合
	signalMap := map[string]bool{}
	for _, s := range signals {
		signalMap[s] = true
	}

	// Stop 覆盖 Slow
	if signalMap["forward_stop"] {
		delete(signalMap, "forward_slow")
	}
	if signalMap["back_stop"] {
		delete(signalMap, "back_slow")
	}

	// 双向合并
	if signalMap["forward_stop"] && signalMap["back_stop"] {
		signalMap = map[string]bool{"inside_stop": true}
	}
	if signalMap["forward_slow"] && signalMap["back_slow"] {
		signalMap = map[string]bool{"inside_slow": true}
	}

	for s := range signalMap {
		key := src + ":" + s
		var state *SignalState

		v, ok := signalStateMap.Load(key)
		if !ok {
			state = &SignalState{LastSent: false, LastTimeSec: 0}
			signalStateMap.Store(key, state)
		} else {
			state = v.(*SignalState)
		}

		// 边沿触发 & 时间去抖
		if state.LastSent && now-state.LastTimeSec < 1.0 {
			continue
		}

		// 构造 CameraData
		data := plc.CameraData{
			Camera:    src,
			Timestamp: now,
		}
		switch s {
		case "forward_stop":
			data.Direction = "front"
			data.ActionLevel = 2
		case "forward_slow":
			data.Direction = "front"
			data.ActionLevel = 1
		case "back_stop":
			data.Direction = "back"
			data.ActionLevel = 2
		case "back_slow":
			data.Direction = "back"
			data.ActionLevel = 1
		case "inside_stop":
			data.Direction = "inside"
			data.ActionLevel = 2
		case "inside_slow":
			data.Direction = "inside"
			data.ActionLevel = 1
		}

		// 队列发送：Stop 阻塞，Slow 非阻塞
		blocking := s == "forward_stop" || s == "back_stop" || s == "inside_stop"
		if blocking {
			queue <- data // 阻塞
		} else {
			select {
			case queue <- data:
			default:
				common.Log.Infof("[%s] PLC队列满, 丢弃信号: %s", src, s)
				continue
			}
		}

		// 更新状态
		state.LastSent = true
		state.LastTimeSec = now
		signalStateMap.Store(key, state)

		common.Log.Infof("[%s] PLC发送信号: %s", src, s)
	}
}

// GetLastSnapshot 快照
func (dl *DeviceLogic) GetLastSnapshot(c *gin.Context) {
	src := c.Query("src")
	if src == "" {
		src = "cam1"
	}

	if v, ok := lastSnapshots.Load(src); ok {
		snap := v.(LastSnapshot)
		c.JSON(200, gin.H{
			"url":       snap.Path,
			"timestamp": snap.Timestamp,
		})
		return
	}

	c.JSON(200, gin.H{"url": ""})
}

// SaveZones roi的配置入库
func (dl *DeviceLogic) SaveZones(c *gin.Context, req interface{}) (data interface{}, rspError interface{}) {
	r, ok := req.(*request.FrontendROIReq)
	if !ok {
		return nil, ReqAssertErr
	}

	// 先检查流是否存在
	if r.Stream == "" {
		return nil, fmt.Errorf("stream", "流名称不能为空")
	}

	// 准备要保存的数据
	var (
		zoneNames  []string
		zoneColors []string
		points     [][]float32
		labels     [][]string
		signals    [][]string
	)

	for _, zone := range r.Zones {
		zoneNames = append(zoneNames, zone.Name)
		zoneColors = append(zoneColors, zone.Color)
		points = append(points, zone.Points)
		labels = append(labels, zone.Labels)
		signals = append(signals, zone.Signals)
	}

	// 转换为JSON格式
	namesJSONStr := plugins.Struct2Json(zoneNames)
	namesJSON := []byte(namesJSONStr)

	colorsJSONStr := plugins.Struct2Json(zoneColors)
	colorsJSON := []byte(colorsJSONStr)

	pointsJSONStr := plugins.Struct2Json(points)
	pointsJSON := []byte(pointsJSONStr)

	labelsJSONStr := plugins.Struct2Json(labels)
	labelsJSON := []byte(labelsJSONStr)

	signalsJSONStr := plugins.Struct2Json(signals)
	signalsJSON := []byte(signalsJSONStr)

	// 创建ROIZone数据
	zoneData := model.ROIZone{
		StreamName: r.Stream,
		Names:      namesJSON,
		Color:      colorsJSON,
		Points:     pointsJSON,
		Labels:     labelsJSON,
		Signals:    signalsJSON,
	}

	// 调用数据库层保存区域数据
	err := isql.DeviceIsql.SaveZones(r.Stream, []model.ROIZone{zoneData})
	if err != nil {
		return nil, err
	}

	return "成功保存区域配置", nil
}

// GetZones 根据流名获取上次保存的roi的配置
func (dl *DeviceLogic) GetZones(c *gin.Context, req interface{}) (data interface{}, rspError interface{}) {
	r, ok := req.(*request.GetROIReq)
	if !ok {
		return nil, ReqAssertErr
	}

	if r.StreamName == "" {
		return nil, fmt.Errorf("不能为空")
	}
	zoneDB, err := isql.DeviceIsql.GetZones(r.StreamName)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, fmt.Errorf("未查询到流[%s]的ROI区域数据", r.StreamName)
		}
		return nil, fmt.Errorf("查询ROI数据失败: %v", err)
	}
	// 定义反序列化后的数组
	var (
		names   []string
		colors  []string
		points  [][]float32
		labels  [][]string
		signals [][]string
	)
	// 反序列化数据库中的JSON字段
	if err := json.Unmarshal(zoneDB.Names, &names); err != nil {
		return nil, fmt.Errorf("反序列化区域名称失败: %v", err)
	}
	if err := json.Unmarshal(zoneDB.Color, &colors); err != nil {
		return nil, fmt.Errorf("反序列化区域颜色失败: %v", err)
	}
	if err := json.Unmarshal(zoneDB.Points, &points); err != nil {
		return nil, fmt.Errorf("反序列化坐标点失败: %v", err)
	}
	if err := json.Unmarshal(zoneDB.Labels, &labels); err != nil {
		return nil, fmt.Errorf("反序列化检测标签失败: %v", err)
	}
	if err := json.Unmarshal(zoneDB.Signals, &signals); err != nil {
		return nil, fmt.Errorf("反序列化信号标签失败: %v", err)
	}
	zoneCount := len(names)
	if len(colors) != zoneCount || len(points) != zoneCount || len(labels) != zoneCount || len(signals) != zoneCount {
		return nil, fmt.Errorf("ROI数据数组长度不一致：名称(%d)、颜色(%d)、坐标(%d)、检测标签(%d)、信号标签(%d)",
			zoneCount, len(colors), len(points), len(labels), len(signals))
	}

	// 组装嵌套响应结构体
	resp := &response.FrontendROIResp{
		Stream: r.StreamName,
		Zones:  make([]response.FrontendROIZone, zoneCount),
	}

	for i := 0; i < zoneCount; i++ {
		resp.Zones[i] = response.FrontendROIZone{
			Name:    names[i],
			Color:   colors[i],
			Points:  points[i],
			Labels:  labels[i],
			Signals: signals[i],
		}
	}

	// 返回组装好的嵌套响应结构体
	return resp, nil
}

// GetHikDeviceInfo 海康设备信息业务逻辑
func (dl *DeviceLogic) GetHikDeviceInfo(c *gin.Context, req interface{}) (data interface{}, rspError interface{}) {
	r, ok := req.(*request.HikDeviceInfoReq)
	if !ok {
		return nil, ReqAssertErr
	}
	// 调用海康ISAPI工具
	deviceInfo, err := hikisapi.GetFullDeviceInfo(r.DeviceIP, r.Port, r.Username, r.Password)
	if err != nil {
		return nil, fmt.Errorf("获取海康设备信息失败: %v", err)
	}

	return deviceInfo, nil
}

// RestartHandler 服务自我重启接口
func (dl *DeviceLogic) RestartHandler(c *gin.Context) {
	// 仅允许 POST 请求
	if c.Request.Method != "POST" {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "请求方法不允许"})
		return
	}

	// 获取当前程序可执行文件路径
	path, err := os.Executable()
	if err != nil {
		common.Log.Error("[api] 获取程序路径失败:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "服务重启失败"})
		return
	}

	common.Log.Info("[api] 服务重启指令已接收，程序路径:", path)

	// 先返回 HTTP 响应给客户端
	c.JSON(http.StatusOK, gin.H{
		"msg":  "服务重启指令已执行",
		"path": path,
	})

	// 异步执行重启逻辑
	go func() {
		time.Sleep(300 * time.Millisecond) // 确保 HTTP 响应返回

		common.StopMediaMtx()
		common.StopGo2RTC()

		// Windows 下：启动新进程 + 退出当前程序
		if runtime.GOOS == "windows" {
			cmd := exec.Command(path, os.Args[1:]...)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			cmd.Stdin = os.Stdin
			if err := cmd.Start(); err != nil {
				common.Log.Error("[api] 重启失败:", err)
				return
			}
			common.Log.Info("[api] 新进程已启动，当前进程退出")
			os.Exit(0)
		} else {
			// Linux/macOS 下使用 syscall.Exec 替换进程
			_ = syscall.Exec(path, os.Args, os.Environ())
		}
	}()
}

func (dl *DeviceLogic) Ping(c *gin.Context) {
	c.JSON(200, gin.H{"status": "ok"})
}

var allowedConfigs = map[string]string{
	"go2rtc":   "go2rtc.yaml",
	"mediamtx": "mediamtx.yml",
	"config":   "config.yaml",
}

// ReadConfigHandler 加载配置文件
func (dl *DeviceLogic) ReadConfigHandler(c *gin.Context) {
	fileKey := strings.ToLower(c.Query("file"))
	fileName, ok := allowedConfigs[fileKey]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "不支持的配置文件"})
		return
	}

	filePath := filepath.Join(".", fileName)
	data, err := os.ReadFile(filePath)
	if err != nil {
		common.Log.Error("[config] 读取配置文件失败:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "读取配置失败", "error": err.Error()})
		return
	}

	c.Data(http.StatusOK, "text/yaml", data)
}

// SaveConfigHandler 保存配置文件
func (dl *DeviceLogic) SaveConfigHandler(c *gin.Context) {
	fileKey := strings.ToLower(c.Query("file"))
	fileName, ok := allowedConfigs[fileKey]
	if !ok {
		c.JSON(http.StatusBadRequest, gin.H{"msg": "不支持的配置文件"})
		return
	}

	filePath := filepath.Join(".", fileName)
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		common.Log.Error("[config] 读取请求体失败:", err)
		c.JSON(http.StatusBadRequest, gin.H{"msg": "读取请求体失败", "error": err.Error()})
		return
	}

	if err := os.WriteFile(filePath, body, 0644); err != nil {
		common.Log.Error("[config] 写入配置文件失败:", err)
		c.JSON(http.StatusInternalServerError, gin.H{"msg": "保存配置失败", "error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"msg": "配置保存成功"})
}

// GetSnapshot 告警界面获取快照信息
func (dl *DeviceLogic) GetSnapshot(c *gin.Context, req interface{}) (data interface{}, rspError interface{}) {
	r, ok := req.(*request.SnapshotReq)
	if !ok {
		return nil, ReqAssertErr
	}
	list, total, err := isql.DeviceIsql.GetSnapshot(r)
	if err != nil {
		return nil, plugins.NewMySqlError(fmt.Errorf("%s", "获取快照失败："+err.Error()))
	}
	// 指针切片转值切片
	snapshotList := make([]model.Snapshot, 0, len(list))
	for _, ptr := range list {
		if ptr != nil {
			snapshotList = append(snapshotList, *ptr)
		}
	}
	return response.SnapshotListRsp{
		Total: total,
		List:  snapshotList,
	}, nil
}
