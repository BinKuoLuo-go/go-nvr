/**
@Time : 2026/04/13 11:42
@Author: FangYao( 方少、)
@Description: 获取显卡信息123
@Email: fy20030315@163.com
*/

package common

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

// NvidiaGPU NVIDIA显卡信息
type NvidiaGPU struct {
	ID          int    `json:"id"`
	Name        string `json:"name"`
	MemoryTotal string `json:"memory_total"`
	MemoryFree  string `json:"memory_free"`
}

// GetNvidiaGPUs 通过nvidia-smi获取显卡信息
func GetNvidiaGPUs() ([]NvidiaGPU, error) {
	cmd := exec.Command("nvidia-smi", "--query-gpu=index,name,memory.total,memory.free", "--format=csv,noheader,nounits")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(output), "\n")
	var gpus []NvidiaGPU

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Split(line, ", ")
		if len(parts) < 4 {
			continue
		}

		id, _ := strconv.Atoi(strings.TrimSpace(parts[0]))
		name := strings.TrimSpace(parts[1])
		total := strings.TrimSpace(parts[2]) + " MB"
		free := strings.TrimSpace(parts[3]) + " MB"

		gpus = append(gpus, NvidiaGPU{
			ID:          id,
			Name:        name,
			MemoryTotal: total,
			MemoryFree:  free,
		})
	}

	if len(gpus) == 0 {
		return nil, fmt.Errorf("未检测到NVIDIA显卡")
	}
	return gpus, nil
}

// GetCUDADeviceCount 获取CUDA设备数量
func GetCUDADeviceCount() int {
	gpus, err := GetNvidiaGPUs()
	if err != nil {
		return 0
	}
	return len(gpus)
}
