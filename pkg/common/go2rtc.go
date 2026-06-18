/**
@Time : 2026/01/15 12:30
@Author: FangYao( 方少)
@Description: 初始化go2rtc服务
@Email: fy20030315@163.com
*/

package common

import (
	"context"
	"go-nvr/pkg/config"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"syscall"
	"time"
)

// 全局go2rtc进程上下文
var Go2RTCContext context.Context
var Go2RTCCancel context.CancelFunc
var go2rtcCmd *exec.Cmd

// InitGo2RTC 初始化并启动go2rtc服务
func InitGo2RTC() {
	// 如果已有进程，先停止
	if go2rtcCmd != nil && go2rtcCmd.Process != nil {
		StopGo2RTC()
	}
	Go2RTCContext, Go2RTCCancel = context.WithCancel(context.Background())
	// 自动获取当前系统对应的go2rtc可执行文件路径
	go2rtcPath := getGo2rtcCmdPath()
	cmd := exec.CommandContext(
		Go2RTCContext,
		go2rtcPath,
		"-config", ".\\go2rtc.yaml",
	)

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	Log.Infof("启动 go2rtc...")
	if err := cmd.Start(); err != nil {
		Log.Fatalf("无法启动 go2rtc 服务: %v", err)
	}

	// 监听系统信号，退出
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
		Log.Infof("信号已接收，正在关闭go2rtc...")
		StopGo2RTC()
	}()
}

// StopGo2RTC 安全关闭go2rtc：取消上下文 + 等待退出 + 超时强制Kill
func StopGo2RTC() {
	if Go2RTCCancel != nil {
		Go2RTCCancel()
	}

	// 无进程直接返回
	if go2rtcCmd == nil || go2rtcCmd.Process == nil {
		Log.Infof("go2rtc 未运行")
		return
	}

	// 等待进程退出
	done := make(chan error, 1)
	go func() {
		done <- go2rtcCmd.Wait()
	}()

	select {
	case <-done:
		Log.Infof("go2rtc 正常退出")
	case <-time.After(3 * time.Second):
		// 超时强制杀死
		_ = go2rtcCmd.Process.Kill()
		Log.Infof("go2rtc 超时未退出，已强制杀死")
	}

	// 重置全局变量
	go2rtcCmd = nil
	Go2RTCCancel = nil
	Go2RTCContext = nil
}

// getGo2rtcCmdPath 根据当前系统自动加载对应可执行文件
func getGo2rtcCmdPath() string {
	var go2rtcBinDir string
	if config.Conf.System.DevMode {
		_, currentFile, _, ok := runtime.Caller(0)
		if !ok {
			panic("无法获取当前文件路径，加载go2rtc可执行文件失败")
		}
		// 从当前文件路径向上定位到项目根目录
		commonDir := filepath.Dir(currentFile)
		pkgDir := filepath.Dir(commonDir)
		// 拼接go2rtc的固定目录：pkg/bin/go2rtc
		go2rtcBinDir = filepath.Join(pkgDir, "bin", "go2rtc")
	} else {
		// 生产模式：程序所在目录
		exePath, err := os.Executable()
		if err != nil {
			panic(err)
		}
		exeDir := filepath.Dir(exePath)
		go2rtcBinDir = filepath.Join(exeDir, "pkg", "bin", "go2rtc")
	}

	// 根据系统+架构自动匹配对应的可执行文件
	var go2rtcFileName string

	// windows系统
	if runtime.GOOS == "windows" {
		if runtime.GOARCH == "amd64" {
			go2rtcFileName = "go2rtc.exe"
		}
	}
	// linux系统
	if runtime.GOOS == "linux" {
		if runtime.GOARCH == "amd64" {
			go2rtcFileName = "go2rtc_linux_amd64"
		}
		if runtime.GOARCH == "arm64" {
			go2rtcFileName = "go2rtc_linux_arm64"
		}
	}

	// 未匹配到对应系统/架构的可执行文件
	if go2rtcFileName == "" {
		panic("无法找到适配当前系统的go2rtc版本，系统：" + runtime.GOOS + "，架构：" + runtime.GOARCH)
	}

	// 拼接最终的go2rtc可执行文件绝对路径
	return filepath.Join(go2rtcBinDir, go2rtcFileName)
}
