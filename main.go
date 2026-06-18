/**
@Time : 2026/01/15 11:44
@Author: FangYao( 方少、)
@Description: 入口文件
@Email: fy20030315@163.com
*/

package main

import (
	"context"
	"fmt"
	"go-nvr/cmd/routes"
	"go-nvr/pkg/common"
	"go-nvr/pkg/config"
	"go-nvr/pkg/grpc"
	"go-nvr/pkg/license"
	"go-nvr/pkg/plc"
	"go-nvr/pkg/task"
	"log"
	"net/http"
	_ "net/http/pprof"
	"os"
	"os/signal"
	"time"
)

func main() {
	//defer func() {
	//	if err := recover(); err != nil {
	//		fmt.Println("出现错误:", err)
	//	}
	//}()

	// 加载配置文件到全局配置结构体
	config.InitConfig()

	// 开启 pprof 性能分析（调试模式开启，生产环境根据配置关闭)
	if config.Conf.System.Mode == "debug" {
		go func() {
			addr := "127.0.0.1:6060"
			common.Log.Infof("pprof 调试服务启动: http://%s", addr)
			if err := http.ListenAndServe(addr, nil); err != nil {
				common.Log.Warnf("pprof 服务退出: %v", err)
			}
		}()
	}

	// 初始化日志
	common.InitLogger()

	if err := license.Verify(); err != nil {
		common.Log.Fatalf("授权验证失败，程序退出：%v", err)
	}

	// 初始化数据库
	common.InitDB()

	// 初始化定时任务
	task.InitCronTask()

	// 初始化go2rtc
	common.InitGo2RTC()

	// 初始化 mediamtx
	if config.Conf.System.EnableRecording {
		common.InitMediaMtx()
	} else {
		common.Log.Infof("录制功能未启用,不初始化MediaMtx")
	}

	//  初始化ONNX模型
	//if err := common.InitOnnxModels(); err != nil {
	//	log.Fatalf("初始化ONNX模型失败: %v", err)
	//}
	//defer common.DestroyAllOnnxModels() // 程序退出时销毁所有模型
	//
	//if config.Conf.Onnx.InferEngine == "tensorrt" {
	//	grpc.InitGrpcClient()
	//	defer grpc.Close()
	//}
	if config.Conf.Onnx.InferEngine == "tensorrt" {
		common.Log.Info("当前推理引擎: TensorRT(gRPC)，跳过 ONNX 模型加载")
		// 初始化 gRPC 客户端
		grpc.InitGrpcClient()
		defer grpc.Close()
	} else {
		common.Log.Info("当前推理引擎: ONNX 本地推理")
		// 初始化 ONNX 模型
		if err := common.InitOnnxModels(); err != nil {
			log.Fatalf("初始化ONNX模型失败: %v", err)
		}
		defer common.DestroyAllOnnxModels()
	}

	// 启动PLC控制器
	if config.Conf.System.IsStartOPlc {
		common.Log.Info("PLC配置已启用，开始初始化PLC控制器")
		plc.StartPLC()
	} else {
		common.Log.Info("PLC配置已禁用，跳过PLC初始化")
	}

	// 注册所有路由
	r := routes.InitRoutes()

	host := config.Conf.System.Host
	port := config.Conf.System.Port

	// 配置服务
	srv := &http.Server{
		Addr:    fmt.Sprintf("%s:%d", host, port),
		Handler: r,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			common.Log.Fatalf("监听: %s\n", err)
		}
	}()
	common.Log.Info(fmt.Sprintf("服务运行在 http://%s:%d", host, port))

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit
	common.Log.Info("正在关闭服务...")

	// 停止PLC
	if config.Conf.System.IsStartOPlc {
		plc.StopPLC()
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := srv.Shutdown(ctx); err != nil {
		common.Log.Fatal("服务被迫关闭,错误:", err)
	}

	common.Log.Info("服务退出成功!")
}
