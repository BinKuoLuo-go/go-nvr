/**
@Time : 2026/03/13 09:02
@Author: FangYao( 方少、)
@Description: plc s7协议封装
@Email: fy20030315@163.com
*/

package plc

import (
	"fmt"
	"go-nvr/pkg/config"
	"time"

	"github.com/robinson/gos7"
)

// PLC客户端
type PLCClient struct {
	handler *gos7.TCPClientHandler
	client  gos7.Client
}

// 创建PLCS7客户端
func NewPLCS7Client(cfg *config.PlcConfig) (*PLCClient, error) {
	handler := gos7.NewTCPClientHandler(cfg.IP, cfg.Rack, cfg.Slot)
	handler.Timeout = 5 * time.Second
	handler.IdleTimeout = 5 * time.Second

	err := handler.Connect()
	if err != nil {
		return nil, err
	}

	client := gos7.NewClient(handler)

	return &PLCClient{
		handler: handler,
		client:  client,
	}, nil
}

// 关闭连接
func (p *PLCClient) Close() {
	if p.handler != nil {
		p.handler.Close()
	}
}

// DB 区操作
// 读取 DB
func (p *PLCClient) ReadDB(db int, start int, size int) ([]byte, error) {

	buffer := make([]byte, size)

	err := p.client.AGReadDB(db, start, size, buffer)
	if err != nil {
		return nil, err
	}

	return buffer, nil
}

// 写入 DB
func (p *PLCClient) WriteDB(db int, start int, data []byte) error {

	size := len(data)

	return p.client.AGWriteDB(db, start, size, data)
}

// M 区操作
// 读取 M 区字节
func (p *PLCClient) ReadMB(start int, size int) ([]byte, error) {

	buffer := make([]byte, size)

	err := p.client.AGReadMB(start, size, buffer)
	if err != nil {
		return nil, err
	}

	return buffer, nil
}

// 写入 M 区字节
func (p *PLCClient) WriteMB(start int, data []byte) error {

	size := len(data)

	return p.client.AGWriteMB(start, size, data)
}

// Bit 操作
// 读取位
func (p *PLCClient) ReadBit(byteAddr int, bit int) (bool, error) {

	buffer := make([]byte, 1)

	err := p.client.AGReadMB(byteAddr, 1, buffer)
	if err != nil {
		return false, err
	}

	value := (buffer[0]>>bit)&1 == 1

	return value, nil
}

// 写入位
func (p *PLCClient) WriteBit(byteAddr int, bit int, value bool) error {

	buffer := make([]byte, 1)

	// 先读取原字节
	err := p.client.AGReadMB(byteAddr, 1, buffer)
	if err != nil {
		return err
	}

	// 修改bit
	if value {
		buffer[0] |= 1 << bit
	} else {
		buffer[0] &^= 1 << bit
	}

	return p.client.AGWriteMB(byteAddr, 1, buffer)
}

// Byte 操作
// 读取 byte
func (p *PLCClient) ReadMBByte(byteAddr int) (byte, error) {

	data, err := p.ReadMB(byteAddr, 1)
	if err != nil {
		return 0, err
	}

	return data[0], nil
}

// 写入 byte
func (p *PLCClient) WriteMBByte(byteAddr int, value byte) error {

	data := []byte{value}

	return p.client.AGWriteMB(byteAddr, 1, data)
}

// Ping / 连接检测
func (p *PLCClient) Ping() error {

	_, err := p.ReadMB(0, 1)
	if err != nil {
		return fmt.Errorf("PLC连接检测，失败: %v", err)
	}

	return nil
}
