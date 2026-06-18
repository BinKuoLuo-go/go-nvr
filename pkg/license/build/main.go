/**
@Time : 2026-04-16
@Author: FangYao(方少、)
@Description: 独立授权生成工具(带YAML配置) 双击生成授权文件
@Email: fy20030315@163.com
*/

package main

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/denisbrodbeck/machineid"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// 配置文件结构
type AppConfig struct {
	SecretKey   string `yaml:"secret_key"`   // 签名密钥
	ExpireDays  int    `yaml:"expire_days"`  // 授权天数
	LicenseFile string `yaml:"license_file"` // 授权文件名
	Company     string `yaml:"company"`      // 公司名称
}

// 默认配置
var defaultConfig = AppConfig{
	SecretKey:   "go-nvr-2026-fangyao-secret",
	ExpireDays:  3650,
	LicenseFile: "license.txt",
	Company:     "xx有限公司", // 公司名
}

// 全局变量
var config AppConfig

// 配置文件操作
// loadConfig 读取yaml配置，不存在则自动创建
func loadConfig() error {
	// 配置文件路径
	configPath := "config.yaml"

	// 不存在则创建默认配置
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		data, _ := yaml.Marshal(defaultConfig)
		if err := os.WriteFile(configPath, data, 0644); err != nil {
			return err
		}
		fmt.Println("自动生成配置文件：config.yaml")
		config = defaultConfig
		return nil
	}

	// 读取配置
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(data, &config)
}

func GenerateMachineCode() string {
	// 获取系统级唯一设备ID
	//id, err := machineid.ID()
	// 获取系统级唯一设备ID + 唯一应用标识
	id, err := machineid.ProtectedID("FangYao_NiuBi")
	//fmt.Printf(id) // 注意我这是测试用，不要暴露唯一机器码
	if err != nil {
		return "unknown_device"
	}

	// 公司私有盐,wo这里写死一个固定字符串
	privateSalt := "ShuCheng_GoNVR_FangYao_NB"

	// 拼接：系统ID + 私有盐 再哈希 = 终极唯一机器码
	raw := strings.TrimSpace(id) + "_" + privateSalt

	// SHA256 格式化
	hash := sha256.Sum256([]byte(raw))
	return fmt.Sprintf("%x", hash[:])
}

// 签名授权生成
func generateSignature(machineCode string, issueAt int64, expireAt int64, company string) string {
	data := fmt.Sprintf("%s:%d:%d:%s", machineCode, issueAt, expireAt, company)
	h := hmac.New(sha256.New, []byte(config.SecretKey))
	h.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func GenerateLicense(machineCode string, company string) string {
	issueAt := time.Now().Unix()
	expireAt := time.Now().Add(time.Duration(config.ExpireDays) * 24 * time.Hour).Unix()
	signature := generateSignature(machineCode, issueAt, expireAt, company)

	lic := struct {
		Company     string `json:"company"`
		MachineCode string `json:"machine_code"`
		IssueAt     int64  `json:"issue_at"`
		ExpireAt    int64  `json:"expire_at"`
		Signature   string `json:"signature"`
	}{
		Company:     company,
		MachineCode: machineCode,
		IssueAt:     issueAt,
		ExpireAt:    expireAt,
		Signature:   signature,
	}

	bytes, _ := json.Marshal(lic)
	return base64.StdEncoding.EncodeToString(bytes)
}

func SaveLicense(lic string) error {
	return os.WriteFile(config.LicenseFile, []byte(lic), 0644)
}

// 主函数
func main() {
	fmt.Println("========== 授权生成工具 ==========")

	// 加载/生成配置文件
	if err := loadConfig(); err != nil {
		fmt.Printf("配置加载失败：%v\n", err)
		os.Exit(1)
	}
	fmt.Printf("配置加载成功 | 有效期：%d天\n", config.ExpireDays)
	fmt.Printf("授权公司：%s\n", config.Company) // 打印配置里的公司名

	// 获取当前电脑机器码
	machineCode := GenerateMachineCode()
	fmt.Printf("根据当前机器生成的唯一校验授权机器码：%s\n", machineCode)

	// 生成授权文件
	licenseStr := GenerateLicense(machineCode, config.Company)
	if err := SaveLicense(licenseStr); err != nil {
		fmt.Printf("授权生成失败：%v\n", err)
		os.Exit(1)
	}

	fmt.Printf("授权文件生成成功：%s\n", config.LicenseFile)
	fmt.Println("========================================")
	fmt.Println("提示：将 license.txt 放入主程序目录即可激活")
	os.Exit(0)
}
