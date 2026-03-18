# grok-proxy

grok-proxy 是一个基于 Go 语言的代理服务，将 Grok 网页聊天转换为 OpenAI API 兼容格式。用户使用自己的 sso Cookie 进行调用。

## 功能特性

- OpenAI API 兼容格式
- 支持流式和非流式响应
- 支持多种 Grok 模型
- 支持思考模式 (reasoning_content)
- 支持多模态图片输入
- 支持图片生成
- 支持联网搜索
- 支持 SOCKS5 代理池（随机轮询）

## 快速开始

```bash
docker run -d -p 8080:8080 ghcr.io/kao0312/grok-proxy:latest
```

挂载代理池：

```bash
docker run -d \
  -p 8080:8080 \
  -v ./proxies.txt:/app/proxies.txt \
  ghcr.io/kao0312/grok-proxy:latest
```

自定义配置：

```bash
docker run -d \
  -p 8080:8080 \
  -e LOG_LEVEL=ERROR \
  -v ./proxies.txt:/app/proxies.txt \
  ghcr.io/kao0312/grok-proxy:latest
```

## 环境变量

| 变量名 | 说明 | 默认值 |
|--------|------|--------|
| PORT | 监听端口 | 8080 |
| LOG_LEVEL | 日志级别 | INFO |

## 获取 Grok Cookie

1. 登录 https://grok.com
2. 打开浏览器开发者工具 (F12)
3. 切换到 Application/Storage 标签
4. 在 Cookies 中找到 `sso` 或 `sso-rw` 字段
5. 复制其值作为 API 调用的 Authorization

## 支持的模型

| 模型名称 | ModelMode |
|----------|------|
| grok-3 | MODEL_MODE_FAST |
| grok-4 | MODEL_MODE_EXPERT |
| grok-4-auto | MODEL_MODE_AUTO |
| grok-4.1-thinking | MODEL_MODE_GROK_4_1_THINKING |
| grok-4.20 | MODEL_MODE_GROK_420 |

## 使用示例

### 基础对话

```bash
curl http://localhost:8080/v1/chat/completions \
  -H "Authorization: Bearer YOUR_GROK_COOKIE" \
  -H "Content-Type: application/json" \
  -d '{
    "model": "grok-4",
    "messages": [{"role": "user", "content": "hello"}],
    "stream": true
  }'
```

### 多模态请求

```json
{
  "model": "grok-4.1",
  "messages": [
    {
      "role": "user",
      "content": [
        {"type": "text", "text": "描述这张图片"},
        {"type": "image_url", "image_url": {"url": "https://example.com/image.jpg"}}
      ]
    }
  ]
}
```

### 查看可用模型

```bash
curl http://localhost:8080/v1/models
```

## 支持的图片格式

- HTTP/HTTPS URL
- Base64 编码 (data:image/jpeg;base64,...)

## 代理池配置

在项目根目录创建 `proxies.txt` 文件（Docker 部署通过 `-v` 挂载），每行一个代理：

```
# SOCKS5 代理，支持两种格式
ip:port
ip:port:username:password
```

- 不提供 `proxies.txt` 时自动直连
- 每次请求随机选择一个代理
## 注意事项

- Grok 不支持多轮对话历史，只能拼接历史消息
- System Prompt 会转换为 Grok 的 customPersonality 参数
- `tls-client` 库可绕过大部分 403 错误，依旧报错需要更换 IP
