/*
*
@Time : 2025/12/14 18:40
@Author: FangYao( 方少、)
@Description:  初始化onnxruntime_go服务
@Email: fy20030315@163.com
*/

package common

import (
	"fmt"
	ort "github.com/yalue/onnxruntime_go"
	"go-nvr/pkg/config"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"time"
)

// 全局环境
var (
	envOnce   sync.Once
	envInited bool
)

// Session结构
type ModelSession struct {
	Session *ort.AdvancedSession
	Input   *ort.Tensor[float32]
	Output  *ort.Tensor[float32]
}

type SessionPool struct {
	ModelType string
	Pool      chan *ModelSession
}

// 多模型
var (
	OnnxModels = make(map[string]*SessionPool)
	ModelsMu   sync.RWMutex
)

// 动态库路径
func getSharedLibPath() (string, error) {
	var libDir string

	// 根据配置切换路径
	if config.Conf.System.DevMode {
		// 开发模式：源码路径
		_, currentFile, _, ok := runtime.Caller(0)
		if !ok {
			return "", fmt.Errorf("无法获取当前文件路径")
		}
		commonDir := filepath.Dir(currentFile)
		pkgDir := filepath.Dir(commonDir)

		if config.Conf.Onnx.UseCuda {
			libDir = filepath.Join(pkgDir, "bin", "onnxruntime_go", "third_party_gpu")
		} else {
			libDir = filepath.Join(pkgDir, "bin", "onnxruntime_go", "third_party_cpu")
		}
	} else {
		// 生产模式：程序执行目录
		exePath, err := os.Executable()
		if err != nil {
			return "", err
		}
		exeDir := filepath.Dir(exePath)
		if config.Conf.Onnx.UseCuda {
			libDir = filepath.Join(exeDir, "pkg", "bin", "onnxruntime_go", "third_party_gpu")
		} else {
			libDir = filepath.Join(exeDir, "pkg", "bin", "onnxruntime_go", "third_party_cpu")
		}
	}
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(libDir, "onnxruntime.dll"), nil
	case "darwin":
		if runtime.GOARCH == "arm64" {
			return filepath.Join(libDir, "onnxruntime_arm64.dylib"), nil
		}
		return filepath.Join(libDir, "onnxruntime_amd64.dylib"), nil
	case "linux":
		if runtime.GOARCH == "arm64" {
			return filepath.Join(libDir, "onnxruntime_arm64.so"), nil
		}
		return filepath.Join(libDir, "onnxruntime.so"), nil
	default:
		return "", fmt.Errorf("不支持的系统: %s", runtime.GOOS)
	}
}

// initOrtEnv 初始化 ORT
func initOrtEnv() error {
	var initErr error

	envOnce.Do(func() {
		libPath, err := getSharedLibPath()
		if err != nil {
			initErr = err
			return
		}

		ort.SetSharedLibraryPath(libPath)

		if err = ort.InitializeEnvironment(); err != nil {
			initErr = err
			return
		}

		envInited = true
		Log.Infof("ONNX Runtime 初始化成功")
	})

	return initErr
}

// createSession 创建 Session
func createSession(modelType string) (*ModelSession, error) {
	baseConf := config.Conf.Onnx.Base

	detectConf, err := GetDetectConfig(modelType)
	if err != nil {
		return nil, err
	}

	input, err := ort.NewEmptyTensor[float32](ort.NewShape(detectConf.InputShape...))
	if err != nil {
		return nil, err
	}

	output, err := ort.NewEmptyTensor[float32](ort.NewShape(detectConf.OutputShape...))
	if err != nil {
		input.Destroy()
		return nil, err
	}

	opts, err := ort.NewSessionOptions()
	if err != nil {
		input.Destroy()
		output.Destroy()
		return nil, err
	}

	// 图优化
	_ = opts.SetGraphOptimizationLevel(ort.GraphOptimizationLevelEnableAll)

	// CPU优化
	if !config.Conf.Onnx.UseCuda {
		_ = opts.SetIntraOpNumThreads(2)
		_ = opts.SetInterOpNumThreads(1)
		_ = opts.SetCpuMemArena(true)
		_ = opts.SetMemPattern(true)
	}

	// GPU（失败自动降级）
	if config.Conf.Onnx.UseCuda {
		cuda, err := ort.NewCUDAProviderOptions()
		if err != nil {
			Log.Warnf("CUDA创建失败，回退CPU: %v", err)
		} else {
			if err = opts.AppendExecutionProviderCUDA(cuda); err != nil {
				Log.Warnf("CUDA附加失败，回退CPU: %v", err)
			} else {
				gpus, err := GetNvidiaGPUs()
				if err != nil {
					Log.Infof("CUDA 启用成功 | 无法获取显卡信息: %v", err)
				} else {
					for _, gpu := range gpus {
						Log.Infof("CUDA 设备启用 | ID: %d | 型号: %s | 总显存: %s | 可用: %s",
							gpu.ID, gpu.Name, gpu.MemoryTotal, gpu.MemoryFree)
					}
				}
			}
			cuda.Destroy()
		}
	}

	session, err := ort.NewAdvancedSession(
		detectConf.ModelPath,
		[]string{baseConf.InputName},
		[]string{baseConf.OutputName},
		[]ort.ArbitraryTensor{input},
		[]ort.ArbitraryTensor{output},
		opts,
	)

	// 释放
	opts.Destroy()

	if err != nil {
		input.Destroy()
		output.Destroy()
		return nil, err
	}

	return &ModelSession{
		Session: session,
		Input:   input,
		Output:  output,
	}, nil
}

// InitOnnxModels 初始化模型池
func InitOnnxModels() error {
	if config.Conf.Onnx == nil {
		return fmt.Errorf("onnx 配置为空")
	}

	if err := initOrtEnv(); err != nil {
		return err
	}

	modelType := config.Conf.Onnx.DefaultModel
	if modelType == "" {
		return fmt.Errorf("modelType 未配置")
	}

	var concurrency int
	if config.Conf.Onnx.UseCuda {
		concurrency = config.Conf.Onnx.MaxConcurrency // gpu模式的并发数 最大并发推理数（Session数量）
	} else {
		concurrency = runtime.NumCPU() // cpu模式自动获取cpu核心数
	}
	pool := &SessionPool{
		ModelType: modelType,
		Pool:      make(chan *ModelSession, concurrency),
	}

	for i := 0; i < concurrency; i++ {
		s, err := createSession(modelType)
		if err != nil {
			return err
		}
		pool.Pool <- s
	}

	ModelsMu.Lock()
	OnnxModels[modelType] = pool
	ModelsMu.Unlock()

	Log.Infof("模型[%s] 初始化完成，池大小=%d", modelType, concurrency)
	return nil
}

// AcquireSession 获取 Session
func AcquireSession(modelType string) (*ModelSession, error) {
	ModelsMu.RLock()
	pool, ok := OnnxModels[modelType]
	ModelsMu.RUnlock()

	if !ok || pool == nil {
		return nil, fmt.Errorf("模型[%s]未初始化", modelType)
	}

	// 从池子里拿session
	select {
	case s := <-pool.Pool:
		return s, nil
	case <-time.After(500 * time.Millisecond):
		return nil, fmt.Errorf("模型[%s]繁忙", modelType)
	}
}

// ReleaseSession 归还 Session
func ReleaseSession(modelType string, s *ModelSession) {
	if s == nil {
		return
	}

	ModelsMu.RLock()
	defer ModelsMu.RUnlock()

	if pool, ok := OnnxModels[modelType]; ok && pool != nil {
		select {
		case pool.Pool <- s:
		default:
			// 防止阻塞
			go func() { pool.Pool <- s }()
		}
	}
}

// DestroyAllOnnxModels 销毁
func DestroyAllOnnxModels() {
	ModelsMu.Lock()
	defer ModelsMu.Unlock()

	if !envInited {
		return
	}

	for _, pool := range OnnxModels {
		close(pool.Pool)
		for s := range pool.Pool {
			s.Destroy()
		}
	}

	OnnxModels = make(map[string]*SessionPool)

	ort.DestroyEnvironment()
	envInited = false

	Log.Infof("ONNX 资源已释放")
}

// Destroy Session销毁
func (m *ModelSession) Destroy() {
	if m == nil {
		return
	}
	_ = m.Session.Destroy()
	_ = m.Input.Destroy()
	_ = m.Output.Destroy()
}

// GetDetectConfig 配置
func GetDetectConfig(modelType string) (*config.DetectConfig, error) {
	models := config.Conf.Onnx.Models
	if models == nil {
		return nil, fmt.Errorf("models 配置为空")
	}

	cfg, ok := models[modelType]
	if !ok {
		return nil, fmt.Errorf("模型[%s]不存在", modelType)
	}

	return cfg, nil
}
