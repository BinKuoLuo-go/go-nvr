/*
*
@Time : 2026/01/20 15:03
@Author: FangYao( 方少、)
@Description: 告警录像相关方法
@Email: fy20030315@163.com
*/
package utils

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"go-nvr/pkg/config"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	mediaMTXBaseURL   = "http://localhost:9997/v3"
	retryMaxAttempts  = 3                      // HTTP请求重试次数
	retryInterval     = 500 * time.Millisecond // 重试间隔
	httpClientTimeout = 5 * time.Second        // HTTP超时时间
	minRecordDuration = 10000                  // 最小录制时长10秒，防止碎片化
)

type RecordingStatus struct {
	IsRecording     bool
	LastDetectTime  int64
	StartRecordTime int64 // 录像开始时间
	Mu              sync.Mutex
}

var (
	RecordingStatusMap sync.Map
	pathInitMap        sync.Map // MediaMTX路径单例初始化
)

func GetRecordingStatus(src string) *RecordingStatus {
	val, _ := RecordingStatusMap.LoadOrStore(src, &RecordingStatus{})
	return val.(*RecordingStatus)
}

// Go2RTC RTSP 地址
func GetGo2RTCStreamURL(src string) string {
	return fmt.Sprintf("rtsp://localhost:8554/%s", src)
}

// MediaMTX Path 单例初始化
func EnsureMediaMTXPath(src string) error {
	encoded := url.PathEscape(src)

	if _, loaded := pathInitMap.LoadOrStore(encoded, true); loaded {
		return nil // 已初始化
	}

	recPath := filepath.Join(config.Conf.System.RecordingRootPath, src)
	if err := os.MkdirAll(recPath, 0755); err != nil {
		pathInitMap.Delete(encoded)
		return err
	}

	recordPathPattern := filepath.Join(config.Conf.System.RecordingRootPath, "%path", "%Y-%m-%d_%H-%M-%S-%f")
	cfg := map[string]interface{}{
		"source":             GetGo2RTCStreamURL(src),
		"sourceOnDemand":     false,
		"record":             false,
		"recordPath":         recordPathPattern,
		"recordFormat":       "fmp4",
		"recordPartDuration": "30s",
		"recordDeleteAfter":  "72h",
		"maxReaders":         10,
	}

	return patchPath(encoded, cfg)
}

// StartRecording 开始录像
func StartRecording(src string) error {
	status := GetRecordingStatus(src)
	status.Mu.Lock()
	defer status.Mu.Unlock()
	status.StartRecordTime = time.Now().UnixMilli() // 记录开始时间

	if err := patchPath(url.PathEscape(src), map[string]interface{}{"record": true}); err != nil {
		return err
	}

	status.IsRecording = true
	log.Printf("[%s] 录像启动", src)
	return nil
}

// StopRecording 停止录像
func StopRecording(src string) error {
	if err := patchPath(url.PathEscape(src), map[string]interface{}{"record": false}); err != nil {
		return err
	}

	status := GetRecordingStatus(src)
	status.Mu.Lock()
	defer status.Mu.Unlock()
	status.IsRecording = false
	log.Printf("[%s] 录像停止", src)
	return nil
}

// PATCH 请求 + 重试机制
func patchPath(encodedSrc string, cfg map[string]interface{}) error {
	data, err := json.Marshal(cfg)
	if err != nil {
		return err
	}

	reqURL := fmt.Sprintf("%s/config/paths/patch/%s", mediaMTXBaseURL, encodedSrc)
	client := &http.Client{Timeout: httpClientTimeout}

	var lastErr error
	for i := 0; i < retryMaxAttempts; i++ {
		req, err := http.NewRequest("PATCH", reqURL, bytes.NewBuffer(data))
		if err != nil {
			lastErr = err
			time.Sleep(retryInterval)
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			time.Sleep(retryInterval)
			continue
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		if resp.StatusCode == 200 {
			return nil
		}
		lastErr = fmt.Errorf("MediaMTX PATCH失败: %s %s", resp.Status, body)
		time.Sleep(retryInterval)
	}

	return lastErr
}

// 录像超时自动停止（带最小录像时长）
func StartRecordingTimeoutCheck(ctx context.Context, src string, timeoutMs int64) {
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			status := GetRecordingStatus(src)
			status.Mu.Lock()

			if !status.IsRecording {
				status.Mu.Unlock()
				continue
			}

			now := time.Now().UnixMilli()
			isTimeout := now-status.LastDetectTime > timeoutMs
			recordDuration := now - status.StartRecordTime
			canStop := recordDuration >= minRecordDuration

			if isTimeout && canStop {
				if err := StopRecording(src); err != nil {
					log.Printf("流[%s]停止录制失败: %v", src, err)
				} else {
					log.Printf("流[%s]超时无目标，停止录制 | 时长:%.2fs", src, float64(recordDuration)/1000)
				}
			}

			status.Mu.Unlock()
		}
	}
}
