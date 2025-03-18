package core

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
)

// Config 存储应用配置
type Config struct {
	Prefix      string
	SecretToken string
}

// Response 包含处理结果
type Response struct {
	StatusCode int
	Header     http.Header
	Body       []byte
}

// 验证密钥是否符合安全标准
func ValidateSecretToken(token string) bool {
	if len(token) <= 15 {
		return false
	}

	hasUpper := regexp.MustCompile(`[A-Z]`).MatchString(token)
	hasLower := regexp.MustCompile(`[a-z]`).MatchString(token)
	hasDigit := regexp.MustCompile(`[0-9]`).MatchString(token)

	return hasUpper && hasLower && hasDigit
}

// 创建JSON响应
func JsonResponse(data interface{}, status int) (*Response, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}

	return &Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       jsonData,
	}, nil
}

// 发送请求到Telegram API
func PostToTelegramApi(token string, method string, body interface{}) (*http.Response, error) {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}

	url := fmt.Sprintf("https://api.telegram.org/bot%s/%s", token, method)
	req, err := http.NewRequest("POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	return client.Do(req)
}

// 处理机器人安装
func HandleInstall(r *http.Request, ownerUid string, botToken string, prefix string, secretToken string) (*Response, error) {
	if !ValidateSecretToken(secretToken) {
		return JsonResponse(map[string]interface{}{
			"success": false,
			"message": "Secret token must be at least 16 characters and contain uppercase letters, lowercase letters, and numbers.",
		}, 400)
	}

	url := fmt.Sprintf("%s://%s", getProtocol(r), r.Host)
	webhookUrl := fmt.Sprintf("%s/%s/webhook/%s/%s", url, prefix, ownerUid, botToken)

	webhookData := map[string]interface{}{
		"url":             webhookUrl,
		"allowed_updates": []string{"message"},
		"secret_token":    secretToken,
	}

	resp, err := PostToTelegramApi(botToken, "setWebhook", webhookData)
	if err != nil {
		return JsonResponse(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Error installing webhook: %s", err.Error()),
		}, 500)
	}
	defer resp.Body.Close()

	// 解析响应
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return JsonResponse(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Error parsing response: %s", err.Error()),
		}, 500)
	}

	if ok, _ := result["ok"].(bool); ok {
		return JsonResponse(map[string]interface{}{
			"success": true,
			"message": "Webhook successfully installed.",
		}, 200)
	}

	description := "Unknown error"
	if desc, ok := result["description"].(string); ok {
		description = desc
	}

	return JsonResponse(map[string]interface{}{
		"success": false,
		"message": fmt.Sprintf("Failed to install webhook: %s", description),
	}, 400)
}

// 处理机器人卸载
func HandleUninstall(botToken string, secretToken string) (*Response, error) {
	if !ValidateSecretToken(secretToken) {
		return JsonResponse(map[string]interface{}{
			"success": false,
			"message": "Secret token must be at least 16 characters and contain uppercase letters, lowercase letters, and numbers.",
		}, 400)
	}

	resp, err := PostToTelegramApi(botToken, "deleteWebhook", map[string]interface{}{})
	if err != nil {
		return JsonResponse(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Error uninstalling webhook: %s", err.Error()),
		}, 500)
	}
	defer resp.Body.Close()

	// 解析响应
	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return JsonResponse(map[string]interface{}{
			"success": false,
			"message": fmt.Sprintf("Error parsing response: %s", err.Error()),
		}, 500)
	}

	if ok, _ := result["ok"].(bool); ok {
		return JsonResponse(map[string]interface{}{
			"success": true,
			"message": "Webhook successfully uninstalled.",
		}, 200)
	}

	description := "Unknown error"
	if desc, ok := result["description"].(string); ok {
		description = desc
	}

	return JsonResponse(map[string]interface{}{
		"success": false,
		"message": fmt.Sprintf("Failed to uninstall webhook: %s", description),
	}, 400)
}

// 处理Webhook请求
func HandleWebhook(r *http.Request, ownerUid string, botToken string, secretToken string) (*Response, error) {
	// 验证密钥
	if secretToken != r.Header.Get("X-Telegram-Bot-Api-Secret-Token") {
		return &Response{
			StatusCode: 401,
			Header:     http.Header{"Content-Type": []string{"text/plain"}},
			Body:       []byte("Unauthorized"),
		}, nil
	}

	// 解析请求体
	var update map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&update); err != nil {
		return &Response{
			StatusCode: 500,
			Header:     http.Header{"Content-Type": []string{"text/plain"}},
			Body:       []byte("Internal Server Error"),
		}, nil
	}

	// 检查是否有消息
	message, ok := update["message"].(map[string]interface{})
	if !ok {
		return &Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"text/plain"}},
			Body:       []byte("OK"),
		}, nil
	}

	// 获取回复消息
	reply, hasReply := message["reply_to_message"].(map[string]interface{})

	// 获取聊天ID
	chat, chatOk := message["chat"].(map[string]interface{})
	if !chatOk {
		return &Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"text/plain"}},
			Body:       []byte("OK"),
		}, nil
	}

	chatIdFloat, chatIdOk := chat["id"].(float64)
	chatId := strconv.FormatInt(int64(chatIdFloat), 10)

	// 处理机器人所有者的回复消息
	if hasReply && chatIdOk && chatId == ownerUid {
		// 获取回复标记
		if replyMarkup, hasRM := reply["reply_markup"].(map[string]interface{}); hasRM {
			if inlineKeyboard, hasIK := replyMarkup["inline_keyboard"].([]interface{}); hasIK && len(inlineKeyboard) > 0 {
				if row, ok := inlineKeyboard[0].([]interface{}); ok && len(row) > 0 {
					if button, ok := row[0].(map[string]interface{}); ok {
						var senderUid string

						// 尝试从回调数据获取发送者ID
						if callbackData, hasCallback := button["callback_data"].(string); hasCallback {
							senderUid = callbackData
						} else if urlStr, hasUrl := button["url"].(string); hasUrl && strings.HasPrefix(urlStr, "tg://user?id=") {
							// 如果没有回调数据，尝试从URL获取
							senderUid = strings.TrimPrefix(urlStr, "tg://user?id=")
						}

						// 如果找到发送者ID，转发消息给他
						if senderUid != "" {
							senderIdInt, err := strconv.ParseInt(senderUid, 10, 64)
							if err == nil {
								_, err := PostToTelegramApi(botToken, "copyMessage", map[string]interface{}{
									"chat_id":      senderIdInt,
									"from_chat_id": chatIdFloat,
									"message_id":   message["message_id"],
								})

								if err != nil {
									fmt.Printf("Error forwarding message: %s\n", err.Error())
								}
							}
						}
					}
				}
			}
		}

		return &Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"text/plain"}},
			Body:       []byte("OK"),
		}, nil
	}

	// 处理 /start 命令
	if text, ok := message["text"].(string); ok && text == "/start" {
		return &Response{
			StatusCode: 200,
			Header:     http.Header{"Content-Type": []string{"text/plain"}},
			Body:       []byte("OK"),
		}, nil
	}

	// 处理用户消息
	sender := chat
	senderUidFloat, _ := sender["id"].(float64)
	senderUid := strconv.FormatInt(int64(senderUidFloat), 10)

	var senderName string
	if username, hasUsername := sender["username"].(string); hasUsername {
		senderName = "@" + username
	} else {
		var nameParts []string
		if firstName, hasFirst := sender["first_name"].(string); hasFirst {
			nameParts = append(nameParts, firstName)
		}
		if lastName, hasLast := sender["last_name"].(string); hasLast {
			nameParts = append(nameParts, lastName)
		}
		senderName = strings.Join(nameParts, " ")
	}

	// 尝试使用URL版按钮
	inlineKeyboard := []map[string]interface{}{
		{
			"text": fmt.Sprintf("🔓 From: %s (%s)", senderName, senderUid),
			"url":  fmt.Sprintf("tg://user?id=%s", senderUid),
		},
	}

	response, err := PostToTelegramApi(botToken, "copyMessage", map[string]interface{}{
		"chat_id":      ownerUid,
		"from_chat_id": int64(senderUidFloat),
		"message_id":   message["message_id"],
		"reply_markup": map[string]interface{}{
			"inline_keyboard": [][]map[string]interface{}{inlineKeyboard},
		},
	})

	// 如果URL版失败，尝试使用callback_data版本
	if err != nil || response.StatusCode != 200 {
		inlineKeyboard = []map[string]interface{}{
			{
				"text":          fmt.Sprintf("🔏 From: %s (%s)", senderName, senderUid),
				"callback_data": senderUid,
			},
		}

		_, _ = PostToTelegramApi(botToken, "copyMessage", map[string]interface{}{
			"chat_id":      ownerUid,
			"from_chat_id": int64(senderUidFloat),
			"message_id":   message["message_id"],
			"reply_markup": map[string]interface{}{
				"inline_keyboard": [][]map[string]interface{}{inlineKeyboard},
			},
		})
	}

	return &Response{
		StatusCode: 200,
		Header:     http.Header{"Content-Type": []string{"text/plain"}},
		Body:       []byte("OK"),
	}, nil
}

// 主请求处理函数
func HandleRequest(r *http.Request, config Config) (*Response, error) {
	prefix := config.Prefix
	secretToken := config.SecretToken
	path := r.URL.Path

	// 定义路由模式
	installPattern := regexp.MustCompile(fmt.Sprintf(`^/%s/install/([^/]+)/([^/]+)$`, prefix))
	uninstallPattern := regexp.MustCompile(fmt.Sprintf(`^/%s/uninstall/([^/]+)$`, prefix))
	webhookPattern := regexp.MustCompile(fmt.Sprintf(`^/%s/webhook/([^/]+)/([^/]+)$`, prefix))

	// 路由匹配
	if match := installPattern.FindStringSubmatch(path); match != nil {
		return HandleInstall(r, match[1], match[2], prefix, secretToken)
	}

	if match := uninstallPattern.FindStringSubmatch(path); match != nil {
		return HandleUninstall(match[1], secretToken)
	}

	if match := webhookPattern.FindStringSubmatch(path); match != nil {
		return HandleWebhook(r, match[1], match[2], secretToken)
	}

	// 如果没有匹配的路由，返回404
	return &Response{
		StatusCode: 404,
		Header:     http.Header{"Content-Type": []string{"text/plain"}},
		Body:       []byte("Not Found"),
	}, nil
}

// 获取请求协议（http或https）
func getProtocol(r *http.Request) string {
	if r.TLS != nil {
		return "https"
	}
	if proto := r.Header.Get("X-Forwarded-Proto"); proto != "" {
		return proto
	}
	return "http"
}
