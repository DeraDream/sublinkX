package agent

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestBuildXrayConfig(t *testing.T) {
	tests := []struct {
		name string
		link string
	}{
		{
			name: "shadowsocks",
			link: "ss://YWVzLTEyOC1nY206cGFzc3dvcmQ=@127.0.0.1:8388#test",
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
			cfg, err := buildXrayConfig(tt.link, 18080)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := json.Marshal(cfg); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestXrayAcceptsGeneratedConfigs(t *testing.T) {
	binary := os.Getenv("XRAY_BIN")
	if binary == "" {
		t.Skip("XRAY_BIN is not set")
	}
	links := []string{
		"ss://YWVzLTEyOC1nY206cGFzc3dvcmQ=@127.0.0.1:8388#test",
		"vless://11111111-1111-1111-1111-111111111111@example.com:443?encryption=none&flow=xtls-rprx-vision&fp=chrome&pbk=g-oxbqigzCaXqARxuyD2_vbTYeMD9zn8wnTo02S69QM&security=reality&sid=abcd&sni=example.com&type=tcp#test",
		"vless://11111111-1111-1111-1111-111111111111@example.com:443?encryption=none&host=example.com&path=%2Fws&security=tls&sni=example.com&type=ws#test",
	}
	for _, link := range links {
		cfg, err := buildXrayConfig(link, 18080)
		if err != nil {
			t.Fatal(err)
		}
		data, _ := json.Marshal(cfg)
		path := filepath.Join(t.TempDir(), "config.json")
		if err := os.WriteFile(path, data, 0600); err != nil {
			t.Fatal(err)
		}
		if output, err := exec.Command(binary, "run", "-test", "-c", path).CombinedOutput(); err != nil {
			t.Fatalf("xray rejected config: %v\n%s", err, output)
		}
	}
}
