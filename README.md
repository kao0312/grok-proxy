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

## 快速开始

### 本地运行

```bash
# 克隆项目
git clone https://github.com/kao0312/grok-proxy.git
cd grok-proxy

# 安装依赖
go mod download

# 运行服务
go run main.go
```

### Docker 部署

```bash
# 构建镜像
docker build -t grok-proxy .

# 运行容器
docker run -d -p 8080:8080 grok-proxy
```

自定义配置：

```bash
docker run -d \
  -p 8000:8080 \
  -e LOG_LEVEL=ERROR \
  grok-proxy
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

| 模型名称 | 说明 |
|----------|------|
| grok-3 | Grok 3 基础模型 |
| grok-4 | Grok 4 基础模型 |
| grok-4-fast | Grok 4 快速版本 |
| grok-4-heavy | Grok 4 高级版本 |
| grok-4.1 | Grok 4.1 标准版 |
| grok-4.1-thinking | Grok 4.1 思考模式 |

## 使用示例

### 基础对话

```bash
curl http://localhost:8000/v1/chat/completions \
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
curl http://localhost:8000/v1/models
```

## 支持的图片格式

- HTTP/HTTPS URL
- Base64 编码 (data:image/jpeg;base64,...)

## 注意事项

- Grok 不支持多轮对话历史，只能拼接历史消息
- 若上游返回图片，会**公开聊天对话**以访问图床链接
- System Prompt 会转换为 Grok 的 customPersonality 参数
- **目前官网不显示思考内容**，因此 `reasoning_content` 仅展示搜索结果（包括非推理模型）
- `tls-client` 库可绕过大部分 403 错误，依旧报错需要更换 IP
