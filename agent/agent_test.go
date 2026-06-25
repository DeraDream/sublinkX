package agent

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestBuildSingBoxConfig(t *testing.T) {
	tests := []struct {
		name string
		link string
	}{
		{
			name: "shadowsocks",
			link: "ss://YWVzLTEyOC1nY206cGFzc3dvcmQ=@127.0.0.1:8388#test",
		},
		{
			name: "shadowsocks 2022",
			link: "ss://MjAyMi1ibGFrZTMtYWVzLTEyOC1nY206OEpDc1Bzc2ZnUzh0aVJ3aU1saEFyZz09@127.0.0.1:8388#test",
		},
		{
			name: "vless reality",
			link: "vless://11111111-1111-1111-1111-111111111111@example.com:443?encryption=none&flow=xtls-rprx-vision&fp=chrome&pbk=g-oxbqigzCaXqARxuyD2_vbTYeMD9zn8wnTo02S69QM&security=reality&sid=abcd&sni=example.com&type=tcp#test",
		},
		{
			name: "vless websocket tls",
			link: "vless://11111111-1111-1111-1111-111111111111@example.com:443?encryption=none&host=example.com&path=%2Fws&security=tls&sni=example.com&type=ws#test",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := buildSingBoxConfig(tt.link, 18080)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := json.Marshal(cfg); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestSingBoxShadowsocks2022EndToEnd(t *testing.T) {
	if os.Getenv("SING_BOX_INTEGRATION") != "1" {
		t.Skip("SING_BOX_INTEGRATION is not set")
	}
	binary := os.Getenv("SING_BOX_BIN")
	if binary == "" {
		t.Skip("SING_BOX_BIN is not set")
	}
	port, err := freePort()
	if err != nil {
		t.Fatal(err)
	}
	password := "8JCsPssfgS8tiRwiMlhARg=="
	serverConfig := map[string]any{
		"log": map[string]any{"level": "warn"},
		"inbounds": []any{map[string]any{
			"type": "shadowsocks", "listen": "127.0.0.1", "listen_port": port,
			"method": "2022-blake3-aes-128-gcm", "password": password,
		}},
		"outbounds": []any{map[string]any{"type": "direct"}},
	}
	data, _ := json.Marshal(serverConfig)
	configPath := filepath.Join(t.TempDir(), "server.json")
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		t.Fatal(err)
	}
	cmd := exec.Command(binary, "run", "-c", configPath)
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer stopProcess(cmd)
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := waitPort(ctx, port); err != nil {
		t.Fatal(err)
	}
	auth := base64.RawURLEncoding.EncodeToString(
		[]byte("2022-blake3-aes-128-gcm:" + password),
	)
	link := fmt.Sprintf("ss://%s@127.0.0.1:%d#integration", auth, port)
	result, err := runSingBoxTest(ctx, binary, link, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.LatencyMs <= 0 {
		t.Fatalf("invalid latency: %d", result.LatencyMs)
	}
}

func TestSingBoxAcceptsGeneratedConfigs(t *testing.T) {
	binary := os.Getenv("SING_BOX_BIN")
	if binary == "" {
		t.Skip("SING_BOX_BIN is not set")
	}
	links := []string{
		"ss://YWVzLTEyOC1nY206cGFzc3dvcmQ=@127.0.0.1:8388#test",
		"ss://MjAyMi1ibGFrZTMtYWVzLTEyOC1nY206OEpDc1Bzc2ZnUzh0aVJ3aU1saEFyZz09@127.0.0.1:8388#test",
		"vless://11111111-1111-1111-1111-111111111111@example.com:443?encryption=none&flow=xtls-rprx-vision&fp=chrome&pbk=g-oxbqigzCaXqARxuyD2_vbTYeMD9zn8wnTo02S69QM&security=reality&sid=abcd&sni=example.com&type=tcp#test",
		"vless://11111111-1111-1111-1111-111111111111@example.com:443?encryption=none&host=example.com&path=%2Fws&security=tls&sni=example.com&type=ws#test",
	}
	for _, link := range links {
		cfg, err := buildSingBoxConfig(link, 18080)
		if err != nil {
			t.Fatal(err)
		}
		data, _ := json.Marshal(cfg)
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		if output, err := exec.Command(binary, "check", "-c", path).CombinedOutput(); err != nil {
			t.Fatalf("sing-box rejected config: %v\n%s", err, output)
		}
	}
}
