/**
@Time : 2026/01/20 08:44
@Author: FangYao( 方少、)
@Description: yolo 模型相关预处理
@Email: fy20030315@163.com
*/

package common

import (
	"fmt"
	"github.com/nfnt/resize"
	ort "github.com/yalue/onnxruntime_go"
	"go-nvr/pkg/config"
	"go-nvr/pkg/grpc"
	"image"
	"image/color"
	"image/draw"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"sort"
)

// 数据结构
type BoundingBox struct {
	Label      string
	Confidence float32
	X1, Y1     float32
	X2, Y2     float32
}

func (b BoundingBox) Rect() image.Rectangle {
	return image.Rect(
		int(b.X1), int(b.Y1),
		int(b.X2), int(b.Y2),
	).Canon()
}

type FrameDetections struct {
	FrameID int           `json:"frameId"`
	Boxes   []BoundingBox `json:"boxes"`
}

// 转 RGBA
func toRGBA(img image.Image) *image.RGBA {
	if rgba, ok := img.(*image.RGBA); ok {
		return rgba
	}
	bounds := img.Bounds()
	rgba := image.NewRGBA(bounds)
	draw.Draw(rgba, bounds, img, bounds.Min, draw.Src)
	return rgba
}

// Tensor填充
func fillTensorFromImage(img image.Image, input *ort.Tensor[float32], width, height int) error {

	data := input.GetData()
	channelSize := width * height
	if len(data) < channelSize*3 {
		return fmt.Errorf("输入张量尺寸不匹配")
	}

	// resize 算法：NearestNeighbor:最近邻插值  Bilinear:双线性插值   Lanczos
	// 如果关闭 ffmpeg 缩放，则在这里使用 nfnt/resize 缩放
	//if !config.Conf.Onnx.FFmpegResize {
	//	img = resize.Resize(uint(width), uint(height), img, resize.NearestNeighbor)
	//}
	imgProcessed, _, _, _ := Letterbox(img, width, height)
	// 强制RGBA
	rgba := toRGBA(imgProcessed)

	imgW := rgba.Bounds().Dx()
	imgH := rgba.Bounds().Dy()
	if imgW != width || imgH != height {
		return fmt.Errorf("图片缩放失败: 期望 %dx%d, 实际 %dx%d", width, height, imgW, imgH)
	}

	pix := rgba.Pix
	stride := rgba.Stride

	r := data[0:channelSize]
	g := data[channelSize : channelSize*2]
	b := data[channelSize*2 : channelSize*3]

	inv255 := float32(1.0 / 255.0)

	i := 0
	for y := 0; y < height; y++ {
		row := pix[y*stride:]
		for x := 0; x < width; x++ {
			idx := x * 4

			r[i] = float32(row[idx]) * inv255
			g[i] = float32(row[idx+1]) * inv255
			b[i] = float32(row[idx+2]) * inv255

			i++
		}
	}

	return nil
}

// processYoloOutput YOLO后处理
//func processYoloOutput(output []float32, origW, origH int, inputW, inputH int, confThreshold float32, numClasses int) []BoundingBox {
//
//	boxes := make([]BoundingBox, 0)
//	stride := len(output) / (numClasses + 4)
//
//	for i := 0; i < stride; i++ {
//		bestClass := -1
//		bestScore := float32(-1e9)
//
//		for c := 0; c < numClasses; c++ {
//			score := output[(c+4)*stride+i]
//			if score > bestScore {
//				bestScore = score
//				bestClass = c
//			}
//		}
//
//		if bestScore < confThreshold {
//			continue
//		}
//
//		xc := output[i]
//		yc := output[stride+i]
//		w := output[2*stride+i]
//		h := output[3*stride+i]
//
//		x1 := (xc - w/2) / float32(inputW) * float32(origW)
//		y1 := (yc - h/2) / float32(inputH) * float32(origH)
//		x2 := (xc + w/2) / float32(inputW) * float32(origW)
//		y2 := (yc + h/2) / float32(inputH) * float32(origH)
//
//		boxes = append(boxes, BoundingBox{
//			Label:      yoloClasses[bestClass],
//			Confidence: bestScore,
//			X1:         x1,
//			Y1:         y1,
//			X2:         x2,
//			Y2:         y2,
//		})
//	}
//
//	sort.Slice(boxes, func(i, j int) bool {
//		return boxes[i].Confidence > boxes[j].Confidence
//	})
//
//	return nms(boxes, config.Conf.Onnx.NmsThreshold)
//}

// processYoloOutput YOLO后处理：仅ONNX推理使用
func processYoloOutput(output []float32, origW, origH int, inputW, inputH int, confThreshold float32, numClasses int) []BoundingBox {
	// 先计算letterbox的缩放比例和偏移
	scale := math.Min(float64(inputW)/float64(origW), float64(inputH)/float64(origH))
	padX := (inputW - int(float64(origW)*scale)) / 2
	padY := (inputH - int(float64(origH)*scale)) / 2

	boxes := make([]BoundingBox, 0)
	stride := len(output) / (numClasses + 4)

	for i := 0; i < stride; i++ {
		bestClass := -1
		bestScore := float32(-1e9)

		for c := 0; c < numClasses; c++ {
			score := output[(c+4)*stride+i]
			if score > bestScore {
				bestScore = score
				bestClass = c
			}
		}

		if bestScore < confThreshold {
			continue
		}

		xc := output[i]
		yc := output[stride+i]
		w := output[2*stride+i]
		h := output[3*stride+i]

		// 640尺寸下的xyxy
		x1_640 := xc - w/2
		y1_640 := yc - h/2
		x2_640 := xc + w/2
		y2_640 := yc + h/2

		// 减去填充偏移
		x1_noscale := x1_640 - float32(padX)
		x2_noscale := x2_640 - float32(padX)
		y1_noscale := y1_640 - float32(padY)
		y2_noscale := y2_640 - float32(padY)

		// 缩放回原图尺寸
		x1 := x1_noscale / float32(scale)
		y1 := y1_noscale / float32(scale)
		x2 := x2_noscale / float32(scale)
		y2 := y2_noscale / float32(scale)

		boxes = append(boxes, BoundingBox{
			Label:      yoloClasses[bestClass],
			Confidence: bestScore,
			X1:         max(0, x1),
			Y1:         max(0, y1),
			X2:         min(float32(origW), x2),
			Y2:         min(float32(origH), y2),
		})
	}

	sort.Slice(boxes, func(i, j int) bool {
		return boxes[i].Confidence > boxes[j].Confidence
	})

	return nms(boxes, config.Conf.Onnx.NmsThreshold)
}

// NMS 非极大值抑制
func nms(boxes []BoundingBox, threshold float32) []BoundingBox {
	result := make([]BoundingBox, 0)

	for _, box := range boxes {
		keep := true
		for _, kept := range result {
			if iou(box, kept) > threshold {
				keep = false
				break
			}
		}
		if keep {
			result = append(result, box)
		}
	}

	return result
}

func iou(a, b BoundingBox) float32 {
	r1 := a.Rect()
	r2 := b.Rect()
	inter := r1.Intersect(r2)
	if inter.Empty() {
		return 0
	}
	interArea := inter.Dx() * inter.Dy()
	area1 := r1.Dx() * r1.Dy()
	area2 := r2.Dx() * r2.Dy()
	return float32(interArea) / float32(area1+area2-interArea)
}

// DetectImageWithYoloFromImage 推理入口
func DetectImageWithYoloFromImage(modelType string, img image.Image) ([]BoundingBox, error) {

	//session, err := AcquireSession(modelType)
	//if err != nil {
	//	return nil, err
	//}
	//defer ReleaseSession(modelType, session)
	//
	//detectConf, err := GetDetectConfig(modelType)
	//if err != nil {
	//	return nil, err
	//}
	//
	//inputW := int(detectConf.InputShape[3])
	//inputH := int(detectConf.InputShape[2])
	//
	//// 预处理
	//if err := fillTensorFromImage(img, session.Input, inputW, inputH); err != nil {
	//	return nil, err
	//}
	//
	//// 推理
	//if err := session.Session.Run(); err != nil {
	//	return nil, err
	//}
	//
	//// 后处理
	//return processYoloOutput(
	//	session.Output.GetData(),
	//	img.Bounds().Dx(),
	//	img.Bounds().Dy(),
	//	inputW,
	//	inputH,
	//	config.Conf.Onnx.Confidence,
	//	80,
	//), nil

	// 读取配置
	engine := config.Conf.Onnx.InferEngine
	if engine == "" {
		engine = "onnx"
	}

	// ONNX 推理
	if engine == "onnx" {
		session, err := AcquireSession(modelType)
		if err != nil {
			return nil, err
		}
		defer ReleaseSession(modelType, session)

		detectConf, err := GetDetectConfig(modelType)
		if err != nil {
			return nil, err
		}

		inputW := int(detectConf.InputShape[3])
		inputH := int(detectConf.InputShape[2])

		// 预处理
		if err := fillTensorFromImage(img, session.Input, inputW, inputH); err != nil {
			return nil, err
		}

		// 推理
		if err := session.Session.Run(); err != nil {
			return nil, err
		}

		// 后处理
		return processYoloOutput(
			session.Output.GetData(),
			img.Bounds().Dx(),
			img.Bounds().Dy(),
			inputW,
			inputH,
			config.Conf.Onnx.Confidence,
			int(detectConf.OutputShape[1])-4,
		), nil
	}

	// TensorRT gRPC 推理
	if engine == "tensorrt" {
		// 转换为RGBA
		rgba := toRGBA(img)
		// 调用gRPC
		boxes, err := grpc.Infer(
			rgba.Pix,
			int32(rgba.Bounds().Dx()),
			int32(rgba.Bounds().Dy()),
			config.Conf.Onnx.Confidence,
			config.Conf.Onnx.NmsThreshold,
			modelType,
		)
		if err != nil {
			return nil, err
		}

		// 转换为项目内部格式
		result := make([]BoundingBox, len(boxes))
		for i, b := range boxes {
			result[i] = BoundingBox{
				Label:      b.Label,
				Confidence: b.Confidence,
				X1:         b.X1,
				Y1:         b.Y1,
				X2:         b.X2,
				Y2:         b.Y2,
			}
		}
		return result, nil
	}

	return nil, fmt.Errorf("不支持的推理引擎: %s", engine)

}

// Letterbox 与Ultralytics YOLO完全一致的等比缩放+灰边填充
func Letterbox(img image.Image, targetWidth, targetHeight int) (image.Image, float64, int, int) {
	srcW := img.Bounds().Dx()
	srcH := img.Bounds().Dy()

	// 计算等比缩放比例
	scale := math.Min(float64(targetWidth)/float64(srcW), float64(targetHeight)/float64(srcH))
	newW := int(float64(srcW) * scale)
	newH := int(float64(srcH) * scale)

	// 等比缩放
	resized := resize.Resize(uint(newW), uint(newH), img, resize.Bilinear)

	// 计算填充偏移
	padX := (targetWidth - newW) / 2
	padY := (targetHeight - newH) / 2

	// 创建目标画布，填充灰色114
	dst := image.NewRGBA(image.Rect(0, 0, targetWidth, targetHeight))
	draw.Draw(dst, dst.Bounds(), &image.Uniform{color.RGBA{114, 114, 114, 255}}, image.Point{}, draw.Src)

	// 把缩放后的图贴到画布中心
	draw.Draw(dst, image.Rect(padX, padY, padX+newW, padY+newH), resized, resized.Bounds().Min, draw.Src)

	return dst, scale, padX, padY
}

// COCO类别
//var yoloClasses = []string{
//	"person", "bicycle", "car", "motorcycle", "airplane", "bus", "train", "truck", "boat",
//	"traffic light", "fire hydrant", "stop sign", "parking meter", "bench", "bird", "cat", "dog",
//	"horse", "sheep", "cow", "elephant", "bear", "zebra", "giraffe", "backpack", "umbrella",
//	"handbag", "tie", "suitcase", "frisbee", "skis", "snowboard", "sports ball", "kite",
//	"baseball bat", "baseball glove", "skateboard", "surfboard", "tennis racket", "bottle",
//	"wine glass", "cup", "fork", "knife", "spoon", "bowl", "banana", "apple", "sandwich",
//	"orange", "broccoli", "carrot", "hot dog", "pizza", "donut", "cake", "chair", "couch",
//	"potted plant", "bed", "dining table", "toilet", "tv", "laptop", "mouse", "remote",
//	"keyboard", "cell phone", "microwave", "oven", "toaster", "sink", "refrigerator", "book",
//	"clock", "vase", "scissors", "teddy bear", "hair drier", "toothbrush",
//}

var yoloClasses = []string{"person", "car"}
