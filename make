

// 平台
linux：Linux 系统（最常用，如 Ubuntu、CentOS 等）
windows：Windows 系统（生成 .exe 可执行文件）
darwin：苹果 macOS 系统（基于 BSD 的 macOS，如 Intel 或 M 系列芯片的 Mac）
freebsd：FreeBSD 系统
openbsd：OpenBSD 系统
netbsd：NetBSD 系统
dragonfly：DragonFly BSD 系统
solaris：Oracle Solaris 系统
illumos：Illumos 系统（类 Solaris 的开源系统）
android：Android 系统（需结合特定工具链）
ios：iOS 系统（需配合 Xcode 工具链）
js：WebAssembly 环境（编译为 .wasm 文件，可在浏览器中运行）
plan9：Plan 9 系统（小众操作系统）

// 架构
amd64：x86-64 架构（最常用，如 Intel/AMD 的 64 位处理器）
386：x86 32 位架构（老旧的 32 位 Intel/AMD 处理器）
arm64：ARM 64 位架构（如手机、服务器的 ARM 芯片，如苹果 M 系列、AWS Graviton）
arm：ARM 32 位架构（如旧款手机、嵌入式设备，支持 armv5/armv6/armv7）
ppc64：PowerPC 64 位架构（大端模式，如部分 IBM 服务器）
ppc64le：PowerPC 64 位架构（小端模式）
mips：MIPS 架构（大端模式，常见于路由器、嵌入式设备）
mipsle：MIPS 架构（小端模式）
mips64：MIPS 64 位架构（大端模式）
mips64le：MIPS 64 位架构（小端模式）
s390x：IBM Z 系列大型机架构（64 位）
riscv64：RISC-V 64 位架构（新兴开源架构，用于嵌入式、服务器等）
wasm：WebAssembly 虚拟架构（配合 GOOS=js 使用


// 交叉编译
$env:CGO_ENABLED = "0"  // 0无cgo 1开启cgo
$env:GOARCH = "amd64"  // "amd64","arm","""arm64"
$env:GOOS = "linux" // "windows" "linux" "darwin"
// $env:GOARM="7" // 仅arm架构使用


# 执行编译 加参数-ldflags="-s -w" -o 用于去掉调试信息函数名，简单的防反编译
go build -ldflags="-s -w" -o gohljgs main.go
# 执行编译（生成Linux平台的可执行文件）
go build -ldflags="-s -w" -o xx main.go
# 执行编译（生成windows平台的可执行文件）
go build -ldflags="-s -w" -o xx.exe main.go

