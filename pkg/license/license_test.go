/**
@Time : 2026/04/16 14:35
@Author: FangYao( 方少、)
@Description:
@Email: fy20030315@163.com
*/

package license

import (
	"log"
	"testing"
)

// TestPrintMachineCode 获取当前机器的唯一机器码
// 执行命令：go test -run TestPrintMachineCode -v
func TestPrintMachineCode(t *testing.T) {
	machineCode := GenerateMachineCode()
	log.Printf("生成机器码：%s\n", machineCode)
	log.Println("复制此机器码，用于生成授权")
}
