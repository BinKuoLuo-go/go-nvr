/**
@Time : 2026/01/20 15:03
@Author: FangYao( 方少、)
@Description:图片相关工具方法
@Email: fy20030315@163.com
*/

package utils

import (
	"encoding/json"
	"fmt"
	"github.com/fogleman/gg"
	"github.com/nfnt/resize"
	"go-nvr/cmd/service/isql"
	"go-nvr/model"
	"go-nvr/pkg/common"
	"go-nvr/pkg/config"
	"image"
	"image/draw"
	"image/jpeg"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	LineWidth            = 2.5                // 边框宽度
	FillOpacity          = 0.3                // 半透明填充透明度
	FontSize             = 20                 // 标签字体大小
	LabelPadding         = 4                  // 标签内边距
	ROILineWidth         = 3                  // ROI边框宽度
	ROIFillOpacity       = 0.35               // ROI填充透明度
	DefaultFontPath      = "./font/basic.ttf" // 路径字体
	ImageQualityOriginal = 85                 // 原图图片质量
	ImageQualityROI      = 60                 // roi图压缩质量
)

// ROI 定义
type ROI struct {
	Name    string    `json:"name"`
	Points  []float32 `json:"points"` // [x0,y0,x1,y1,...] 相对比例 0~1
	Color   string    `json:"color"`
	Labels  []string  `json:"labels"` // 允许检测的标签列表
	Signals []string  `json:"signals"`
}

// SaveSnapshotOriginal 保存原图
//func SaveSnapshotOriginal(src string, img image.Image) string {
//	dateDir := time.Now().Format("20060102")
//	saveDir := filepath.Join(config.Conf.System.SnapshotRootPath, src, "original", dateDir)
//	_ = os.MkdirAll(saveDir, os.ModePerm)
//	fileName := fmt.Sprintf("person_%d.jpg", time.Now().UnixMilli())
//	fullPath := filepath.Join(saveDir, fileName)
//	f, err := os.Create(fullPath)
//	if err != nil {
//		return ""
//	}
//	defer f.Close()
//
//	_ = jpeg.Encode(f, img, &jpeg.Options{Quality: ImageQualityOriginal})
//
//	return fullPath
//}

// SaveSnapshotOriginal 保存原图
func SaveSnapshotOriginal(src string, img image.Image, origW, origH int) string {
	dateDir := time.Now().Format("20060102")
	saveDir := filepath.Join(config.Conf.System.SnapshotRootPath, src, "original", dateDir)
	_ = os.MkdirAll(saveDir, os.ModePerm)
	fileName := fmt.Sprintf("person_%d.jpg", time.Now().UnixMilli())
	fullPath := filepath.Join(saveDir, fileName)
	f, err := os.Create(fullPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	// 自动还原为原始分辨率
	var saveImg = img
	if origW > 0 && origH > 0 && (img.Bounds().Dx() != origW || img.Bounds().Dy() != origH) {
		saveImg = resize.Resize(uint(origW), uint(origH), img, resize.Lanczos3)
	}
	_ = jpeg.Encode(f, saveImg, &jpeg.Options{Quality: ImageQualityOriginal})
	return fullPath
}

// SaveSnapshotWithROI  ROI绘制
func SaveSnapshotWithROI(src string, img image.Image, boxes []common.BoundingBox, roiList []ROI) string {
	dateDir := time.Now().Format("20060102")
	saveDir := filepath.Join(config.Conf.System.SnapshotRootPath, src, "roi", dateDir)
	if err := os.MkdirAll(saveDir, os.ModePerm); err != nil {
		log.Printf("创建ROI目录失败: %v", err)
		return ""
	}
	fileName := fmt.Sprintf("person_%d.jpg", time.Now().UnixMilli())
	//fileName := fmt.Sprintf("person_%d.webp", time.Now().UnixMilli())
	fullPath := filepath.Join(saveDir, fileName)

	dc := gg.NewContextForImage(img)
	dc.SetLineWidth(LineWidth)

	// 加载字体
	if err := dc.LoadFontFace(DefaultFontPath, FontSize); err != nil {
		log.Printf("加载自定义字体失败: %v, 使用默认字体", err)
	}

	w := float64(img.Bounds().Dx())
	h := float64(img.Bounds().Dy())

	// 绘制ROI区域
	for _, roi := range roiList {
		if len(roi.Points) < 4 || len(roi.Points)%2 != 0 {
			continue
		}
		r, g, b := HexToRGB(roi.Color)
		// 半透明填充
		dc.SetRGBA(float64(r)/255, float64(g)/255, float64(b)/255, ROIFillOpacity)

		dc.MoveTo(float64(roi.Points[0])*w, float64(roi.Points[1])*h)
		for i := 1; i < len(roi.Points)/2; i++ {
			x := float64(roi.Points[i*2]) * w
			y := float64(roi.Points[i*2+1]) * h
			dc.LineTo(x, y)
		}
		dc.ClosePath()
		dc.Fill()

		// 边框
		dc.SetRGB(float64(r)/255, float64(g)/255, float64(b)/255)
		dc.SetLineWidth(ROILineWidth)
		dc.Stroke()
	}

	// 绘制检测框 + 内嵌标签
	for _, box := range boxes {
		x1, y1 := float64(box.X1), float64(box.Y1)
		x2, y2 := float64(box.X2), float64(box.Y2)
		boxWidth := x2 - x1
		boxHeight := y2 - y1

		// 配色
		var r, g, b float64
		switch box.Label {
		case "person":
			r, g, b = 1, 0, 0
		default:
			r, g, b = 0, 1, 0
		}

		// 半透明填充
		dc.SetRGBA(r, g, b, FillOpacity)
		dc.DrawRectangle(x1, y1, boxWidth, boxHeight)
		dc.Fill()

		// 实心边框
		dc.SetRGB(r, g, b)
		dc.SetLineWidth(LineWidth)
		dc.DrawRectangle(x1, y1, boxWidth, boxHeight)
		dc.Stroke()

		//标签内嵌在检测框左上角内部
		labelText := fmt.Sprintf("%s %.2f", box.Label, box.Confidence)
		textW, textH := dc.MeasureString(labelText)

		// 坐标：框内，左上角
		labelBgX := x1 + LabelPadding
		labelBgY := y1 + LabelPadding
		labelBgW := textW + LabelPadding*2
		labelBgH := textH + LabelPadding

		// 标签半透明背景
		dc.SetRGBA(r, g, b, FillOpacity)
		dc.DrawRectangle(labelBgX, labelBgY, labelBgW, labelBgH)
		dc.Fill()

		// 白色文字（内嵌显示）
		dc.SetRGB(1, 1, 1)
		dc.DrawString(labelText, labelBgX+LabelPadding, labelBgY+textH)
	}

	// 保存图片
	f, err := os.Create(fullPath)
	if err != nil {
		log.Printf("创建ROI图片失败: %v", err)
		return ""
	}
	defer f.Close()

	// 编码为jepg
	if err := jpeg.Encode(f, dc.Image(), &jpeg.Options{Quality: ImageQualityROI}); err != nil {
		log.Printf("保存ROI图片为jpg失败: %v", err)
		return ""
	}

	// 编码为webp
	//if err := nativewebp.Encode(f, dc.Image(), nil); err != nil {
	//	log.Printf("保存ROI图片为webp格式失败: %v", err)
	//	return ""
	//}

	return fullPath
}

// hexToRGB 十六进制颜色转RGB
func HexToRGB(hex string) (uint8, uint8, uint8) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) != 6 {
		return 255, 0, 0 // 默认红色
	}
	r, _ := strconv.ParseUint(hex[0:2], 16, 8)
	g, _ := strconv.ParseUint(hex[2:4], 16, 8)
	b, _ := strconv.ParseUint(hex[4:6], 16, 8)
	return uint8(r), uint8(g), uint8(b)
}

// ContainsLabel
func ContainsLabel(box common.BoundingBox, labels []string) bool {
	for _, l := range labels {
		if box.Label == l {
			return true
		}
	}
	return false
}

// 点是否在多边形内
func IsBoxInROI(box common.BoundingBox, roiPoints []float32, w, h int) bool {
	poly := make([][2]float32, len(roiPoints)/2)
	for i := 0; i < len(roiPoints)/2; i++ {
		poly[i][0] = roiPoints[i*2] * float32(w)
		poly[i][1] = roiPoints[i*2+1] * float32(h)
	}
	cx := (box.X1 + box.X2) / 2
	cy := (box.Y1 + box.Y2) / 2
	return pointInPolygon(cx, cy, poly)
}

// pointInPolygon
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

// LoadROIsFromDB 从数据库加载ROI配置并处理
func LoadROIsFromDB(stream string) ([]ROI, error) {
	dbZone, err := isql.DeviceIsql.GetZones(stream)
	if err != nil {
		// 没配置 ROI，不算错误
		return nil, nil
	}

	var (
		names   []string
		colors  []string
		points  [][]float32
		labels  [][]string
		signals [][]string
	)

	if err := json.Unmarshal(dbZone.Names, &names); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(dbZone.Color, &colors); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(dbZone.Points, &points); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(dbZone.Labels, &labels); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(dbZone.Signals, &signals); err != nil {
		return nil, err
	}

	roiList := make([]ROI, 0, len(names))
	for i := range names {
		roiList = append(roiList, ROI{
			Name:    names[i],
			Color:   colors[i],
			Points:  points[i],
			Labels:  labels[i],
			Signals: signals[i],
		})
	}

	return roiList, nil
}

// SaveSnapshotToDB 快照信息入库
func SaveSnapshotToDB(
	stream string,
	label string,
	score float64,
	imgPath string,
	modelName string,
) {

	snap := &model.Snapshot{
		StreamName: stream,
		RecordAt:   time.Now(),
		Label:      label,
		Score:      score,
		ImagePath:  imgPath,
		Model:      modelName,
	}

	err := isql.DeviceIsql.SaveSnapshot(snap)
	if err != nil {
		log.Println("快照入库失败:", err)
	}
}

// PolygonToMinRect 将多边形ROI点转为 最小外接矩形
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
	// 边界保护
	return image.Rect(
		maxx(0, int(minX)),
		maxx(0, int(minY)),
		minn(w, int(maxX)),
		minn(h, int(maxY)),
	)
}

// CropImage 裁剪图像
func CropImage(img image.Image, rect image.Rectangle) image.Image {
	rect = rect.Intersect(img.Bounds())
	if rect.Empty() {
		return nil
	}
	dst := image.NewRGBA(rect)
	draw.Draw(dst, dst.Bounds(), img, rect.Min, draw.Src)
	return dst
}

// MapBoxToOriginal 将裁剪图的检测框 映射回 原图坐标
func MapBoxToOriginal(box common.BoundingBox, offsetX, offsetY int) common.BoundingBox {
	return common.BoundingBox{
		Label:      box.Label,
		Confidence: box.Confidence,
		X1:         box.X1 + float32(offsetX),
		Y1:         box.Y1 + float32(offsetY),
		X2:         box.X2 + float32(offsetX),
		Y2:         box.Y2 + float32(offsetY),
	}
}

// SaveCroppedSnapshotOriginal 保存裁剪后的原始小图
func SaveCroppedSnapshotOriginal(src string, img image.Image) string {
	dateDir := time.Now().Format("20060102")
	saveDir := filepath.Join(config.Conf.System.SnapshotRootPath, src, "crop_original", dateDir)
	_ = os.MkdirAll(saveDir, os.ModePerm)
	fileName := fmt.Sprintf("crop_raw_%d.jpg", time.Now().UnixMilli())
	fullPath := filepath.Join(saveDir, fileName)
	f, err := os.Create(fullPath)
	if err != nil {
		return ""
	}
	defer f.Close()

	// 高质量保存裁剪原图
	_ = jpeg.Encode(f, img, &jpeg.Options{Quality: ImageQualityOriginal})

	return fullPath
}

// max/min 辅助函数
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
