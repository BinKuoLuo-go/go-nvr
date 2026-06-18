/**
@Time : 2026/01/16 14:45
@Author: FangYao( 方少、)
@Description: ffmpeg基础封装
@Email: fy20030315@163.com
*/

package ffmpeg

import (
	"context"
	"io"
	"os/exec"
	"runtime"
	"sync"
	"syscall"
)

type FFmpeg struct {
	cmd    *exec.Cmd          // ffmpeg 进程实例
	ctx    context.Context    // 上下文
	cancel context.CancelFunc // 用于主动取消 ctx
	stdin  io.WriteCloser     // 输入管道
	stdout io.ReadCloser      // 输出管道
	stderr io.ReadCloser      // 错误输出
	once   sync.Once          // 用于 Stop / Kill / cancel 逻辑只执行一次
	done   chan struct{}      // 进程完全退出后关闭
}

// NewFFmpeg
func NewFFmpeg(ctx context.Context, args ...string) (*FFmpeg, error) {
	cctx, cancel := context.WithCancel(ctx)
	cmd := exec.CommandContext(cctx, "ffmpeg", args...)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		cancel()
		return nil, err
	}
	return &FFmpeg{
		cmd:    cmd,
		ctx:    cctx,
		cancel: cancel,
		stdin:  stdin,
		stdout: stdout,
		stderr: stderr,
		done:   make(chan struct{}),
	}, nil
}

// Start 启动ffmpeg
func (f *FFmpeg) Start() error {
	if err := f.cmd.Start(); err != nil {
		return err
	}
	go func() {
		defer close(f.done)
		_ = f.cmd.Wait()
	}()
	return nil
}

// Stdin 返回 stdin pipe
func (f *FFmpeg) Stdin() io.WriteCloser {
	return f.stdin
}

// Stdout 返回 stdout pipe
func (f *FFmpeg) Stdout() io.ReadCloser {
	return f.stdout
}

// Stderr 返回 stderr 管道（报错用）
func (f *FFmpeg) Stderr() io.ReadCloser {
	return f.stderr
}

// Stop 停止
func (f *FFmpeg) Stop() {
	f.once.Do(func() {
		f.cancel()
		if f.cmd.Process != nil {
			if runtime.GOOS == "windows" {
				// Windows下直接Kill
				_ = f.cmd.Process.Kill()
			} else {
				_ = f.cmd.Process.Signal(syscall.SIGTERM)
			}
		}
	})
}

// Kill 杀死进程
func (f *FFmpeg) Kill() {
	f.once.Do(func() {
		f.cancel()
		if f.cmd.Process != nil {
			_ = f.cmd.Process.Kill()
		}
	})
}

// Wait 等待退出
func (f *FFmpeg) Wait() {
	<-f.done
}
