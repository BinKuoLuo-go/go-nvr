/**
@Time : 2026/02/04 15:32
@Author: FangYao( 方少、)
@Description:
@Email: fy20030315@163.com
*/

package onvif

import (
	"context"
	"github.com/use-go/onvif"
	"log"
	"testing"
)

var (
	deviceXaddr = "192.168.1.4:80" // 设备的IP 和端口
	username    = "admin"          // 设备的 ONVIF 用户名
	password    = "1234abcd"       // 设备的 ONVIF 密码
)

func TestOnvif(t *testing.T) {
	// 创建设备参数
	params := onvif.DeviceParams{
		Xaddr:    deviceXaddr,
		Username: username,
		Password: password,
	}

	// 建立连接
	dev, err := onvif.NewDevice(params)
	if err != nil {
		log.Fatalf("创建设备连接失败: %v", err)
	}

	ctx := context.Background()

	// 获取设备基本信息
	GetDeviceInformation(ctx, dev)

	// 获取 RTSP 流地址
	GetRTSPUri(ctx, dev, password, username)

	// 获取ptz
	GetPTZService(ctx, dev)
}
