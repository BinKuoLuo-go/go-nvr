/*
*
@Time : 2026/03/24
@Author: FangYao( 方少、)
@Description: 测试 仅检测人物 + 本地视频读取 + 裁剪ROI推理+保存裁剪图
*/
package common

import (
	"context"
	"fmt"
	"github.com/fogleman/gg"
	"go-nvr/pkg/config"
	"go-nvr/pkg/ffmpeg"
	"go.uber.org/zap"
	"image"
	"image/draw"
	"image/jpeg"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

// 独立常量
const (
	LineWidth       = 2.5
	FillOpacity     = 0.3
	FontSize        = 20
	LabelPadding    = 4
	DefaultFontPath = "./font/basic.ttf"
	ImageQuality    = 85
	modelName       = "yolo-v11n" // 固定模型
)

// 通用工具函数
func maxx(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func minn(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 裁剪推理专用
func PolygonToMinRect(points []float32, w, h int) image.Rectangle {
	if len(points) < 4 || len(points)%2 != 0 {
		return image.Rect(0, 0, w, h)
	}
	fw, fh := float32(w), float32(h)
	minX, minY := fw, fh
	maxX, maxY := float32(0), float32(0)

	for i := 0; i < len(points); i += 2 {
		x := points[i] * fw
		y := points[i+1] * fh
		if x < minX {
			minX = x
		}
		if x > maxX {
			maxX = x
		}
		if y < minY {
			minY = y
		}
		if y > maxY {
			maxY = y
		}
	}
	return image.Rect(maxx(0, int(minX)), maxx(0, int(minY)), minn(w, int(maxX)), minn(h, int(maxY)))
}
func CropImage(img image.Image, rect image.Rectangle) image.Image {
	rect = rect.Intersect(img.Bounds())
	if rect.Empty() {
		return nil
	}
	dst := image.NewRGBA(rect)
	draw.Draw(dst, dst.Bounds(), img, rect.Min, draw.Src)
	return dst
}
func MapBoxToOriginal(box BoundingBox, offsetX, offsetY int) BoundingBox {
	return BoundingBox{
		Label:      box.Label,
		Confidence: box.Confidence,
		X1:         box.X1 + float32(offsetX), Y1: box.Y1 + float32(offsetY),
		X2: box.X2 + float32(offsetX), Y2: box.Y2 + float32(offsetY),
	}
}

// Filter模式专用：判断目标是否在ROI多边形内
func pointInPolygon(x, y float32, poly [][2]float32) bool {
	n := len(poly)
	inside := false
	j := n - 1
	for i := 0; i < n; i++ {
		xi, yi := poly[i][0], poly[i][1]
		xj, yj := poly[j][0], poly[j][1]
		if ((yi > y) != (yj > y)) && (x < (xj-xi)*(y-yi)/(yj-yi)+xi) {
			inside = !inside
		}
		j = i
	}
	return inside
}
func IsBoxInROI(box BoundingBox, roiPoints []float32, w, h int) bool {
	poly := make([][2]float32, len(roiPoints)/2)
	for i := 0; i < len(roiPoints)/2; i++ {
		poly[i][0] = roiPoints[i*2] * float32(w)
		poly[i][1] = roiPoints[i*2+1] * float32(h)
	}
	cx := (box.X1 + box.X2) / 2
	cy := (box.Y1 + box.Y2) / 2
	return pointInPolygon(cx, cy, poly)
}

// 视频读取
func readLocalVideo(ctx context.Context, videoPath string) (<-chan image.Image, error) {
	width, height, err := ffmpeg.GetVideoSize(videoPath)
	if err != nil {
		return nil, err
	}
	frameSize := width * height * 4

	args := []string{"-i", videoPath, "-f", "rawvideo", "-pix_fmt", "rgba", "-an", "-sn", "-loglevel", "error", "-"}
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, err
	}

	ch := make(chan image.Image, 5)
	go func() {
		defer close(ch)
		defer cmd.Wait()
		buf := make([]byte, frameSize)
		for {
			select {
			case <-ctx.Done():
				cmd.Process.Kill()
				return
			default:
			}
			_, err := io.ReadFull(stdout, buf)
			if err != nil {
				cmd.Process.Kill()
				return
			}
			img := image.NewRGBA(image.Rect(0, 0, width, height))
			copy(img.Pix, buf)
			ch <- img
		}
	}()
	return ch, nil
}

func mockConfigForTest() {
	config.Conf = &config.Config{
		Onnx: &config.OnnxConfig{
			UseCuda:      true,
			DefaultModel: modelName,
			Confidence:   0.5,
			Base:         &config.OnnxBaseConfig{InputName: "images", OutputName: "output0"},
			Models: map[string]*config.DetectConfig{
				"yolo-v11n": {ModelPath: "../../onnxModel/yolo11n.onnx", InputShape: []int64{1, 3, 640, 640}, OutputShape: []int64{1, 84, 8400}},
			},
			ROIDetectMode: "crop", // crop / filter
		},
	}
}

// 绘图保存
func initLoggerForTest() {
	logger, _ := zap.NewDevelopment()
	Log = logger.Sugar()
}
func saveDetectImage(img image.Image, boxes []BoundingBox, savePath string) string {
	dc := gg.NewContextForImage(img)
	dc.SetLineWidth(LineWidth)
	_ = dc.LoadFontFace(DefaultFontPath, FontSize)

	for _, box := range boxes {
		x1, y1 := float64(box.X1), float64(box.Y1)
		x2, y2 := float64(box.X2), float64(box.Y2)
		r, g, b := 1.0, 0.0, 0.0

		dc.SetRGBA(r, g, b, FillOpacity)
		dc.DrawRectangle(x1, y1, x2-x1, y2-y1)
		dc.Fill()
		dc.SetRGB(r, g, b)
		dc.DrawRectangle(x1, y1, x2-x1, y2-y1)
		dc.Stroke()

		labelText := fmt.Sprintf("person %.2f", box.Confidence)
		textW, textH := dc.MeasureString(labelText)
		bgX := x1 + LabelPadding
		bgY := y1 + LabelPadding
		dc.SetRGBA(r, g, b, FillOpacity)
		dc.DrawRectangle(bgX, bgY, textW+LabelPadding*2, textH+LabelPadding)
		dc.Fill()
		dc.SetRGB(1, 1, 1)
		dc.DrawString(labelText, bgX+LabelPadding, bgY+textH)
	}

	_ = os.MkdirAll(filepath.Dir(savePath), os.ModePerm)
	file, err := os.Create(savePath)
	if err != nil {
		return ""
	}
	defer file.Close()
	_ = jpeg.Encode(file, dc.Image(), &jpeg.Options{Quality: ImageQuality})
	return savePath
}
func parseMockROIPoints() []float32 {
	// 统一ROI区域：画面中间80%
	return []float32{0.2, 0.2, 0.8, 0.2, 0.8, 0.8, 0.2, 0.8}
}

// 测试入口
func TestROIMode(t *testing.T) {
	mockConfigForTest()
	initLoggerForTest()

	// 初始化模型
	if err := InitOnnxModels(); err != nil {
		t.Fatalf("模型初始化失败: %v", err)
	}
	defer DestroyAllOnnxModels()

	// 测试视频
	videoPaths := []string{
		"E:\\FyProject\\Go\\porject\\go-nvr\\testVide\\test_video6.mp4",
		"E:\\FyProject\\Go\\porject\\go-nvr\\testVide\\test_video7.mp4",
	}
	roiPoints := parseMockROIPoints()
	mode := config.Conf.Onnx.ROIDetectMode // 获取当前模式
	saveFrameInterval := 2
	t.Logf("当前测试模式: [%s] 模式", mode)

	// 遍历视频
	for idx, videoPath := range videoPaths {
		if _, err := os.Stat(videoPath); os.IsNotExist(err) {
			t.Logf("视频不存在: %s", videoPath)
			continue
		}
		videoName := filepath.Base(videoPath)
		t.Logf("\n===== 第%d个视频: %s =====", idx+1, videoName)

		// 根据模式自动生成保存路径
		baseSaveDir := fmt.Sprintf("yolo_%s_test", mode)
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()

		frameChan, err := readLocalVideo(ctx, videoPath)
		if err != nil {
			t.Logf("打开视频失败: %v", err)
			continue
		}

		frameCount, saveCount := 0, 0
		startTime := time.Now()

		// 处理每一帧
		for img := range frameChan {
			frameCount++
			if frameCount%saveFrameInterval != 0 {
				continue
			}
			imgW, imgH := img.Bounds().Dx(), img.Bounds().Dy()
			var personBoxes []BoundingBox
			var cropImg image.Image // 裁剪图（仅crop模式用）

			// 自动切换模式逻辑
			if mode == "crop" {
				// Crop模式裁剪区域 → 小图推理 → 坐标映射
				roiRect := PolygonToMinRect(roiPoints, imgW, imgH)
				cropImg = CropImage(img, roiRect)
				if cropImg == nil {
					continue
				}
				cropBoxes, err := DetectImageWithYoloFromImage(modelName, cropImg)
				if err != nil {
					t.Logf("第%d帧推理失败: %v", frameCount, err)
					continue
				}
				// 映射坐标+过滤人物
				for _, box := range cropBoxes {
					mappedBox := MapBoxToOriginal(box, roiRect.Min.X, roiRect.Min.Y)
					if mappedBox.Label == "person" && mappedBox.Confidence >= config.Conf.Onnx.Confidence {
						personBoxes = append(personBoxes, mappedBox)
					}
				}
			} else if mode == "filter" {
				// 全图推理 → ROI过滤
				allBoxes, err := DetectImageWithYoloFromImage(modelName, img)
				if err != nil {
					t.Logf("第%d帧推理失败: %v", frameCount, err)
					continue
				}
				// 过滤人物+ROI内目标
				for _, box := range allBoxes {
					if box.Label == "person" && box.Confidence >= config.Conf.Onnx.Confidence && IsBoxInROI(box, roiPoints, imgW, imgH) {
						personBoxes = append(personBoxes, box)
					}
				}
			}

			// 无目标则跳过
			if len(personBoxes) == 0 {
				continue
			}

			//自动保存图片
			fileName := fmt.Sprintf("frame_%06d.jpg", frameCount)
			if mode == "crop" {
				// Crop模式：保存裁剪小图 + 原图
				saveDetectImage(cropImg, personBoxes, filepath.Join(baseSaveDir, videoName, "crop", fileName))
				saveDetectImage(img, personBoxes, filepath.Join(baseSaveDir, videoName, "original", fileName))
			} else {
				// Filter模式：保存过滤后结果
				saveDetectImage(img, personBoxes, filepath.Join(baseSaveDir, videoName, "filter_result", fileName))
			}

			saveCount++
			t.Logf("第%d帧 | 检测到人物: %d | 保存成功", frameCount, len(personBoxes))
		}

		t.Logf("===== 视频处理完成 | 总帧数:%d | 保存:%d | 耗时:%v =====", frameCount, saveCount, time.Since(startTime))
	}

	t.Logf("\n[%s]模式测试全部完成！", mode)
	if mode == "crop" {
		t.Log("图片路径: ./yolo_crop_test/")
	} else {
		t.Log("图片路径: ./yolo_filter_test/")
	}
}
