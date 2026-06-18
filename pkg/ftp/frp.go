/**
@Time : 2026/03/30 11:08
@Author: FangYao( 方少、)
@Description: ftp
@Email: fy20030315@163.com
*/

package ftp

import (
	"bytes"
	"fmt"
	"github.com/jlaffaye/ftp"
	"time"
)

func UploadToDeviceFTP(ip string, user, pwd string, imgData []byte, fileName string) (string, error) {
	// 连接FTP
	addr := fmt.Sprintf("%s:21", ip)
	conn, err := ftp.Dial(addr, ftp.DialWithTimeout(5*time.Second))
	if err != nil {
		return "", fmt.Errorf("FTP连接失败: %w", err)
	}
	defer conn.Quit()

	// 登录
	if err := conn.Login(user, pwd); err != nil {
		return "", fmt.Errorf("FTP登录失败: %w", err)
	}

	// 创建目录
	err = conn.MakeDir("alarm")
	err = conn.ChangeDir("alarm")
	err = conn.MakeDir("images")
	_ = conn.ChangeDir("images")

	// 上传图片（内存直传，不写本地）
	buf := bytes.NewBuffer(imgData)
	if err := conn.Stor(fileName, buf); err != nil {
		return "", fmt.Errorf("上传失败: %w", err)
	}

	// 返回设备访问路径
	return fmt.Sprintf("ftp://%s:%s@%s/alarm/images/%s", user, pwd, ip, fileName), nil
}
