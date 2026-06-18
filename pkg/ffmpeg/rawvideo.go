///**
//  @Time : 2026/01/21 09:43
//  @Author: FangYao( 方少、)
//  @Description: 流处理供上层模型推理用
//  @Email: fy20030315@163.com
//*/
//
//package ffmpeg
//
//import (
//	"context"
//	"errors"
//	"fmt"
//	"go-nvr/pkg/common"
//	"go-nvr/pkg/config"
//	"image"
//	"io"
//	"os/exec"
//	"strconv"
//	"strings"
//)
//
//// ReadVideoStream 读取视频流 RTSP
//func ReadVideoStream(ctx context.Context, videoPath string) (<-chan image.Image, *FFmpeg, error) {
//
//	var width, height, frameSize int
//	var args []string
//
//	// 判断配置：FFmpegResize=true 则用ffmpeg缩放；false 则读取原始分辨率，后面用 nfnt/resize 缩放
//	if config.Conf.Onnx.FFmpegResize {
//		common.Log.Infof("使用ffmpeg缩放")
//		// 从 map 中获取对应模型配置
//		detectConf, ok := config.Conf.Onnx.Models[config.Conf.Onnx.DefaultModel]
//		if !ok {
//			return nil, nil, fmt.Errorf("未找到默认模型配置: %s", config.Conf.Onnx.DefaultModel)
//		}
//		// 从配置读取模型需要的宽高
//		width = int(detectConf.InputShape[3])
//		height = int(detectConf.InputShape[2])
//
//		// 计算一帧数据大小 RGBA = 4 通道
//		frameSize = width * height * 4
//
//		// FFmpeg 命令：自动缩放到模型要求尺寸
//		args = []string{
//			"-rtsp_transport", "tcp",
//			"-fflags", "nobuffer",
//			"-flags", "low_delay",
//			"-i", videoPath,
//			"-vf", fmt.Sprintf("scale=%d:%d", width, height), // 动态缩放
//			"-f", "rawvideo",
//			"-pix_fmt", "rgba",
//			"-an",
//			"-sn",
//			"-loglevel", "error",
//			"-",
//		}
//	} else {
//		common.Log.Infof("使用nfnt/resize 缩放")
//		// 不ffmpeg resize，读取原始尺寸
//		originWidth, originHeight, err := GetVideoSize(videoPath)
//		if err != nil {
//			return nil, nil, err
//		}
//		width = originWidth
//		height = originHeight
//
//		frameSize = width * height * 4
//
//		args = []string{
//			"-rtsp_transport", "tcp",
//			"-fflags", "nobuffer",
//			"-flags", "low_delay",
//			//"-hwaccel", "cuda", // 开启硬件解码（NVIDIA/核显）
//			//"-hwaccel", "nvmpi",  // Jetson 专属硬解
//			"-i", videoPath,
//			"-f", "rawvideo",
//			"-pix_fmt", "rgba",
//			"-an",
//			"-sn",
//			"-loglevel", "error",
//			"-",
//		}
//	}
//
//	proc, err := NewFFmpeg(ctx, args...)
//	if err != nil {
//		return nil, nil, err
//	}
//
//	if err := proc.Start(); err != nil {
//		return nil, nil, err
//	}
//
//	ch := make(chan image.Image, 5)
//
//	go func() {
//
//		defer close(ch)
//
//		buf := make([]byte, frameSize)
//
//		for {
//
//			select {
//			case <-ctx.Done():
//				proc.Stop()
//				return
//			default:
//			}
//
//			n, err := io.ReadFull(proc.Stdout(), buf)
//			if err != nil || n != frameSize {
//				fmt.Printf("[FFmpeg] 读取帧失败: %v, 读取长度: %d\n", err, n)
//				proc.Stop()
//				return
//			}
//			img := GetRGBA(image.Rect(0, 0, width, height))
//			copy(img.Pix, buf)
//
//			select {
//			case ch <- img:
//
//			default:
//				PutRGBA(img)
//			}
//		}
//	}()
//
//	return ch, proc, nil
//}
//
//// getVideoSize 使用 ffprobe 获取视频原始宽高
//func GetVideoSize(videoPath string) (int, int, error) {
//	cmd := exec.Command("ffprobe",
//		"-v", "error",
//		"-select_streams", "v:0",
//		"-show_entries", "stream=width,height",
//		"-of", "csv=p=0:s=x",
//		videoPath,
//	)
//	out, err := cmd.Output()
//	if err != nil {
//		return 0, 0, err
//	}
//
//	// 输出类似 "1920x1080"
//	parts := strings.Split(strings.TrimSpace(string(out)), "x")
//	if len(parts) != 2 {
//		return 0, 0, fmt.Errorf("无法解析视频尺寸: %s", out)
//	}
//
//	width, err := strconv.Atoi(parts[0])
//	if err != nil {
//		return 0, 0, err
//	}
//	height, err := strconv.Atoi(parts[1])
//	if err != nil {
//		return 0, 0, err
//	}
//
//	return width, height, nil
//}
//
//// GetVideoFps 通过ffprobe解析视频流的真实帧率
//func GetVideoFps(videoPath string) (int, error) {
//	// 使用ffprobe获取视频帧率信息
//	cmd := exec.Command(
//		"ffprobe",
//		"-v", "error",
//		"-select_streams", "v:0",
//		"-show_entries", "stream=r_frame_rate",
//		"-of", "csv=p=0",
//		videoPath,
//	)
//	output, err := cmd.Output()
//	if err != nil {
//		return 0, fmt.Errorf("执行ffprobe失败: %v", err)
//	}
//
//	// 解析帧率
//	rateStr := strings.TrimSpace(string(output))
//	parts := strings.Split(rateStr, "/")
//	if len(parts) != 2 {
//		return 0, fmt.Errorf("解析帧率失败，格式异常: %s", rateStr)
//	}
//
//	num, err := strconv.Atoi(parts[0])
//	if err != nil {
//		return 0, fmt.Errorf("解析帧率分子失败: %v", err)
//	}
//	den, err := strconv.Atoi(parts[1])
//	if err != nil {
//		return 0, fmt.Errorf("解析帧率分母失败: %v", err)
//	}
//
//	if den == 0 {
//		return 0, errors.New("帧率分母为0")
//	}
//
//	// 计算整数帧率（四舍五入）
//	fps := int(float64(num)/float64(den) + 0.5)
//	if fps <= 0 {
//		return 0, errors.New("计算出的帧率无效")
//	}
//	return fps, nil
//}

/**
  @Time : 2026/01/21 09:43
  @Author: FangYao( 方少、)
  @Description: 流处理供上层模型推理用
  @Email: fy20030315@163.com
*/

package ffmpeg

import (
	"context"
	"errors"
	"fmt"
	"go-nvr/pkg/common"
	"go-nvr/pkg/config"
	"image"
	"io"
	"os/exec"
	"strconv"
	"strings"
)

// Frame 携带缩放后的图像 + 视频原始宽高（新增）
type Frame struct {
	Image image.Image // FFmpeg缩放后的帧（推理用）
	OrigW int         // 视频原始宽度
	OrigH int         // 视频原始高度
}

// ReadVideoStream 读取视频流 RTSP
func ReadVideoStream(ctx context.Context, videoPath string) (<-chan Frame, *FFmpeg, error) {

	var width, height, frameSize int
	var args []string
	var origW, origH int // 视频原始宽高

	// 先获取视频原始分辨率（必执行）
	originW, originH, err := GetVideoSize(videoPath)
	if err != nil {
		return nil, nil, err
	}
	origW, origH = originW, originH

	// 判断配置：FFmpegResize=true 则用ffmpeg缩放；false 则读取原始分辨率，后面用 nfnt/resize 缩放
	if config.Conf.Onnx.FFmpegResize {
		common.Log.Infof("使用ffmpeg缩放")
		// 从 map 中获取对应模型配置
		detectConf, ok := config.Conf.Onnx.Models[config.Conf.Onnx.DefaultModel]
		if !ok {
			return nil, nil, fmt.Errorf("未找到默认模型配置: %s", config.Conf.Onnx.DefaultModel)
		}
		// 从配置读取模型需要的宽高
		width = int(detectConf.InputShape[3])
		height = int(detectConf.InputShape[2])

		// 计算一帧数据大小 RGBA = 4 通道
		frameSize = width * height * 4

		// FFmpeg 命令：自动缩放到模型要求尺寸
		args = []string{
			"-rtsp_transport", "tcp",
			"-fflags", "nobuffer",
			"-flags", "low_delay",
			"-i", videoPath,
			//"-vf", fmt.Sprintf(
			//	"scale=w=%d:h=%d:force_original_aspect_ratio=decrease,pad=%d:%d:(ow-iw)/2:(oh-ih)/2:color=0x727272",
			//	width, height, width, height),
			"-vf", fmt.Sprintf("scale=%d:%d", width, height), // 动态缩放
			//"-vf", fmt.Sprintf(
			//	"scale=%d:%d:force_original_aspect_ratio=increase,crop=%d:%d",
			//	width, height, width, height,
			//),
			"-f", "rawvideo",
			"-pix_fmt", "rgba",
			"-an",
			"-sn",
			"-loglevel", "error",
			"-",
		}
	} else {
		common.Log.Infof("FFmpeg输出原始分辨率 %dx%d", origW, origH)
		// 不ffmpeg resize，读取原始尺寸
		width = origW
		height = origH

		frameSize = width * height * 4

		args = []string{
			"-rtsp_transport", "tcp",
			"-fflags", "nobuffer",
			"-flags", "low_delay",
			"-probesize", "32", // 减少探测时间
			"-analyzeduration", "0", // 关闭流分析
			// Jetson Orin NX 专属硬解
			//"-hwaccel", "nvmpi",
			//"-hwaccel_output_format", "nv12",
			"-i", videoPath,
			"-f", "rawvideo",
			"-pix_fmt", "rgba",
			"-an",
			"-sn",
			"-loglevel", "error",
			"-",
		}
	}
	//else {
	//	common.Log.Infof("使用nfnt/resize 缩放")
	//	// 不ffmpeg resize，读取原始尺寸
	//	width = origW
	//	height = origH
	//
	//	frameSize = width * height * 4
	//
	//	args = []string{
	//		"-rtsp_transport", "tcp",
	//		"-fflags", "nobuffer",
	//		"-flags", "low_delay",
	//		//"-hwaccel", "cuda", // 开启硬件解码（NVIDIA/核显）
	//		//"-hwaccel", "nvmpi",  // Jetson 专属硬解
	//		"-i", videoPath,
	//		"-f", "rawvideo",
	//		"-pix_fmt", "rgba",
	//		"-an",
	//		"-sn",
	//		"-loglevel", "error",
	//		"-",
	//	}
	//}

	proc, err := NewFFmpeg(ctx, args...)
	if err != nil {
		return nil, nil, err
	}

	if err := proc.Start(); err != nil {
		return nil, nil, err
	}

	ch := make(chan Frame, 5)

	go func() {

		defer close(ch)

		buf := make([]byte, frameSize)

		for {

			select {
			case <-ctx.Done():
				proc.Stop()
				return
			default:
			}

			n, err := io.ReadFull(proc.Stdout(), buf)
			if err != nil || n != frameSize {
				fmt.Printf("[FFmpeg] 读取帧失败: %v, 读取长度: %d\n", err, n)
				proc.Stop()
				return
			}
			img := GetRGBA(image.Rect(0, 0, width, height))
			copy(img.Pix, buf)

			// 封装帧+原始尺寸发送
			select {
			case ch <- Frame{
				Image: img,
				OrigW: origW,
				OrigH: origH,
			}:
			default:
				PutRGBA(img)
			}
		}
	}()

	return ch, proc, nil
}

// getVideoSize 使用 ffprobe 获取视频原始宽高
func GetVideoSize(videoPath string) (int, int, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=width,height",
		"-of", "csv=p=0:s=x",
		videoPath,
	)
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, err
	}

	// 输出类似 "1920x1080"
	parts := strings.Split(strings.TrimSpace(string(out)), "x")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("无法解析视频尺寸: %s", out)
	}

	width, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, err
	}
	height, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, err
	}

	return width, height, nil
}

// GetVideoFps 通过ffprobe解析视频流的真实帧率
func GetVideoFps(videoPath string) (int, error) {
	// 使用ffprobe获取视频帧率信息
	cmd := exec.Command(
		"ffprobe",
		"-v", "error",
		"-select_streams", "v:0",
		"-show_entries", "stream=r_frame_rate",
		"-of", "csv=p=0",
		videoPath,
	)
	output, err := cmd.Output()
	if err != nil {
		return 0, fmt.Errorf("执行ffprobe失败: %v", err)
	}

	// 解析帧率
	rateStr := strings.TrimSpace(string(output))
	parts := strings.Split(rateStr, "/")
	if len(parts) != 2 {
		return 0, fmt.Errorf("解析帧率失败，格式异常: %s", rateStr)
	}

	num, err := strconv.Atoi(parts[0])
	if err != nil {
		return 0, fmt.Errorf("解析帧率分子失败: %v", err)
	}
	den, err := strconv.Atoi(parts[1])
	if err != nil {
		return 0, fmt.Errorf("解析帧率分母失败: %v", err)
	}

	if den == 0 {
		return 0, errors.New("帧率分母为0")
	}

	// 计算整数帧率（四舍五入）
	fps := int(float64(num)/float64(den) + 0.5)
	if fps <= 0 {
		return 0, errors.New("计算出的帧率无效")
	}
	return fps, nil
}
