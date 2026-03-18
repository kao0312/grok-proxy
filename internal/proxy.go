package internal

import (
	"bufio"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
)

var (
	proxies []string
	proxyMu sync.RWMutex
)

// LoadProxies 从 proxies.txt 文件加载代理列表
// 格式: ip:port:username:password 或 ip:port
func LoadProxies(path string) {
	file, err := os.Open(path)
	if err != nil {
		LogInfo("No proxies.txt found, running without proxy")
		return
	}
	defer file.Close()

	var loaded []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		loaded = append(loaded, line)
	}

	proxyMu.Lock()
	proxies = loaded
	proxyMu.Unlock()

	LogInfo("Loaded %d proxies from %s", len(loaded), path)
}

// getRandomProxyUrl 随机返回一个代理 URL（socks5://[user:pass@]ip:port）
// 无代理时返回空字符串
func getRandomProxyUrl() string {
	proxyMu.RLock()
	defer proxyMu.RUnlock()

	if len(proxies) == 0 {
		return ""
	}

	line := proxies[rand.Intn(len(proxies))]
	parts := strings.Split(line, ":")

	switch len(parts) {
	case 2:
		return fmt.Sprintf("socks5://%s:%s", parts[0], parts[1])
	case 4:
		return fmt.Sprintf("socks5://%s:%s@%s:%s", parts[2], parts[3], parts[0], parts[1])
	default:
		LogWarn("Invalid proxy format: %s", line)
		return ""
	}
}
