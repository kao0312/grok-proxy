package main

import (
	"net/http"

	"grok-proxy/internal"
)

func main() {
	internal.LoadConfig()
	internal.InitLogger()

	http.HandleFunc("/v1/models", internal.HandleModels)
	http.HandleFunc("/v1/chat/completions", internal.HandleChatCompletions)

	addr := ":" + internal.Cfg.Port
	internal.LogInfo("Server starting on %s", addr)
	if err := http.ListenAndServe(addr, nil); err != nil {
		internal.LogError("Server failed: %v", err)
	}
}
