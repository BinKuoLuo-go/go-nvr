/**
@Time : 2026/03/13 09:03
@Author: FangYao( 方少、)
@Description: 测试plc s7协议
@Email: fy20030315@163.com
*/

package plc

import (
	"fmt"
	"go-nvr/pkg/config"
	"testing"
	"time"
)

func TestNewPLCClient(t *testing.T) {
	plc, err := NewPLCS7Client(&config.PlcConfig{
		IP:   "192.168.1.2",
		Rack: 0,
		Slot: 1,
	})

	if err != nil {
		panic(err)
	}

	defer plc.Close()

	fmt.Println("开始测试")
	testPoints := []struct {
		byteAddr int
		bit      int
	}{
		{16, 0},
		{17, 0},
		{18, 0},
		{19, 0},
		{20, 0},
		{21, 0},
		{22, 0},
	}
	operationTimes := map[string][]float64{
		"read":        {},
		"write_true":  {},
		"write_false": {},
		"restore":     {},
	}
	type origin struct {
		byteAddr int
		bit      int
		value    bool
	}

	var originals []origin

	// 读取原始值
	for _, point := range testPoints {
		start := time.Now()
		val, err := plc.ReadBit(point.byteAddr, point.bit)
		if err != nil {
			t.Fatalf("读取失败 M%d.%d: %v", point.byteAddr, point.bit, err)
		}

		elapsed := time.Since(start).Milliseconds()
		operationTimes["read"] = append(operationTimes["read"], float64(elapsed))
		fmt.Printf("M%d.%d 原始值: %v [耗时: %dms]\n",
			point.byteAddr, point.bit, val, elapsed)

		originals = append(originals, origin{
			byteAddr: point.byteAddr,
			bit:      point.bit,
			value:    val,
		})
	}

	fmt.Println("正在写入 True...")

	for _, p := range testPoints {

		start := time.Now()

		err := plc.WriteBit(p.byteAddr, p.bit, true)

		elapsed := time.Since(start).Milliseconds()

		operationTimes["write_true"] =
			append(operationTimes["write_true"], float64(elapsed))

		if err != nil {
			fmt.Printf("写入失败 M%d.%d\n", p.byteAddr, p.bit)
			continue
		}

		time.Sleep(100 * time.Millisecond)

		verifyStart := time.Now()

		val, _ := plc.ReadBit(p.byteAddr, p.bit)

		verifyElapsed := time.Since(verifyStart).Milliseconds()

		status := "验证失败"
		if val {
			status = "成功"
		}

		fmt.Printf("%s M%d.%d=True 验证值:%v [写入:%dms 验证:%dms]\n",
			status, p.byteAddr, p.bit, val, elapsed, verifyElapsed)
	}

	fmt.Println("等待 2 秒...")
	time.Sleep(2 * time.Second)

	// 写入 False
	fmt.Println("正在写入 False...")

	for _, p := range testPoints {

		start := time.Now()

		err := plc.WriteBit(p.byteAddr, p.bit, false)

		elapsed := time.Since(start).Milliseconds()

		operationTimes["write_false"] =
			append(operationTimes["write_false"], float64(elapsed))

		if err != nil {
			fmt.Printf("写入失败 M%d.%d\n", p.byteAddr, p.bit)
			continue
		}

		time.Sleep(100 * time.Millisecond)

		verifyStart := time.Now()

		val, _ := plc.ReadBit(p.byteAddr, p.bit)

		verifyElapsed := time.Since(verifyStart).Milliseconds()

		status := "验证失败"
		if !val {
			status = "成功"
		}

		fmt.Printf("%s M%d.%d=False 验证值:%v [写入:%dms 验证:%dms]\n",
			status, p.byteAddr, p.bit, val, elapsed, verifyElapsed)
	}

	fmt.Println("恢复原始值...")

	for _, o := range originals {

		start := time.Now()

		err := plc.WriteBit(o.byteAddr, o.bit, o.value)

		elapsed := time.Since(start).Milliseconds()

		operationTimes["restore"] =
			append(operationTimes["restore"], float64(elapsed))

		if err != nil {
			fmt.Printf("恢复失败 M%d.%d\n", o.byteAddr, o.bit)
			continue
		}

		time.Sleep(100 * time.Millisecond)

		val, _ := plc.ReadBit(o.byteAddr, o.bit)

		verifyElapsed := time.Since(start).Milliseconds()

		status := "验证失败"
		if val == o.value {
			status = "成功"
		}

		fmt.Printf("%s M%d.%d 恢复为 %v 验证:%v [恢复:%dms 验证:%dms]\n",
			status, o.byteAddr, o.bit, o.value, val, elapsed, verifyElapsed)
	}

	// 统计

	fmt.Println("操作耗时统计(ms)")

	avg := func(arr []float64) float64 {
		var sum float64
		for _, v := range arr {
			sum += v
		}
		return sum / float64(len(arr))
	}

	fmt.Printf("读取平均: %.2fms\n", avg(operationTimes["read"]))
	fmt.Printf("写True平均: %.2fms\n", avg(operationTimes["write_true"]))
	fmt.Printf("写False平均: %.2fms\n", avg(operationTimes["write_false"]))
	fmt.Printf("恢复平均: %.2fms\n", avg(operationTimes["restore"]))

}
