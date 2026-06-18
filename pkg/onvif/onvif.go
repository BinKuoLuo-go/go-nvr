/**
@Time : 2026/02/04 14:38
@Author: FangYao( 方少、)
@Description:onvif 协议封装
@Email: fy20030315@163.com
*/

package onvif

import (
	"context"
	"fmt"
	"github.com/use-go/onvif/media"
	"github.com/use-go/onvif/ptz"
	"log"
	"net/url"
	"strings"

	"github.com/use-go/onvif"
	"github.com/use-go/onvif/device"
	sdkdevice "github.com/use-go/onvif/sdk/device"
	sdkmedia "github.com/use-go/onvif/sdk/media"
	sdkptz "github.com/use-go/onvif/sdk/ptz"
)

// GetDeviceInformation 获取设备基本信息
func GetDeviceInformation(ctx context.Context, dev *onvif.Device) {
	getDeviceInformation := device.GetDeviceInformation{}
	response, err := sdkdevice.Call_GetDeviceInformation(ctx, dev, getDeviceInformation)
	if err != nil {
		panic(err)
	}
	fmt.Println("设备信息:", response)
}

// GetRTSPUri 获取 RTSP 流地址
func GetRTSPUri(ctx context.Context, dev *onvif.Device, password, username string) {
	// 获取媒体配置文件
	getProfiles := media.GetProfiles{}
	profilesResponse, err := sdkmedia.Call_GetProfiles(ctx, dev, getProfiles)
	if err != nil {
		fmt.Println(err)
	}

	profileToken := profilesResponse.Profiles[0].Token
	getStreamUri := media.GetStreamUri{
		ProfileToken: profileToken,
	}
	streamUriResponse, err := sdkmedia.Call_GetStreamUri(ctx, dev, getStreamUri)
	if err != nil {
		fmt.Println(err)
	}
	// 嵌入用户名和密码
	var rtspStr = ""
	uri := streamUriResponse.MediaUri
	parts := strings.SplitN(string(uri.Uri), "://", 2)
	if len(parts) == 2 {
		encodedPassword := url.QueryEscape(password)
		rtspStr = fmt.Sprintf("%s://%s:%s@%s", parts[0], username, encodedPassword, parts[1])
	}
	fmt.Println("凭证-RTSP地址:")
	fmt.Println(rtspStr)
}

// GetPTZService 获取 PTZ 服务能力
func GetPTZService(ctx context.Context, dev *onvif.Device) {
	// 创建请求
	getServiceCapabilities := ptz.GetServiceCapabilities{}

	// 发送请求并接收响应
	response, err := sdkptz.Call_GetServiceCapabilities(ctx, dev, getServiceCapabilities)
	if err != nil {
		log.Fatalf("获取 PTZ 服务能力失败: %v", err)
	}

	fmt.Println("设备支持的 PTZ 服务能力:")
	fmt.Printf("EFlip: %v\n", response.Capabilities.EFlip)
	fmt.Printf("Reverse: %v\n", response.Capabilities.Reverse)
	fmt.Printf("GetCompatibleConfigurations: %v\n", response.Capabilities.GetCompatibleConfigurations)
	fmt.Printf("MoveStatus: %v\n", response.Capabilities.MoveStatus)
	fmt.Printf("StatusPosition: %v\n", response.Capabilities.StatusPosition)
}
