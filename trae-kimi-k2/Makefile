.PHONY: test build example deps clean

# 默认目标
all: test

# 安装依赖
deps:
	go mod tidy
	go mod download

# 运行测试
test:
	go test -v ./...

# 运行测试并生成覆盖率报告
test-coverage:
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# 构建示例
build-example:
	cd example && go build -o ../bin/example

# 运行示例
run-example: build-example
	./bin/example

# 格式化代码
fmt:
	go fmt ./...

# 静态检查
lint:
	go vet ./...

# 清理
clean:
	rm -rf bin/
	rm -f coverage.out coverage.html

# 帮助
help:
	@echo "Available targets:"
	@echo "  deps          - 安装依赖"
	@echo "  test          - 运行测试"
	@echo "  test-coverage - 运行测试并生成覆盖率报告"
	@echo "  build-example - 构建示例程序"
	@echo "  run-example   - 运行示例程序"
	@echo "  fmt           - 格式化代码"
	@echo "  lint          - 静态检查"
	@echo "  clean         - 清理构建产物"
	@echo "  help          - 显示帮助信息"