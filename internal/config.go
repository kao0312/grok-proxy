package internal

import (
	"os"

	http "github.com/bogdanfinn/fhttp"
	tls_client "github.com/bogdanfinn/tls-client"
	"github.com/bogdanfinn/tls-client/profiles"
	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

type Config struct {
	Port string
}

var Cfg *Config

func LoadConfig() {
	godotenv.Load()

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	Cfg = &Config{
		Port: port,
	}
}

// 固定请求头
func SetCommonHeaders(req *http.Request) {
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Language", "zh-CN,zh;q=0.9")
	req.Header.Set("Baggage", "sentry-public_key=b311e0f2690c81f25e2c4cf6d4f7ce1c")
	req.Header.Set("Connection", "keep-alive")
	req.Header.Set("Origin", "https://grok.com")
	req.Header.Set("Priority", "u=1, i")
	req.Header.Set("Referer", "https://grok.com/")
	req.Header.Set("Sec-Ch-Ua", `"Not(A:Brand";v="99", "Google Chrome";v="133", "Chromium";v="133"`)
	req.Header.Set("Sec-Ch-Ua-Mobile", "?0")
	req.Header.Set("Sec-Ch-Ua-Platform", `"macOS"`)
	req.Header.Set("Sec-Fetch-Dest", "empty")
	req.Header.Set("Sec-Fetch-Mode", "cors")
	req.Header.Set("Sec-Fetch-Site", "same-origin")
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/133.0.0.0 Safari/537.36")
	req.Header.Set("x-statsig-id", "ZTpUeXBlRXJyb3I6IENhbm5vdCByZWFkIHByb3BlcnRpZXMgb2YgdW5kZWZpbmVkIChyZWFkaW5nICdjaGlsZE5vZGVzJyk=")
	req.Header.Set("x-xai-request-id", uuid.New().String())
}

// 聊天请求
func SetChatHeaders(req *http.Request, cookie string) {
	SetCommonHeaders(req)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Cookie", cookie)
}

// 上传请求
func SetUploadHeaders(req *http.Request, cookie string) {
	SetCommonHeaders(req)
	req.Header.Set("Content-Type", "text/plain;charset=UTF-8")
	req.Header.Set("Cookie", cookie)
}

// TLS 客户端（伪装成 Chrome 浏览器）
func GetHTTPClient() tls_client.HttpClient {
	options := []tls_client.HttpClientOption{
		tls_client.WithTimeoutSeconds(600),
		tls_client.WithClientProfile(profiles.Chrome_131),
		tls_client.WithRandomTLSExtensionOrder(), // 随机 TLS 扩展顺序，必须启用
	}

	client, err := tls_client.NewHttpClient(tls_client.NewNoopLogger(), options...)
	if err != nil {
		LogError("Failed to create TLS client: %v", err)
		return nil
	}

	return client
}
