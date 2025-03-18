package handler

import (
	"net/http"
	"os"
	"wegram-bot-plus/core"
)

func Handler(w http.ResponseWriter, r *http.Request) {
	// 获取环境变量配置
	config := core.Config{
		Prefix:      getEnvOrDefault("PREFIX", "public"),
		SecretToken: getEnvOrDefault("SECRET_TOKEN", ""),
	}

	// 调用核心处理逻辑
	response, err := core.HandleRequest(r, config)
	if err != nil {
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// 设置响应头
	for key, values := range response.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}

	// 设置状态码并写入响应体
	w.WriteHeader(response.StatusCode)
	w.Write(response.Body)
}

// 获取环境变量，如不存在则返回默认值
func getEnvOrDefault(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}
