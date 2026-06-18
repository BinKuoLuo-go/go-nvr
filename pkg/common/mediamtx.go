/**
@Time : 2026/01/30 16:06
@Author: FangYao( 方少、)
@Description: 初始化mediamtx服务 主要是用于录制，go2rtc不支持录制
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

// 全局Mediamtx进程上下文
var MediaMtxContext context.Context
var MediaMtxCancel context.CancelFunc
var mediaMtxCmd *exec.Cmd

// InitMediaMtx 初始化并启动Mediamtx服务
func InitMediaMtx() {

	// 如果已有进程，先停止
	if mediaMtxCmd != nil && mediaMtxCmd.Process != nil {
		StopMediaMtx()
	}
	MediaMtxContext, MediaMtxCancel = context.WithCancel(context.Background())
	// 自动获取当前系统对应的go2rtc可执行文件路径
	mediaMtxPath := getMediaMtxCmdPath()
	cmd := exec.CommandContext(
		MediaMtxContext,
		mediaMtxPath,
		".\\mediamtx.yml",
	)
	mediaMtxCmd = cmd

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	Log.Infof("启动 Mediamtx...")
	if err := cmd.Start(); err != nil {
		Log.Fatalf("无法启动 Mediamtx 服务: %v", err)
	}

	// 监听系统信号，退出
	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt, syscall.SIGTERM)
		<-sig
		Log.Infof("信号已接收，正在关闭Mediamtx...")
		StopMediaMtx()
	}()
}

// StopMediaMtx 安全关闭mediamtx：取消上下文 + 等待退出 + 超时强制Kill
func StopMediaMtx() {
	if MediaMtxCancel != nil {
		MediaMtxCancel()
	}

	// 无进程直接返回
	if mediaMtxCmd == nil || mediaMtxCmd.Process == nil {
		Log.Infof("Mediamtx 未运行")
		return
	}

	// 等待进程退出
	done := make(chan error, 1)
	go func() {
		done <- mediaMtxCmd.Wait()
	}()

	select {
	case <-done:
		Log.Infof("Mediamtx 正常退出")
	case <-time.After(3 * time.Second):
		// 超时未退出，强制杀死进程
		_ = mediaMtxCmd.Process.Kill()
		Log.Infof("Mediamtx 超时未退出，已强制杀死")
	}

	// 重置全局变量
	mediaMtxCmd = nil
	MediaMtxCancel = nil
	MediaMtxContext = nil
}

// getMediaMtxCmdPath 根据当前系统自动加载对应可执行文件
func getMediaMtxCmdPath() string {

	var mediaMtxBinDir string
	// 根据配置切换路径
	if config.Conf.System.DevMode {
		_, currentFile, _, ok := runtime.Caller(0)
		if !ok {
			panic("无法获取当前文件路径，加载mediamtx可执行文件失败")
		}
		// 从当前文件路径向上定位到项目根目录
		commonDir := filepath.Dir(currentFile)
		pkgDir := filepath.Dir(commonDir)
		// 拼接mediamtx的固定目录：pkg/bin/mediamtx
		mediaMtxBinDir = filepath.Join(pkgDir, "bin", "mediamtx")

	} else {
		// 生产模式：程序执行目录
		exePath, err := os.Executable()
		if err != nil {
			panic(err)
		}
		exeDir := filepath.Dir(exePath)
		mediaMtxBinDir = filepath.Join(exeDir, "pkg", "bin", "mediamtx")
	}

	// 根据系统+架构自动匹配对应的可执行文件
	var mediaMtxFileName string

	// windows系统
	if runtime.GOOS == "windows" {
		if runtime.GOARCH == "amd64" {
			mediaMtxFileName = "mediamtx.exe"
		}
	}
	// linux系统
	if runtime.GOOS == "linux" {
		if runtime.GOARCH == "amd64" {
			mediaMtxFileName = "mediamtx_linux_amd64"
		}
	}

	// 未匹配到对应系统/架构的可执行文件
	if mediaMtxFileName == "" {
		panic("无法找到适配当前系统的mediamtx版本，系统：" + runtime.GOOS + "，架构：" + runtime.GOARCH)
	}

	// 拼接最终的mediamtx可执行文件绝对路径
	return filepath.Join(mediaMtxBinDir, mediaMtxFileName)
}
