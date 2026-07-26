package api

import (
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestSubscriptionFormatSelectsEgernExplicitly(t *testing.T) {
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "/c/?client=egern", nil)
	if got := subscriptionFormat(context, "egern"); got != "egern" {
		t.Fatalf("subscriptionFormat() = %q, want egern", got)
	}
}

func TestSubscriptionFormatDetectsEgernUserAgent(t *testing.T) {
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest("GET", "/c/", nil)
	context.Request.Header.Set("User-Agent", "Egern/1.0")
	if got := subscriptionFormat(context, ""); got != "egern" {
		t.Fatalf("subscriptionFormat() = %q, want egern", got)
	}
}
