package telegram

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"io"
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sublink/models"
	nodeparser "sublink/node"
	"sync"
	"time"
)

type Manager struct {
	mu     sync.Mutex
	cancel context.CancelFunc
	bot    *Bot
}

type Bot struct {
	config models.TelegramConfig
	client *http.Client
	mu     sync.Mutex
	states map[int64]string
}

type updateResponse struct {
	OK          bool     `json:"ok"`
	Result      []Update `json:"result"`
	Description string   `json:"description"`
}

type apiResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description"`
}

type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message"`
	CallbackQuery *CallbackQuery `json:"callback_query"`
}

type Message struct {
	MessageID int64  `json:"message_id"`
	Chat      Chat   `json:"chat"`
	Text      string `json:"text"`
}

type Chat struct {
	ID int64 `json:"id"`
}

type CallbackQuery struct {
	ID      string   `json:"id"`
	Data    string   `json:"data"`
	Message *Message `json:"message"`
}

type inlineKeyboard struct {
	InlineKeyboard [][]inlineButton `json:"inline_keyboard"`
}

type inlineButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

type replyKeyboard struct {
	Keyboard        [][]replyButton `json:"keyboard"`
	ResizeKeyboard  bool            `json:"resize_keyboard"`
	OneTimeKeyboard bool            `json:"one_time_keyboard"`
	IsPersistent    bool            `json:"is_persistent"`
}

type replyButton struct {
	Text string `json:"text"`
}

var DefaultManager = &Manager{}

func (m *Manager) Reload(config models.TelegramConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.cancel != nil {
		m.cancel()
		m.cancel = nil
		m.bot = nil
	}
	if !config.Enabled {
		return nil
	}
	if strings.TrimSpace(config.Token) == "" {
		return errors.New("Telegram Token 不能为空")
	}
	if len(parseAdminIDs(config.AdminChatIDs)) == 0 {
		return errors.New("至少配置一个管理员聊天 ID")
	}

	bot := NewBot(config)
	if err := bot.GetMe(); err != nil {
		return err
	}
	ctx, cancel := context.WithCancel(context.Background())
	m.cancel = cancel
	m.bot = bot
	go bot.Run(ctx)
	return nil
}

func (m *Manager) TestMessage(config models.TelegramConfig) error {
	if strings.TrimSpace(config.Token) == "" {
		return errors.New("Telegram Token 不能为空")
	}
	ids := parseAdminIDs(config.AdminChatIDs)
	if len(ids) == 0 {
		return errors.New("至少配置一个管理员聊天 ID")
	}
	bot := NewBot(config)
	if err := bot.GetMe(); err != nil {
		return err
	}
	for _, id := range ids {
		if err := bot.SendHTML(id, "✅ <b>SublinkX Telegram 机器人连接测试成功</b>", mainReplyKeyboard()); err != nil {
			return err
		}
	}
	return nil
}

func NewBot(config models.TelegramConfig) *Bot {
	if strings.TrimSpace(config.APIBaseURL) == "" {
		config.APIBaseURL = "https://api.telegram.org"
	}
	return &Bot{
		config: config,
		client: &http.Client{Timeout: 40 * time.Second},
		states: make(map[int64]string),
	}
}

func (b *Bot) Run(ctx context.Context) {
	var offset int64
	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		updates, err := b.getUpdates(ctx, offset)
		if err != nil {
			if ctx.Err() != nil {
				return
			}
			log.Println("Telegram 获取消息失败:", err)
			time.Sleep(3 * time.Second)
			continue
		}
		for _, update := range updates {
			offset = update.UpdateID + 1
			b.handleUpdate(update)
		}
	}
}

func (b *Bot) GetMe() error {
	var response apiResponse
	if err := b.call("getMe", nil, &response); err != nil {
		return err
	}
	if !response.OK {
		return errors.New(response.Description)
	}
	return nil
}

func (b *Bot) getUpdates(ctx context.Context, offset int64) ([]Update, error) {
	payload := map[string]any{
		"offset":          offset,
		"timeout":         30,
		"allowed_updates": []string{"message", "callback_query"},
	}
	var response updateResponse
	if err := b.callContext(ctx, "getUpdates", payload, &response); err != nil {
		return nil, err
	}
	if !response.OK {
		return nil, errors.New(response.Description)
	}
	return response.Result, nil
}

func (b *Bot) handleUpdate(update Update) {
	if update.CallbackQuery != nil && update.CallbackQuery.Message != nil {
		query := update.CallbackQuery
		chatID := query.Message.Chat.ID
		if !b.isAdmin(chatID) {
			_ = b.answerCallback(query.ID, "无权操作")
			return
		}
		_ = b.answerCallback(query.ID, "")
		b.handleCallback(chatID, query.Data)
		return
	}
	if update.Message == nil || strings.TrimSpace(update.Message.Text) == "" {
		return
	}
	chatID := update.Message.Chat.ID
	if !b.isAdmin(chatID) {
		_ = b.SendHTML(chatID, fmt.Sprintf("⛔ <b>当前 Chat ID 未授权</b>\n\n<code>%d</code>", chatID), nil)
		return
	}
	b.handleMessage(chatID, strings.TrimSpace(update.Message.Text))
}

func (b *Bot) handleMessage(chatID int64, text string) {
	if text == "/cancel" {
		b.setState(chatID, "")
		_ = b.SendHTML(chatID, "已取消当前操作。", mainReplyKeyboard())
		return
	}

	state := b.getState(chatID)
	if state == "add_node" && !strings.HasPrefix(text, "/") {
		b.addNode(chatID, text)
		return
	}

	switch normalizeCommand(text) {
	case "/start", "/menu":
		_ = b.SendHTML(chatID, welcomeMessage(), mainReplyKeyboard())
	case "/id":
		_ = b.SendHTML(chatID, fmt.Sprintf("🪪 <b>当前 Chat ID</b>\n\n<code>%d</code>", chatID), mainReplyKeyboard())
	case "/nodes", "📋 节点列表":
		b.sendNodeList(chatID)
	case "/subs", "🧾 订阅列表":
		b.sendSubscriptionList(chatID)
	case "/addnode", "➕ 添加节点":
		b.promptAddNode(chatID)
	case "🗑 删除节点":
		b.sendDeleteNodeList(chatID)
	default:
		_ = b.SendHTML(chatID, "我还不认识这个指令。请使用输入框上方的菜单操作。", mainReplyKeyboard())
	}
}

func (b *Bot) handleCallback(chatID int64, data string) {
	switch {
	case data == "nodes":
		b.sendNodeList(chatID)
	case data == "subs":
		b.sendSubscriptionList(chatID)
	case data == "add_node":
		b.promptAddNode(chatID)
	case data == "delete_nodes":
		b.sendDeleteNodeList(chatID)
	case strings.HasPrefix(data, "delete_node:"):
		id := strings.TrimPrefix(data, "delete_node:")
		keyboard := inlineKeyboard{InlineKeyboard: [][]inlineButton{
			{{Text: "确认删除", CallbackData: "confirm_delete:" + id}, {Text: "取消", CallbackData: "nodes"}},
		}}
		_ = b.SendHTML(chatID, "⚠️ <b>确认删除这个节点？</b>\n\n删除后无法恢复。", keyboard)
	case strings.HasPrefix(data, "confirm_delete:"):
		b.deleteNode(chatID, strings.TrimPrefix(data, "confirm_delete:"))
	case data == "menu":
		_ = b.SendHTML(chatID, welcomeMessage(), mainReplyKeyboard())
	}
}

func (b *Bot) sendNodeList(chatID int64) {
	nodes, err := models.GetNodeList()
	if err != nil {
		_ = b.SendHTML(chatID, "读取节点失败："+escapeHTML(err.Error()), mainReplyKeyboard())
		return
	}

	var text strings.Builder
	fmt.Fprintf(&text, "📋 <b>节点列表</b>\n共 <b>%d</b> 个节点\n\n", len(nodes))
	if len(nodes) == 0 {
		text.WriteString("当前还没有节点。可以点击输入框上方的「➕ 添加节点」开始添加。")
	}
	for _, item := range nodes {
		groups := make([]string, 0, len(item.GroupNodes))
		for _, group := range item.GroupNodes {
			groups = append(groups, group.Name)
		}
		protocol := "unknown"
		if parsed, err := url.Parse(item.Link); err == nil && parsed.Scheme != "" {
			protocol = parsed.Scheme
		}

		fmt.Fprintf(&text, "▫️ <b>#%d %s</b>\n", item.ID, escapeHTML(item.Name))
		fmt.Fprintf(&text, "协议：<code>%s</code>\n", escapeHTML(strings.ToUpper(protocol)))
		if len(groups) > 0 {
			fmt.Fprintf(&text, "分组：%s\n", escapeHTML(strings.Join(groups, " / ")))
		} else {
			text.WriteString("分组：未分组\n")
		}
		if protocol == "ss" {
			fmt.Fprintf(&text, "SS 链接：\n<code>%s</code>\n", escapeHTML(item.Link))
		}
		text.WriteString("\n")
		if text.Len() > 3500 {
			text.WriteString("节点较多，当前消息仅展示前半部分。")
			break
		}
	}

	keyboard := inlineKeyboard{InlineKeyboard: [][]inlineButton{
		{{Text: "➕ 添加节点", CallbackData: "add_node"}, {Text: "🗑 删除节点", CallbackData: "delete_nodes"}},
		{{Text: "🔄 刷新列表", CallbackData: "nodes"}, {Text: "🏠 主菜单", CallbackData: "menu"}},
	}}
	_ = b.SendHTML(chatID, text.String(), keyboard)
}

func (b *Bot) sendDeleteNodeList(chatID int64) {
	nodes, err := models.GetNodeList()
	if err != nil {
		_ = b.SendHTML(chatID, "读取节点失败："+escapeHTML(err.Error()), mainReplyKeyboard())
		return
	}
	rows := make([][]inlineButton, 0, len(nodes)+1)
	for index, item := range nodes {
		if index >= 40 {
			break
		}
		rows = append(rows, []inlineButton{{
			Text:         fmt.Sprintf("#%d %s", item.ID, item.Name),
			CallbackData: fmt.Sprintf("delete_node:%d", item.ID),
		}})
	}
	rows = append(rows, []inlineButton{{Text: "返回节点列表", CallbackData: "nodes"}})
	_ = b.SendHTML(chatID, "🗑 <b>选择要删除的节点</b>\n\n最多显示前 40 个节点。", inlineKeyboard{InlineKeyboard: rows})
}

func (b *Bot) sendSubscriptionList(chatID int64) {
	var sub models.Subcription
	subs, err := sub.List()
	if err != nil {
		_ = b.SendHTML(chatID, "读取订阅失败："+escapeHTML(err.Error()), mainReplyKeyboard())
		return
	}

	var text strings.Builder
	fmt.Fprintf(&text, "🧾 <b>订阅列表</b>\n共 <b>%d</b> 个订阅\n\n", len(subs))
	if len(subs) == 0 {
		text.WriteString("当前还没有订阅。")
	}
	for index, item := range subs {
		fmt.Fprintf(&text, "▫️ <b>%d. %s</b>\n节点数：<code>%d</code>\n\n", index+1, escapeHTML(item.Name), len(item.Nodes))
		if text.Len() > 3500 {
			text.WriteString("订阅较多，当前消息仅展示前半部分。")
			break
		}
	}
	keyboard := inlineKeyboard{InlineKeyboard: [][]inlineButton{
		{{Text: "🔄 刷新订阅", CallbackData: "subs"}, {Text: "🏠 主菜单", CallbackData: "menu"}},
	}}
	_ = b.SendHTML(chatID, text.String(), keyboard)
}

func (b *Bot) promptAddNode(chatID int64) {
	b.setState(chatID, "add_node")
	_ = b.SendHTML(chatID, "➕ <b>添加节点</b>\n\n请发送节点链接。\n如需分组，请在第二行填写分组名，多个分组用逗号分隔。\n\n发送 <code>/cancel</code> 可取消当前操作。", mainReplyKeyboard())
}

func (b *Bot) addNode(chatID int64, input string) {
	lines := strings.Split(input, "\n")
	link := strings.TrimSpace(lines[0])
	if !strings.Contains(link, "://") {
		_ = b.SendHTML(chatID, "节点链接格式不正确，请重新发送，或使用 <code>/cancel</code> 取消。", mainReplyKeyboard())
		return
	}
	groupText := ""
	if len(lines) > 1 {
		groupText = strings.TrimSpace(lines[1])
	}
	item := models.Node{Link: link}
	name, err := decodeNodeName(link)
	if err != nil {
		_ = b.SendHTML(chatID, "解析节点失败："+escapeHTML(err.Error()), mainReplyKeyboard())
		return
	}
	item.Name = name
	if err := item.Add(); err != nil {
		_ = b.SendHTML(chatID, "添加节点失败："+escapeHTML(err.Error()), mainReplyKeyboard())
		return
	}
	if groupText != "" {
		for _, groupName := range strings.Split(groupText, ",") {
			groupName = strings.TrimSpace(groupName)
			if groupName == "" {
				continue
			}
			group := &models.GroupNode{Name: groupName}
			if err := group.Add(); err != nil {
				_ = b.SendHTML(chatID, "节点已添加，但创建分组失败："+escapeHTML(err.Error()), mainReplyKeyboard())
				return
			}
			if err := group.Ass(&item); err != nil {
				_ = b.SendHTML(chatID, "节点已添加，但关联分组失败："+escapeHTML(err.Error()), mainReplyKeyboard())
				return
			}
		}
	}
	b.setState(chatID, "")
	_ = b.SendHTML(chatID, "✅ <b>节点添加成功</b>\n\n"+escapeHTML(item.Name), mainReplyKeyboard())
}

func (b *Bot) deleteNode(chatID int64, idText string) {
	id, err := strconv.Atoi(idText)
	if err != nil {
		_ = b.SendHTML(chatID, "节点 ID 格式不正确。", mainReplyKeyboard())
		return
	}
	item := models.Node{ID: id}
	if err := models.DB.First(&item, id).Error; err != nil {
		_ = b.SendHTML(chatID, "节点不存在或已删除。", mainReplyKeyboard())
		return
	}
	name := item.Name
	if err := item.Del(); err != nil {
		_ = b.SendHTML(chatID, "删除节点失败："+escapeHTML(err.Error()), mainReplyKeyboard())
		return
	}
	_ = b.SendHTML(chatID, "✅ 节点已删除："+escapeHTML(name), mainReplyKeyboard())
}

func decodeNodeName(link string) (string, error) {
	parsed, err := url.Parse(link)
	if err != nil {
		return "", err
	}
	switch parsed.Scheme {
	case "ss":
		item, err := nodeparser.DecodeSSURL(link)
		return item.Name, err
	case "ssr":
		item, err := nodeparser.DecodeSSRURL(link)
		return item.Qurey.Remarks, err
	case "trojan":
		item, err := nodeparser.DecodeTrojanURL(link)
		return item.Name, err
	case "vmess":
		item, err := nodeparser.DecodeVMESSURL(link)
		return item.Ps, err
	case "vless":
		item, err := nodeparser.DecodeVLESSURL(link)
		return item.Name, err
	case "hy", "hysteria":
		item, err := nodeparser.DecodeHYURL(link)
		return item.Name, err
	case "hy2", "hysteria2":
		item, err := nodeparser.DecodeHY2URL(link)
		return item.Name, err
	case "tuic":
		item, err := nodeparser.DecodeTuicURL(link)
		return item.Name, err
	default:
		return "", fmt.Errorf("暂不支持 %s 协议", parsed.Scheme)
	}
}

func (b *Bot) SendMessage(chatID int64, text string, keyboard any) error {
	payload := map[string]any{"chat_id": chatID, "text": text}
	if keyboard != nil {
		payload["reply_markup"] = keyboard
	}
	var response apiResponse
	if err := b.call("sendMessage", payload, &response); err != nil {
		return err
	}
	if !response.OK {
		return errors.New(response.Description)
	}
	return nil
}

func (b *Bot) SendHTML(chatID int64, text string, keyboard any) error {
	payload := map[string]any{
		"chat_id":                  chatID,
		"text":                     text,
		"parse_mode":               "HTML",
		"disable_web_page_preview": true,
	}
	if keyboard != nil {
		payload["reply_markup"] = keyboard
	}
	var response apiResponse
	if err := b.call("sendMessage", payload, &response); err != nil {
		return err
	}
	if !response.OK {
		return errors.New(response.Description)
	}
	return nil
}

func (b *Bot) answerCallback(callbackID string, text string) error {
	payload := map[string]any{"callback_query_id": callbackID}
	if text != "" {
		payload["text"] = text
	}
	return b.call("answerCallbackQuery", payload, &apiResponse{})
}

func (b *Bot) call(method string, payload any, target any) error {
	return b.callContext(context.Background(), method, payload, target)
}

func (b *Bot) callContext(ctx context.Context, method string, payload any, target any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	endpoint := strings.TrimRight(b.config.APIBaseURL, "/") + "/bot" + b.config.Token + "/" + method
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	response, err := b.client.Do(request)
	if err != nil {
		return errors.New("无法连接 Telegram API")
	}
	defer response.Body.Close()
	responseBody, err := io.ReadAll(io.LimitReader(response.Body, 1<<20))
	if err != nil {
		return err
	}
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return fmt.Errorf("Telegram API 返回 %d: %s", response.StatusCode, string(responseBody))
	}
	if err := json.Unmarshal(responseBody, target); err != nil {
		return err
	}
	return nil
}

func (b *Bot) isAdmin(chatID int64) bool {
	for _, id := range parseAdminIDs(b.config.AdminChatIDs) {
		if id == chatID {
			return true
		}
	}
	return false
}

func parseAdminIDs(value string) []int64 {
	matches := regexp.MustCompile(`-?\d+`).FindAllString(value, -1)
	ids := make([]int64, 0, len(matches))
	for _, match := range matches {
		id, err := strconv.ParseInt(match, 10, 64)
		if err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

func (b *Bot) getState(chatID int64) string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.states[chatID]
}

func (b *Bot) setState(chatID int64, state string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	if state == "" {
		delete(b.states, chatID)
		return
	}
	b.states[chatID] = state
}

func mainReplyKeyboard() replyKeyboard {
	return replyKeyboard{
		Keyboard: [][]replyButton{
			{{Text: "📋 节点列表"}, {Text: "🧾 订阅列表"}},
			{{Text: "➕ 添加节点"}, {Text: "🗑 删除节点"}},
		},
		ResizeKeyboard:  true,
		IsPersistent:    true,
		OneTimeKeyboard: false,
	}
}

func normalizeCommand(text string) string {
	command := strings.Split(strings.TrimSpace(text), " ")[0]
	switch command {
	case "节点列表":
		return "📋 节点列表"
	case "订阅列表":
		return "🧾 订阅列表"
	case "添加节点":
		return "➕ 添加节点"
	case "删除节点":
		return "🗑 删除节点"
	default:
		return command
	}
}

func welcomeMessage() string {
	return "✨ <b>SublinkX 管理机器人</b>\n\n输入框上方的四个常用操作已固定显示：\n<code>节点列表 / 订阅列表 / 添加节点 / 删除节点</code>"
}

func escapeHTML(value string) string {
	return html.EscapeString(value)
}
