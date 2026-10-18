package node

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These templates deliberately live under rule_templates, the directory used
// by Template Management, rather than the legacy ./template directory.
func TestManagedTemplatesReceiveInjectedNodes(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "rule_templates")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	clashPath := filepath.Join(dir, "clash.yaml")
	egernPath := filepath.Join(dir, "egern.yaml")
	loonPath := filepath.Join(dir, "loon.conf")
	if err := os.WriteFile(clashPath, []byte("proxies: []\nproxy-groups:\n  - name: Proxy\n    type: select\nrules: []\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(egernPath, []byte("proxies: []\npolicy_groups:\n  - select:\n      name: Proxy\n      policies:\n        - DIRECT\nrules:\n  - default:\n      policy: Proxy\n"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(loonPath, []byte("[Proxy]\n__LOON_PROXY_ENTRIES__\n[Proxy Group]\nProxy = select,__LOON_ALL_NODES__,DIRECT\n[Rule]\nFINAL,Proxy\n"), 0600); err != nil {
		t.Fatal(err)
	}

	link := testSSLink("Managed Node")
	clash, err := EncodeClash([]string{link}, SqlConfig{Clash: clashPath, GroupNodesTemplate: clashPath, GroupNodes: map[string]PolicyGroupNodeRule{"Proxy": {Mode: "include", Nodes: []string{"Managed Node"}}}})
	if err != nil || !strings.Contains(string(clash), "Managed Node") {
		t.Fatalf("managed Clash injection failed: %v\n%s", err, clash)
	}
	egern, err := EncodeEgern([]string{link}, SqlConfig{Egern: egernPath, GroupNodesTemplate: egernPath, GroupNodes: map[string]PolicyGroupNodeRule{"Proxy": {Mode: "include", Nodes: []string{"Managed Node"}}}})
	if err != nil || !strings.Contains(string(egern), "Managed Node") {
		t.Fatalf("managed Egern injection failed: %v\n%s", err, egern)
	}
	loon, err := EncodeLoon([]string{link}, SqlConfig{Loon: loonPath, GroupNodesTemplate: loonPath, GroupNodes: map[string]PolicyGroupNodeRule{"Proxy": {Mode: "include", Nodes: []string{"Managed Node"}}}})
	if err != nil || !strings.Contains(loon, "Proxy = select,Managed Node,DIRECT") {
		t.Fatalf("managed Loon injection failed: %v\n%s", err, loon)
	}
}
