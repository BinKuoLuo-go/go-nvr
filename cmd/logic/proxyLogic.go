/**
@Time : 2026/01/19 11:05
@Author: FangYao( 方少、)
@Description: go2rtc api反向代理
@Email: fy20030315@163.com
*/

package logic

import (
	"github.com/gin-gonic/gin"
	"io"
	"net/http"
)

type ProxyLogic struct{}

// WebRTCProxy rtsp转webrtc
func (pl *ProxyLogic) WebRTCProxy(c *gin.Context) {
	target := go2rtcBase + "/api/webrtc"

	// 构建代理请求
	proxyReq, err := http.NewRequest(
		c.Request.Method,
		target+"?"+c.Request.URL.RawQuery,
		c.Request.Body,
	)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	// 复制必要请求头
	proxyReq.Header.Set("Content-Type", "application/json")

	// 发起请求
	resp, err := http.DefaultClient.Do(proxyReq)
	if err != nil {
		c.Status(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	// 复制响应头
	for k, v := range resp.Header {
		for _, vv := range v {
			c.Writer.Header().Add(k, vv)
		}
	}

	// 设置状态码
	c.Status(resp.StatusCode)

	// 透传body
	_, _ = io.Copy(c.Writer, resp.Body)
}

// LogsProxy 日志代理
func (pl *ProxyLogic) LogsProxy(c *gin.Context) {

	go2rtcLogURL := go2rtcBase + "/api/log" // go2rtc 日志地址

	switch c.Request.Method {
	case http.MethodGet:
		// GET 请求，获取日志
		resp, err := http.Get(go2rtcLogURL)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer resp.Body.Close()

		body, err := io.ReadAll(resp.Body)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		// 返回原始日志文本
		c.Data(resp.StatusCode, "application/json", body)

	case http.MethodDelete:
		// ，清空日志
		req, err := http.NewRequest(http.MethodDelete, go2rtcLogURL, nil)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}

		client := &http.Client{}
		resp, err := client.Do(req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		c.Data(resp.StatusCode, "text/plain", body)

	default:
		c.JSON(http.StatusMethodNotAllowed, gin.H{"error": "method not allowed"})
	}
}

// StreamsDotProxy go2rtc streams.dot 代理
func (pl *ProxyLogic) StreamsDotProxy(c *gin.Context) {
	target := go2rtcBase + "/api/streams.dot"
	if c.Request.URL.RawQuery != "" {
		target += "?" + c.Request.URL.RawQuery
	}

	req, err := http.NewRequest(http.MethodGet, target, nil)
	if err != nil {
		c.Status(http.StatusInternalServerError)
		return
	}

	// 文本
	req.Header.Set("Accept", "text/plain")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.Status(http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()

	c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.Status(resp.StatusCode)

	_, _ = io.Copy(c.Writer, resp.Body)
}

// StreamsProxy streams 流相关代理
func (pl *ProxyLogic) StreamsProxy(c *gin.Context) {
	SimpleProxy(c, "/api/streams")
}

// InfoProxy go2rtc 基本信息代理
func (pl *ProxyLogic) InfoProxy(c *gin.Context) {
	SimpleProxy(c, "/api")
}

// ConfigProxy go2rtc 配置文件代理
func (pl *ProxyLogic) ConfigProxy(c *gin.Context) {
	SimpleProxy(c, "/api/config")
}

// DeviceDiscoveryProxy 设备发现代理
func (pl *ProxyLogic) DeviceDiscoveryProxy(c *gin.Context) {
	// 根据请求路径判断要代理go2rtc api
	var path string
	switch c.Query("service") {
	case "ffmpeg/devices":
		path = "/api/ffmpeg/devices"
	case "ffmpeg/hardware":
		path = "/api/ffmpeg/hardware"
	case "dvrip":
		path = "/api/dvrip"
	case "onvif":
		path = "/api/onvif"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown device discovery service"})
		return
	}

	SimpleProxy(c, path)
}

// ControlProxy go2rtc的 exit / restart
func (pl *ProxyLogic) ControlProxy(c *gin.Context) {
	var path string
	switch c.Query("action") {
	case "exit":
		path = "/api/exit"
	case "restart":
		path = "/api/restart"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown action"})
		return
	}
	SimpleProxy(c, path)
}

// StreamProxy 代理各种流接口（MP4/HLS/MJPEG/FLV等）
func (pl *ProxyLogic) StreamProxy(c *gin.Context) {
	var path string
	switch c.Request.URL.Path {
	case "/proxy/stream.mp4":
		path = "/api/stream.mp4"
	case "/proxy/stream.m3u8":
		path = "/api/stream.m3u8"
	case "/proxy/hls/playlist.m3u8":
		path = "/api/hls/playlist.m3u8"
	case "/proxy/hls/segment.ts":
		path = "/api/hls/segment.ts"
	case "/proxy/hls/init.mp4":
		path = "/api/hls/init.mp4"
	case "/proxy/hls/segment.m4s":
		path = "/api/hls/segment.m4s"
	case "/proxy/stream.mjpeg":
		path = "/api/stream.mjpeg"
	case "/proxy/stream.ascii":
		path = "/api/stream.ascii"
	case "/proxy/stream.y4m":
		path = "/api/stream.y4m"
	case "/proxy/stream.ts":
		path = "/api/stream.ts"
	case "/proxy/stream.aac":
		path = "/api/stream.aac"
	case "/proxy/stream.flv":
		path = "/api/stream.flv"
	case "/proxy/frame.jpeg":
		path = "/api/frame.jpeg"
	case "/proxy/frame.mp4":
		path = "/api/frame.mp4"
	default:
		c.JSON(http.StatusBadRequest, gin.H{"error": "unknown stream endpoint"})
		return
	}
	SimpleProxy(c, path)
}
