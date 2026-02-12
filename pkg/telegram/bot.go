// Package telegram provides a Telegram bot bridge using the tgbotapi library.
package telegram

import (
	"context"
	"fmt"
	"strings"
	"sync"

	"ClosedWheeler/pkg/logger"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

// Bot wraps tgbotapi.BotAPI for Telegram communication.
type Bot struct {
	api    *tgbotapi.BotAPI
	chatID int64
	logger *logger.Logger
	mu     sync.Mutex
}

// NewBot validates the token via an API call and returns a ready Bot.
// Returns (nil, nil) when token is empty (Telegram not configured).
func NewBot(token string, chatID int64, log *logger.Logger) (*Bot, error) {
	if token == "" {
		return nil, nil
	}

	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return nil, err
	}
	api.Debug = false

	return &Bot{
		api:    api,
		chatID: chatID,
		logger: log,
	}, nil
}

// Start begins receiving updates in a background goroutine and dispatches
// each update to handler. It blocks until ctx is cancelled.
func (b *Bot) Start(ctx context.Context, handler func(Update)) {
	u := tgbotapi.NewUpdate(0)
	u.Timeout = 30

	updates := b.api.GetUpdatesChan(u)

	go func() {
		for {
			select {
			case <-ctx.Done():
				b.logf("Telegram polling stopped")
				return
			case upd, ok := <-updates:
				if !ok {
					return
				}
				handler(convertUpdate(upd))
			}
		}
	}()
}

// Stop terminates the long-polling connection.
func (b *Bot) Stop() {
	b.api.StopReceivingUpdates()
}

// SendMessage sends a text message to the default chat ID.
func (b *Bot) SendMessage(text string) error {
	return b.SendMessageToChat(b.chatID, text)
}

// SendMessageToChat sends a Markdown-formatted message to a specific chat.
// Messages longer than 4000 characters are automatically split.
func (b *Bot) SendMessageToChat(chatID int64, text string) error {
	if chatID == 0 {
		return nil
	}

	parts := splitText(text, 4000)
	for _, part := range parts {
		if err := b.sendSingleMessage(chatID, part); err != nil {
			return err
		}
	}
	return nil
}

// sendSingleMessage sends one message with Markdown parse mode.
// On parse error, it retries without formatting.
func (b *Bot) sendSingleMessage(chatID int64, text string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown

	_, err := b.api.Send(msg)
	if err != nil && isParseError(err) {
		b.logf("Markdown parse error, retrying without formatting: %v", err)
		msg.ParseMode = ""
		_, err = b.api.Send(msg)
	}
	return err
}

// SendMessageWithButtons sends a message with an inline keyboard and returns
// the sent message ID (useful for later editing).
func (b *Bot) SendMessageWithButtons(chatID int64, text string, buttons [][]InlineButton) (int, error) {
	if chatID == 0 {
		return 0, nil
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	msg := tgbotapi.NewMessage(chatID, text)
	msg.ParseMode = tgbotapi.ModeMarkdown
	msg.ReplyMarkup = toInlineKeyboard(buttons)

	sent, err := b.api.Send(msg)
	if err != nil && isParseError(err) {
		b.logf("Markdown parse error (buttons), retrying without formatting: %v", err)
		msg.ParseMode = ""
		sent, err = b.api.Send(msg)
	}
	if err != nil {
		return 0, err
	}
	return sent.MessageID, nil
}

// EditMessageText edits the text of an existing message and removes its inline keyboard.
func (b *Bot) EditMessageText(chatID int64, messageID int, text string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	edit := tgbotapi.NewEditMessageText(chatID, messageID, text)
	edit.ParseMode = tgbotapi.ModeMarkdown

	_, err := b.api.Send(edit)
	if err != nil && isParseError(err) {
		edit.ParseMode = ""
		_, err = b.api.Send(edit)
	}
	return err
}

// SendChatAction sends a "typing..." indicator to the specified chat.
func (b *Bot) SendChatAction(chatID int64) error {
	action := tgbotapi.NewChatAction(chatID, tgbotapi.ChatTyping)
	_, err := b.api.Request(action)
	return err
}

// AnswerCallbackQuery acknowledges a callback query to dismiss the loading state.
func (b *Bot) AnswerCallbackQuery(callbackQueryID, text string) error {
	cb := tgbotapi.NewCallback(callbackQueryID, text)
	_, err := b.api.Request(cb)
	return err
}

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

// InlineButton represents an inline keyboard button.
type InlineButton struct {
	Text         string `json:"text"`
	CallbackData string `json:"callback_data"`
}

// Update represents a Telegram update.
type Update struct {
	UpdateID      int64          `json:"update_id"`
	Message       *Message       `json:"message,omitempty"`
	CallbackQuery *CallbackQuery `json:"callback_query,omitempty"`
}

// Message represents a Telegram message.
type Message struct {
	MessageID int    `json:"message_id"`
	Text      string `json:"text"`
	Chat      struct {
		ID int64 `json:"id"`
	} `json:"chat"`
}

// IsCommand returns true if the message text starts with '/'.
func (m *Message) IsCommand() bool {
	return m != nil && strings.HasPrefix(m.Text, "/")
}

// Command extracts the command name without the leading '/' and bot suffix.
// For "/start@mybot" it returns "start".
func (m *Message) Command() string {
	if m == nil || !strings.HasPrefix(m.Text, "/") {
		return ""
	}
	cmd := m.Text[1:]
	// Take first word only
	if idx := strings.IndexAny(cmd, " @"); idx != -1 {
		cmd = cmd[:idx]
	}
	return cmd
}

// CallbackQuery represents an incoming callback query from an inline keyboard.
type CallbackQuery struct {
	ID      string   `json:"id"`
	Data    string   `json:"data"`
	Message *Message `json:"message"`
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// convertUpdate maps a tgbotapi.Update to our Update type.
func convertUpdate(u tgbotapi.Update) Update {
	out := Update{
		UpdateID: int64(u.UpdateID),
	}

	if u.Message != nil {
		out.Message = &Message{
			MessageID: u.Message.MessageID,
			Text:      u.Message.Text,
		}
		out.Message.Chat.ID = u.Message.Chat.ID
	}

	if u.CallbackQuery != nil {
		out.CallbackQuery = &CallbackQuery{
			ID:   u.CallbackQuery.ID,
			Data: u.CallbackQuery.Data,
		}
		if u.CallbackQuery.Message != nil {
			out.CallbackQuery.Message = &Message{
				MessageID: u.CallbackQuery.Message.MessageID,
				Text:      u.CallbackQuery.Message.Text,
			}
			out.CallbackQuery.Message.Chat.ID = u.CallbackQuery.Message.Chat.ID
		}
	}

	return out
}

// toInlineKeyboard converts our InlineButton slices to tgbotapi markup.
func toInlineKeyboard(rows [][]InlineButton) tgbotapi.InlineKeyboardMarkup {
	var keyboard [][]tgbotapi.InlineKeyboardButton
	for _, row := range rows {
		var kbRow []tgbotapi.InlineKeyboardButton
		for _, btn := range row {
			kbRow = append(kbRow, tgbotapi.NewInlineKeyboardButtonData(btn.Text, btn.CallbackData))
		}
		keyboard = append(keyboard, kbRow)
	}
	return tgbotapi.InlineKeyboardMarkup{InlineKeyboard: keyboard}
}

// splitText splits text into chunks of at most maxLen characters,
// preferring to break at newlines.
func splitText(text string, maxLen int) []string {
	if len(text) <= maxLen {
		return []string{text}
	}

	var parts []string
	for len(text) > 0 {
		if len(text) <= maxLen {
			parts = append(parts, text)
			break
		}

		splitPos := maxLen
		if nl := strings.LastIndex(text[:maxLen], "\n"); nl > maxLen/2 {
			splitPos = nl + 1
		}

		parts = append(parts, text[:splitPos])
		text = text[splitPos:]
	}
	return parts
}

// isParseError returns true if the error is a Telegram parse/markdown error.
func isParseError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return strings.Contains(msg, "can't parse") ||
		strings.Contains(msg, "Bad Request: can't parse")
}

// GetBotUsername returns the bot's Telegram username (e.g. "MyAgentBot").
// Returns empty string if the bot or API is not initialized.
func (b *Bot) GetBotUsername() string {
	if b == nil || b.api == nil {
		return ""
	}
	return b.api.Self.UserName
}

// GetChatID returns the currently configured default chat ID.
func (b *Bot) GetChatID() int64 {
	return b.chatID
}

// SetChatID updates the default chat ID at runtime.
func (b *Bot) SetChatID(id int64) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.chatID = id
}

// ValidateToken checks if a bot token is valid by calling the Telegram API.
// Returns the bot username on success.
func ValidateToken(token string) (string, error) {
	if token == "" {
		return "", fmt.Errorf("token is empty")
	}
	api, err := tgbotapi.NewBotAPI(token)
	if err != nil {
		return "", err
	}
	return api.Self.UserName, nil
}

// logf writes to the logger if available.
func (b *Bot) logf(format string, args ...any) {
	if b.logger != nil {
		b.logger.Info(format, args...)
	}
}
