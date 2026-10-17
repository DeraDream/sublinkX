package node

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEncodeLoonUsesProxyURIsInProxySection(t *testing.T) {
	output, err := EncodeLoon([]string{
		"ss://YWVzLTEyOC1nY206cGFzc0BleGFtcGxlLmNvbTo0NDM#SS",
		"tuic://unsupported",
		"vless://uuid@example.com:443?type=ws#VLESS",
	}, SqlConfig{})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "[Proxy]\nss://") || !strings.Contains(output, "vless://uuid") {
		t.Fatalf("Loon output did not contain supported proxy URIs: %q", output)
	}
	if strings.Contains(output, "tuic://") {
		t.Fatalf("Loon output must omit unsupported TUIC URI: %q", output)
	}
}

func TestEncodeLoonInjectsTemplate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "loon.conf")
	if err := os.WriteFile(path, []byte("[General]\ndns-server = system\n[Proxy]\n[Rule]\nFINAL,DIRECT"), 0600); err != nil {
		t.Fatal(err)
	}
	output, err := EncodeLoon([]string{"trojan://password@example.com:443#node"}, SqlConfig{Loon: path})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "[Proxy]\ntrojan://") || !strings.Contains(output, "[Rule]\nFINAL,DIRECT") {
		t.Fatalf("unexpected rendered Loon template: %q", output)
	}
}
