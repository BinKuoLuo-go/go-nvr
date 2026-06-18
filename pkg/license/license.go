/**
@Time : 2026/04/16 14:18
@Author: FangYao( 方少、)
@Description:license授权
@Email: fy20030315@163.com
*/

package license

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/denisbrodbeck/machineid"
	"go-nvr/pkg/common"
	"os"
	"strings"
	"time"
)

// 配置项
const (
	secretKey      = "go-nvr-2026-fangyao-secret" // 密钥
	licenseFile    = "license.txt"
	lastRunFile    = "last_run.json"
	OurCompanyName = "山东数程信息科技有限公司"
)

// License 授权结构体
type License struct {
	Company     string `json:"company"`
	IssueAt     int64  `json:"issue_at"`
	MachineCode string `json:"machine_code"`
	ExpireAt    int64  `json:"expire_at"`
	Signature   string `json:"signature"`
}

// LastRun 上次运行时间
type LastRun struct {
	Timestamp int64 `json:"timestamp"`
}

// 生成机器码
func GenerateMachineCode() string {
	id, err := machineid.ProtectedID("FangYao_NiuBi")
	if err != nil {
		return "unknown_device"
	}

	privateSalt := "ShuCheng_GoNVR_FangYao_NB"
	raw := strings.TrimSpace(id) + "_" + privateSalt

	hash := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", hash[:])
}

// 校验签名算法
func generateSignature(machineCode string, issueAt, expireAt int64, company string) string {
	data := fmt.Sprintf("%s:%d:%d:%s", machineCode, issueAt, expireAt, company)
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// 读取授权
func LoadLicense() (string, error) {
	data, err := os.ReadFile(licenseFile)
	return string(data), err
}

// 解析并验证授权
func ParseLicense(encoded string) (*License, error) {
	bytes, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return nil, err
	}

	var lic License
	if err := json.Unmarshal(bytes, &lic); err != nil {
		return nil, err
	}

	// 校验签名
	if generateSignature(lic.MachineCode, lic.IssueAt, lic.ExpireAt, lic.Company) != lic.Signature {
		return nil, errors.New("[验证授权] 授权文件被篡改")
	}

	now := time.Now().Unix()

	if now < lic.IssueAt {
		return nil, errors.New("[验证授权] 系统时间被篡改")
	}

	if now > lic.ExpireAt {
		return nil, errors.New("[验证授权] 授权已过期")
	}

	return &lic, nil
}

// 防时间回退
func CheckTimeValidity() bool {
	now := time.Now().Unix()

	data, err := os.ReadFile(lastRunFile)
	if err == nil {
		var last LastRun
		_ = json.Unmarshal(data, &last)
		if now < last.Timestamp {
			return false
		}
	}

	lastData, _ := json.Marshal(LastRun{Timestamp: now})
	_ = os.WriteFile(lastRunFile, lastData, 0644)
	return true
}

// 对外入口
func Verify() error {
	if !CheckTimeValidity() {
		return errors.New("[校验系统时间] 系统时间被篡改")
	}

	localMachine := GenerateMachineCode()

	licStr, err := LoadLicense()
	if err != nil {
		return fmt.Errorf("[验证授权] 未找到授权文件，机器码：%s", localMachine)
	}

	lic, err := ParseLicense(licStr)
	if err != nil {
		return err
	}

	if lic.MachineCode != localMachine {
		return errors.New("[验证授权] 授权与当前硬件不匹配")
	}

	common.Log.Infof("授权成功（由 %s 授权）", OurCompanyName)
	common.Log.Infof("客户公司：%s", lic.Company)
	common.Log.Infof("过期时间：%s", time.Unix(lic.ExpireAt, 0).Format("2006-01-02 15:04:05"))

	return nil
}
