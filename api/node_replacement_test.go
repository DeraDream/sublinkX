package api

import (
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"

	"sublink/models"

	"github.com/gin-gonic/gin"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
)

func TestNodeAddReplacesServerBeforeSaving(t *testing.T) {
	previousDB := models.DB
	t.Cleanup(func() { models.DB = previousDB })

	database, err := gorm.Open(sqlite.Open(filepath.Join(t.TempDir(), "replacement.db")), &gorm.Config{})
	if err != nil {
		t.Fatal(err)
	}
	if err := database.AutoMigrate(&models.Node{}, &models.IPEntry{}); err != nil {
		t.Fatal(err)
	}
	models.DB = database
	entry := models.IPEntry{Alias: "香港入口", Address: "198.51.100.24"}
	if err := database.Create(&entry).Error; err != nil {
		t.Fatal(err)
	}

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/nodes/add", NodeAdd)
	form := url.Values{
		"name":          {"实时替换测试"},
		"link":          {"ss://YWVzLTI1Ni1nY206cGFzcw==@203.0.113.10:8388#修改后的备注"},
		"replace_ip_id": {"1"},
	}
	request := httptest.NewRequest(http.MethodPost, "/nodes/add", strings.NewReader(form.Encode()))
	request.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	response := httptest.NewRecorder()
	router.ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("NodeAdd status = %d, body = %s", response.Code, response.Body.String())
	}

	var saved models.Node
	if err := database.First(&saved).Error; err != nil {
		t.Fatal(err)
	}
	want := "ss://YWVzLTI1Ni1nY206cGFzcw==@198.51.100.24:8388#修改后的备注"
	if saved.Link != want {
		t.Fatalf("saved link = %q, want %q", saved.Link, want)
	}
}
