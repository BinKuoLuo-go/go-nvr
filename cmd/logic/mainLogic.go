/**
@Time : 2026/01/16 09:16
@Author: FangYao( 方少、)
@Description:业务逻辑层
@Email: fy20030315@163.com
*/

package logic

import (
	"bytes"
	"fmt"
	"github.com/gin-gonic/gin"
	"go-nvr/pkg/plugins"
	"io"
	"net/http"
	"strings"
)

// 业务逻辑层的统一入口，集中处理基础业务逻辑
var (
	Device = &DeviceLogic{}
	Proxy  = &ProxyLogic{}
)

// 统一错误
var (
	ReqAssertErr = plugins.NewRepError(plugins.SystemErr, fmt.Errorf("请求异常"))
)

const go2rtcBase = "http://localhost:1984"

// SimpleProxy 通用代理方法
func SimpleProxy(c *gin.Context, path string) {
	target := go2rtcBase + path
	if c.Request.URL.RawQuery != "" {
		if strings.Contains(path, "?") {
			target += "&" + c.Request.URL.RawQuery
		} else {
			target += "?" + c.Request.URL.RawQuery
		}
	}

	var body io.Reader
	if c.Request.Body != nil {
		data, _ := io.ReadAll(c.Request.Body)
		body = bytes.NewReader(data)
	}

	req, err := http.NewRequest(c.Request.Method, target, body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 复制请求头并去除重复
	for k, v := range c.Request.Header {
		for _, vv := range v {
			// 如果已经存在该请求头，使用 Set 来覆盖
			if req.Header.Get(k) == "" {
				req.Header.Add(k, vv) // 如果没有重复，使用 Add
			} else {
				req.Header.Set(k, vv) // 如果有重复，使用 Set 覆盖
			}
		}
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	// 复制响应头并去除重复
	for k, v := range resp.Header {
		for _, vv := range v {
			// 如果已经存在该响应头，使用 Set 来覆盖
			if c.Writer.Header().Get(k) == "" {
				c.Writer.Header().Add(k, vv) // 如果没有重复，使用 Add
			} else {
				c.Writer.Header().Set(k, vv) // 如果有重复，使用 Set 覆盖
			}
		}
	}

	c.Status(resp.StatusCode)
	_, _ = io.Copy(c.Writer, resp.Body)
}
