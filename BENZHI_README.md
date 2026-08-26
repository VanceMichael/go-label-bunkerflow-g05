# BENZHI_README

这是一个面向船对船绿色甲醇加注作业的多租户运营管理系统，用于协调船舶资质、码头窗口、燃料质量、安全许可、交付取样和结算流程。

## 标准构建、运行和测试命令

进入容器后执行：

```bash
# 编译
cd '/app' && GOTOOLCHAIN=local go build ./...

# 启动
cd '/app' && GOTOOLCHAIN=local go run ./cmd/server

# 测试
cd '/app' && GOTOOLCHAIN=local go test ./...
```

## Docker 构建和进入容器

```bash
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh benzhi-task-259-amd64 linux/amd64
./build_benzhi_docker.sh benzhi-task-259-arm64 linux/arm64
docker run -it benzhi-task-259-amd64:latest
docker run -it --platform linux/arm64 benzhi-task-259-arm64:latest
```
