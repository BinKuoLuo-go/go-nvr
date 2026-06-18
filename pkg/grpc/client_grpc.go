/**
@Time : 2026/04/325 15:15
@Author: FangYao( 方少、)
@Description: grpc 客户端
@Email: fy20030315@163.com
*/

package grpc

import (
	"context"
	"go-nvr/pkg/config"
	"go-nvr/pkg/grpc/protos"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"sync"
)

var (
	client     protos.InferServiceClient
	clientOnce sync.Once
	conn       *grpc.ClientConn
)

// InitGrpcClient 初始化gRPC客户端
func InitGrpcClient() {
	clientOnce.Do(func() {
		addr := config.Conf.Onnx.TensorRTGrpcAddr
		if addr == "" {
			addr = "127.0.0.1:50051"
		}

		c, err := grpc.NewClient(addr,
			grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithDefaultCallOptions(grpc.MaxCallRecvMsgSize(1024*1024*100)),
		)
		if err != nil {
			panic("连接TensorRT gRPC服务失败: " + err.Error())
		}
		conn = c
		client = protos.NewInferServiceClient(c)
	})
}

// Close 关闭连接
func Close() {
	if conn != nil {
		_ = conn.Close()
	}
}

// Infer 调用Python TensorRT推理
func Infer(imageData []byte, width, height int32, confThresh, nmsThresh float32, modelType string) ([]*protos.BoundingBox, error) {
	req := &protos.InferRequest{
		ImageData:     imageData,
		Width:         width,
		Height:        height,
		ConfThreshold: confThresh,
		NmsThreshold:  nmsThresh,
		ModelType:     modelType,
	}

	resp, err := client.Infer(context.Background(), req)
	if err != nil {
		return nil, err
	}
	return resp.Boxes, nil
}
