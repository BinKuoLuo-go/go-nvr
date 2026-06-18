# GO-NVR 视频监控平台
现在是简陋版，后续等把想要的功能实现了再详细优化

## 核心基于
+ [go2rtc](https://github.com/AlexxIT/go2rtc/)做流媒体转发
+ [onnxruntime_go](https://github.com/yalue/onnxruntime_go)调用onnx实现yolo检测(注意需要开启CGO，具体请看官方文档和issues)
+ [mediamtx](https://github.com/bluenviron/mediamtx)主要用于告警录制(因为go2rtc不支持🤦‍♀️)

## 灵感来源:

- [frigate](https://github.com/blakeblackshear/frigate)一个python版本的nvr
- [gowvp](https://github.com/gowvp/owl)一个go实现的nvr,支持GB28181协议

## 功能：
+ 支持浏览器无插件播放摄像头视频流(比如海康设备就不需要使用他家的插件什么的了，很麻烦，有坑)
+ 支持告警录制(已完成)
+ 支持ROI区域绘制和算法标签设置(已完成)
+ 支持yolo检测(已完成,已测模型支持：yolo8,yolo11,yolo12)
+ 支持边缘端部署(已完成)
+ 

+ 支持 Docker 部署（后续支持，算了吃不了细糠）


## 注意
+ 就是如果是win系统的话，onnxruntime_go库你需要开启CGO和下载配置编译器,我下是mingw64，还有就是cuda版本，我的是12.x左右的

先这样吧后续我在详细写写


## jetson设备编译onnxruntime
```bash

# 1、配置 CMake
# 下载适配Jetson的aarch64架构CMake 3.29.0
wget https://github.com/Kitware/CMake/releases/download/v3.29.0/cmake-3.29.0-linux-aarch64.tar.gz

# 解压CMake压缩包
tar -zxvf cmake-3.29.0-linux-aarch64.tar.gz

# 配置临时环境变量（当前终端立即生效）
export PATH=$(pwd)/cmake-3.29.0-linux-aarch64/bin:$PATH

# 配置永久环境变量
echo "export PATH=$(pwd)/cmake-3.29.0-linux-aarch64/bin:\$PATH" >> ~/.bashrc
source ~/.bashrc

# 验证CMake配置结果
cmake --version

# 克隆源码
# 拉取
git clone --branch v1.24.1 --depth 1 --recursive https://github.com/microsoft/onnxruntime onnxruntime_1.24.1

# 进入该目录
cd onnxruntime_1.24.1

# 执行编译
./build.sh \
--config Release \
--skip_tests \
--cmake_extra_defines onnxruntime_BUILD_UNIT_TESTS=0 \
--cmake_extra_defines CMAKE_CUDA_ARCHITECTURES=native \
--build_shared_lib \
--parallel 2 \
--nvcc_threads 1 \
--use_cuda \
--cudnn_home /usr/lib/aarch64-linux-gnu/ \
--cuda_home /usr/local/cuda/ \
--use_tensorrt \
--tensorrt_home /usr/lib/aarch64-linux-gnu/


# grpc安装
https://github.com/protocolbuffers/protobuf/releases
go get google.golang.org/grpc
go get google.golang.org/protobuf
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest


# 自动生成 Go 代码
protoc --go_out=. --go-grpc_out=. --go_opt=paths=source_relative --go-grpc_opt=paths=source_relative ./pkg/grpc/protos/analysis.proto