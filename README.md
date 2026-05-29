# TermReq

`TermReq` 是一个用 Go 写的终端 HTTP Client，面向日常接口调试场景。

它现在提供：

- 终端内编辑 `method / url / timeout / headers / body`
- 使用 TUI 界面发送请求和查看响应
- 直接粘贴浏览器复制的 `curl` 并应用到当前请求
- 将当前请求导出为 `curl` 命令并复制到剪贴板
- 自动保存请求历史，并在下次启动时恢复
- 自动补全 URL 协议
- JSON 响应自动格式化
- 二进制响应自动转十六进制预览
- 响应大小和耗时展示
- 大响应体截断保护

当前 UI 基于 `Bubble Tea + Bubbles + Lip Gloss`。

## 运行要求

- Go `1.26+`
- 一个正常的交互式终端

## 快速开始

在项目目录中直接运行：

```bash
go run .
```

构建二进制：

```bash
go build .
```

如果你当前环境是离线或受限环境，也可以这样跑：

```bash
env GOCACHE=/tmp/go-build-cache GOPROXY=off GOSUMDB=off GOFLAGS=-mod=mod go run .
```

## 基本用法

启动后，左侧是请求编辑区，右侧是响应查看区。

推荐流程：

1. 选择 HTTP 方法
2. 输入 URL
3. 设置超时
4. 编辑请求头
5. 编辑请求体
6. 按 `Ctrl+S` 发送请求
7. 按 `Tab` 切到 `History` 浏览最近请求，按 `Enter` 回填
8. 在响应区滚动查看结果

也可以在浏览器开发者工具里使用 `Copy as cURL`，回到 TermReq 后在任意焦点位置直接粘贴。TermReq 会自动导入 `method / url / headers / body`。

## 快捷键

| 键位 | 作用 |
| --- | --- |
| `Tab` | 切换到下一个区域 |
| `Shift+Tab` | 切换到上一个区域 |
| `Ctrl+S` | 发送请求 |
| `Ctrl+E` | 导出当前请求为 cURL 命令并复制到剪贴板 |
| `Ctrl+C` | 退出 |
| `Left / Right` 或 `h / l` | 在方法区域切换 Method，或在 Header 区切换 `key/value` |
| `Up / Down` 或 `j / k` | 在 Header 行之间移动，或在响应区滚动 |
| `Ctrl+N` | 在 Header 区新增一行 |
| `Ctrl+D` | 在 Header 区删除当前行 |
| `Ctrl+P` | 在 Body 区格式化 JSON |
| `Enter` | 在 History 区加载当前选中的历史请求 |
| `PageUp / PageDown` | 在响应区翻页 |
| `Home / End` | 在响应区跳到顶部或底部 |

## 请求与响应行为

### URL

- 如果 URL 没写协议，默认补成 `https://`
- 空 URL 会直接报错

### Timeout

- 默认超时是 `30s`
- 纯数字会按秒解释
- 也支持 `250ms`、`2s`、`1m` 这类 Go duration 格式

### Headers

- Header 使用结构化 `key/value` 表单编辑
- 空行会被自动忽略
- 非空行会按 `Key: Value` 转成实际请求头
- Header 名会按标准 MIME 形式规范化

### Body

- Body 支持多行编辑
- 在 Body 区按 `Ctrl+P` 会尝试按 JSON 格式化

### Response

- 默认最多读取 `2 MiB` 响应体
- `JSON` 会自动美化
- 文本内容直接展示
- 二进制内容会显示十六进制 dump
- 默认 `User-Agent` 是 `TermReq/1.0`

### History

- 每次发送请求后都会自动写入本地历史
- 历史默认保存在用户配置目录下的 `termreq/history.json`
- 历史会在下次启动时自动加载
- 相同请求会被移动到顶部，不会无限重复堆积
- 在 `History` 焦点下可用 `Up / Down` 浏览、`Enter` 回填、`Ctrl+D` 删除

## 项目结构

- [main.go](./main.go): 程序入口
- [app.go](./app.go): TUI 主模型、布局、交互
- [theme.go](./theme.go): 颜色和样式
- [httpclient.go](./httpclient.go): HTTP 请求执行、响应格式化
- [httpclient_test.go](./httpclient_test.go): 核心逻辑测试

## 开发

运行测试：

```bash
go test ./...
```

构建检查：

```bash
go build ./...
```

## 说明

这个项目当前重点是终端里的请求编辑和响应查看体验，不是要完整替代 Postman 或 Insomnia。

如果后面继续扩展，比较自然的方向有：

- 收藏夹
- Header / Body 标签页
- 更细的响应信息面板
- 导入导出请求配置
