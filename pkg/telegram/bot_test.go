package telegram

import (
	"fmt"
	"testing"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

func TestNewBot_EmptyToken(t *testing.T) {
	bot, err := NewBot("", 123, nil)
	if err != nil {
		t.Fatalf("expected nil error for empty token, got: %v", err)
	}
	if bot != nil {
		t.Fatal("expected nil bot for empty token")
	}
}

func TestConvertUpdate_Message(t *testing.T) {
	raw := tgbotapi.Update{
		UpdateID: 42,
		Message: &tgbotapi.Message{
			MessageID: 10,
			Text:      "hello",
			Chat:      &tgbotapi.Chat{ID: 999},
		},
	}

	u := convertUpdate(raw)

	if u.UpdateID != 42 {
		t.Errorf("UpdateID = %d, want 42", u.UpdateID)
	}
	if u.Message == nil {
		t.Fatal("expected non-nil Message")
	}
	if u.Message.MessageID != 10 {
		t.Errorf("MessageID = %d, want 10", u.Message.MessageID)
	}
	if u.Message.Text != "hello" {
		t.Errorf("Text = %q, want %q", u.Message.Text, "hello")
	}
	if u.Message.Chat.ID != 999 {
		t.Errorf("Chat.ID = %d, want 999", u.Message.Chat.ID)
	}
	if u.CallbackQuery != nil {
		t.Error("expected nil CallbackQuery")
	}
}

func TestConvertUpdate_CallbackQuery(t *testing.T) {
	raw := tgbotapi.Update{
		UpdateID: 7,
		CallbackQuery: &tgbotapi.CallbackQuery{
			ID:   "cb-123",
			Data: "approve",
			Message: &tgbotapi.Message{
				MessageID: 55,
				Text:      "Approve this?",
				Chat:      &tgbotapi.Chat{ID: 100},
			},
		},
	}

	u := convertUpdate(raw)

	if u.CallbackQuery == nil {
		t.Fatal("expected non-nil CallbackQuery")
	}
	if u.CallbackQuery.ID != "cb-123" {
		t.Errorf("ID = %q, want %q", u.CallbackQuery.ID, "cb-123")
	}
	if u.CallbackQuery.Data != "approve" {
		t.Errorf("Data = %q, want %q", u.CallbackQuery.Data, "approve")
	}
	if u.CallbackQuery.Message == nil {
		t.Fatal("expected non-nil CallbackQuery.Message")
	}
	if u.CallbackQuery.Message.MessageID != 55 {
		t.Errorf("MessageID = %d, want 55", u.CallbackQuery.Message.MessageID)
	}
	if u.CallbackQuery.Message.Chat.ID != 100 {
		t.Errorf("Chat.ID = %d, want 100", u.CallbackQuery.Message.Chat.ID)
	}
}

func TestSplitText(t *testing.T) {
	long := make([]byte, 9000)
	for i := range long {
		long[i] = 'a'
	}
	parts := splitText(string(long), 4000)
	if len(parts) < 2 {
		t.Fatalf("expected at least 2 parts, got %d", len(parts))
	}
	for i, p := range parts {
		if len(p) > 4000 {
			t.Errorf("part %d has length %d, exceeds 4000", i, len(p))
		}
	}
}

func TestSplitText_Short(t *testing.T) {
	parts := splitText("short message", 4000)
	if len(parts) != 1 {
		t.Fatalf("expected 1 part, got %d", len(parts))
	}
	if parts[0] != "short message" {
		t.Errorf("got %q, want %q", parts[0], "short message")
	}
}

func TestSplitText_PrefersNewline(t *testing.T) {
	// Build text: 3000 chars + newline + 2000 chars = 5001 chars
	a := make([]byte, 3000)
	for i := range a {
		a[i] = 'x'
	}
	b := make([]byte, 2000)
	for i := range b {
		b[i] = 'y'
	}
	text := string(a) + "\n" + string(b)

	parts := splitText(text, 4000)
	if len(parts) != 2 {
		t.Fatalf("expected 2 parts, got %d", len(parts))
	}
	// First part should split at the newline (3000 chars + newline = 3001)
	if len(parts[0]) != 3001 {
		t.Errorf("first part length = %d, want 3001 (split at newline)", len(parts[0]))
	}
}

func TestMessageIsCommand(t *testing.T) {
	tests := []struct {
		text string
		want bool
	}{
		{"/start", true},
		{"/help args", true},
		{"hello", false},
		{"", false},
	}
	for _, tt := range tests {
		m := &Message{Text: tt.text}
		if got := m.IsCommand(); got != tt.want {
			t.Errorf("IsCommand(%q) = %v, want %v", tt.text, got, tt.want)
		}
	}
}

func TestMessageIsCommand_Nil(t *testing.T) {
	var m *Message
	if m.IsCommand() {
		t.Error("expected nil message IsCommand to be false")
	}
}

func TestMessageCommand(t *testing.T) {
	tests := []struct {
		text string
		want string
	}{
		{"/start", "start"},
		{"/help args", "help"},
		{"/start@mybot", "start"},
		{"hello", ""},
		{"", ""},
	}
	for _, tt := range tests {
		m := &Message{Text: tt.text}
		if got := m.Command(); got != tt.want {
			t.Errorf("Command(%q) = %q, want %q", tt.text, got, tt.want)
		}
	}
}

func TestMessageCommand_Nil(t *testing.T) {
	var m *Message
	if got := m.Command(); got != "" {
		t.Errorf("expected empty command for nil message, got %q", got)
	}
}

func TestValidateToken_Empty(t *testing.T) {
	name, err := ValidateToken("")
	if err == nil {
		t.Fatal("expected error for empty token")
	}
	if name != "" {
		t.Errorf("expected empty name, got %q", name)
	}
}

func TestSetChatID(t *testing.T) {
	// Create a bot with a minimal struct (no API call needed for SetChatID/GetChatID)
	b := &Bot{chatID: 100}

	if got := b.GetChatID(); got != 100 {
		t.Errorf("GetChatID() = %d, want 100", got)
	}

	b.SetChatID(999)
	if got := b.GetChatID(); got != 999 {
		t.Errorf("after SetChatID(999), GetChatID() = %d, want 999", got)
	}
}

func TestGetBotUsername(t *testing.T) {
	// GetBotUsername reads from api.Self.UserName — we need a Bot with a non-nil api.
	// Since NewBotAPI requires a real token, we test indirectly via the struct.
	// This test simply verifies the method doesn't panic with a zero-value api.Self.
	b := &Bot{
		api: &tgbotapi.BotAPI{
			Self: tgbotapi.User{UserName: "TestBot"},
		},
	}
	if got := b.GetBotUsername(); got != "TestBot" {
		t.Errorf("GetBotUsername() = %q, want %q", got, "TestBot")
	}
}

func TestIsParseError(t *testing.T) {
	tests := []struct {
		err  error
		want bool
	}{
		{nil, false},
		{fmt.Errorf("some other error"), false},
		{fmt.Errorf("Bad Request: can't parse entities"), true},
		{fmt.Errorf("can't parse message text"), true},
	}
	for _, tt := range tests {
		if got := isParseError(tt.err); got != tt.want {
			t.Errorf("isParseError(%v) = %v, want %v", tt.err, got, tt.want)
		}
	}
}
