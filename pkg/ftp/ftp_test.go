/**
@Time : 2026/03/30 11:09
@Author: FangYao( 方少、)
@Description:
@Email: fy20030315@163.com
*/

package ftp

import (
	"fmt"
	"image/jpeg"
	"log"
	"os"
	"testing"
	"time"
)

func TestUploadSnapshotToNVR(t *testing.T) {
	// frp服务器信息
	ip := "192.168.1.4"
	user := "admin"
	pwd := "1234abcd"
	// 本地测试图片
	imgPath := `E:\FyProject\Go\porject\go-nvr\pkg\frp\test.jpg`

	// 读取图片
	imgData, err := os.ReadFile(imgPath)
	if err != nil {
		t.Fatal(err)
	}

	// FTP上传
	fileName := fmt.Sprintf("alarm_%d.jpg", time.Now().Unix())
	url, err := UploadToDeviceFTP(ip, user, pwd, imgData, fileName)
	if err != nil {
		t.Fatal(err)
	}

	t.Log("上传成功！")
	t.Log("设备图片路径：", url)
}

func TestJPEG(t *testing.T) {
	//srcImg1 := "E:\\FyProject\\Go\\porject\\go-nvr\\pkg\\ftp\\by.jpg"
	//dstImg1 := "byy.jpg"

	srcImg2 := "E:\\FyProject\\Go\\porject\\go-nvr\\pkg\\ftp\\cj.jpg"
	dstImg2 := "cjjj.jpg"
	//if err := compressJPEG(srcImg1, dstImg1, 75); err != nil {
	//	t.Fatalf("第一张图片压缩失败：%v", err)
	//}

	if err := compressJPEG(srcImg2, dstImg2, 10); err != nil {
		t.Fatalf("第一张图片压缩失败：%v", err)
	}
}

func compressJPEG(src, dst string, q int) error {
	srcc, err := os.Open(src)
	if err != nil {
		log.Fatal(err)
	}
	defer srcc.Close()

	img, err := jpeg.Decode(srcc)
	if err != nil {
		log.Fatal(err)
	}
	dstfile, err := os.Create(dst)
	if err != nil {
		log.Fatal(err)
	}
	defer dstfile.Close()

	return jpeg.Encode(dstfile, img, &jpeg.Options{Quality: q})
}
