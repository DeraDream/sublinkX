package telegram

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sublink/models"
	"testing"
)

func TestParseAdminIDs(t *testing.T) {
	ids := parseAdminIDs("123, 456；789\ninvalid")
	if len(ids) != 3 || ids[0] != 123 || ids[1] != 456 || ids[2] != 789 {
		t.Fatalf("unexpected admin ids: %#v", ids)
	}
}

func TestManagerTestMessage(t *testing.T) {
	var calledPaths []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calledPaths = append(calledPaths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "result": map[string]any{}})
	}))
	defer server.Close()

	manager := &Manager{}
	err := manager.TestMessage(models.TelegramConfig{
		Token:        "test-token",
		AdminChatIDs: "123,456",
		APIBaseURL:   server.URL,
	})
	if err != nil {
		t.Fatal(err)
	}

	expected := []string{
		"/bottest-token/getMe",
		"/bottest-token/sendMessage",
		"/bottest-token/sendMessage",
	}
	if strings.Join(calledPaths, ",") != strings.Join(expected, ",") {
		t.Fatalf("unexpected calls: %#v", calledPaths)
	}
}
