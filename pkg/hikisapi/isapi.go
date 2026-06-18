/**
@Time : 2026/03/08 10:10
@Author: FangYao(方少、)
@Description: 海康ISAPI协议工具 获取海康设备信息
@Email: fy20030315@163.com
*/

package hikisapi

import (
	"encoding/xml"
	"fmt"
	digest "github.com/xinsnake/go-http-digest-auth-client"
	"go-nvr/model/response"
	"io"
	"strconv"
	"strings"
)

// 设备信息XML /ISAPI/System/deviceinfo
type DeviceInfoXML struct {
	XMLName         xml.Name `xml:"DeviceInfo"`
	DeviceType      string   `xml:"deviceType"`
	DeviceName      string   `xml:"deviceName"`
	Model           string   `xml:"model"`
	MacAddress      string   `xml:"macAddress"`
	SerialNumber    string   `xml:"serialNumber"`
	FirmwareVersion string   `xml:"firmwareVersion"`
	TelecontrolID   string   `xml:"telecontrolID"`
	SystemContact   string   `xml:"systemContact"`
}

// 系统状态XML /ISAPI/System/status
type SystemStatusXML struct {
	CPUUtilization  string `xml:"CPUList>CPU>cpuUtilization"`
	DeviceUpTime    string `xml:"deviceUpTime"`
	MemoryUsage     string `xml:"MemoryList>Memory>memoryUsage"`
	MemoryAvailable string `xml:"MemoryList>Memory>memoryAvailable"`
}

// 码流信息XML /ISAPI/Streaming/channels/101
type StreamInfoXML struct {
	XMLName               xml.Name `xml:"StreamingChannel"`
	ChannelName           string   `xml:"channelName"`
	VideoCodecType        string   `xml:"Video>videoCodecType"`
	ConstantBitRate       string   `xml:"Video>constantBitRate"`
	VideoResolutionWidth  string   `xml:"Video>videoResolutionWidth"`
	VideoResolutionHeight string   `xml:"Video>videoResolutionHeight"`
}

// GetFullDeviceInfo 获取完整设备信息
func GetFullDeviceInfo(ip string, port int, username, password string) (*response.HikDeviceFullInfoResp, error) {
	resp := &response.HikDeviceFullInfoResp{
		IP:       ip,
		Port:     port,
		Username: username,
		Remark:   "success",
	}

	baseURL := fmt.Sprintf("http://%s:%d", ip, port)

	// 获取设备信息
	devInfo, err := getDeviceInfo(baseURL, username, password)
	if err != nil {
		resp.Remark = fmt.Sprintf("获取设备信息失败: %v", err)
		return resp, nil
	}
	fillDeviceInfo(resp, devInfo)

	// 获取系统状态
	status, err := getSystemStatus(baseURL, username, password)
	if err != nil {
		resp.Remark = fmt.Sprintf("获取系统状态失败: %v", err)
	} else {
		fillSystemStatus(resp, status, resp.DeviceType)
	}

	// 摄像头获取码流
	if resp.DeviceType == "摄像头" {
		stream, err := getStreamInfo(baseURL, username, password)
		if err != nil {
			resp.Remark = fmt.Sprintf("获取码流信息失败: %v", err)
		} else {
			fillStreamInfo(resp, stream)
		}
	}

	return resp, nil
}

// digestGet 通用 Digest 请求
func digestGet(url, user, pwd string) ([]byte, error) {
	dr := digest.NewRequest(user, pwd, "GET", url, "")
	resp, err := dr.Execute()
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(resp.Body)
}

func getDeviceInfo(baseURL, user, pwd string) (*DeviceInfoXML, error) {
	url := baseURL + "/ISAPI/System/deviceinfo"
	body, err := digestGet(url, user, pwd)
	if err != nil {
		return nil, err
	}
	var info DeviceInfoXML
	err = xml.Unmarshal(body, &info)
	return &info, err
}

func getSystemStatus(baseURL, user, pwd string) (*SystemStatusXML, error) {
	url := baseURL + "/ISAPI/System/status"
	body, err := digestGet(url, user, pwd)
	if err != nil {
		return nil, err
	}
	var status SystemStatusXML
	err = xml.Unmarshal(body, &status)
	return &status, err
}

func getStreamInfo(baseURL, user, pwd string) (*StreamInfoXML, error) {
	url := baseURL + "/ISAPI/Streaming/channels/101"
	body, err := digestGet(url, user, pwd)
	if err != nil {
		return nil, err
	}
	var stream StreamInfoXML
	err = xml.Unmarshal(body, &stream)
	return &stream, err
}

// 数据填充
func fillDeviceInfo(resp *response.HikDeviceFullInfoResp, info *DeviceInfoXML) {
	resp.DeviceName = info.DeviceName
	resp.Model = info.Model
	resp.MacAddress = info.MacAddress
	resp.SerialNumber = info.SerialNumber
	resp.Firmware = info.FirmwareVersion
	resp.TelecontrolID = info.TelecontrolID
	resp.SystemContact = strings.ReplaceAll(info.SystemContact, ".China", "")
	if info.DeviceType == "IPCamera" {
		resp.DeviceType = "摄像头"
	} else {
		resp.DeviceType = "硬盘录像机"
	}
}

func fillSystemStatus(resp *response.HikDeviceFullInfoResp, status *SystemStatusXML, devType string) {
	resp.CPU = status.CPUUtilization
	sec, _ := strconv.ParseInt(status.DeviceUpTime, 10, 64)
	resp.Uptime = sec / 3600
	if devType == "摄像头" {
		resp.Memory = status.MemoryUsage
	} else {
		used, _ := strconv.ParseFloat(status.MemoryUsage, 64)
		avail, _ := strconv.ParseFloat(status.MemoryAvailable, 64)
		if used+avail > 0 {
			resp.Memory = fmt.Sprintf("%.2f", used/(used+avail)*100)
		}
	}
}

func fillStreamInfo(resp *response.HikDeviceFullInfoResp, stream *StreamInfoXML) {
	resp.ChannelName = stream.ChannelName
	resp.CodecType = stream.VideoCodecType
	resp.BitRate = stream.ConstantBitRate
	resp.Resolution = fmt.Sprintf("%sx%s", stream.VideoResolutionWidth, stream.VideoResolutionHeight)
}
