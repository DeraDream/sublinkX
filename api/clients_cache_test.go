package api

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"sublink/models"
)

func TestTemplateFingerprintTracksLocalContent(t *testing.T) {
	templatePath := filepath.Join(t.TempDir(), "clash.yaml")
	if err := os.WriteFile(templatePath, []byte("mode: rule\n"), 0600); err != nil {
		t.Fatal(err)
	}
	config := fmt.Sprintf(`{"clash":%q}`, templatePath)

	first, cacheable := templateFingerprint(config, "clash")
	if !cacheable {
		t.Fatal("local template should be cacheable")
	}
	if err := os.WriteFile(templatePath, []byte("mode: test\n"), 0600); err != nil {
		t.Fatal(err)
	}
	second, cacheable := templateFingerprint(config, "clash")
	if !cacheable {
		t.Fatal("local template should remain cacheable")
	}
	if first == second {
		t.Fatal("template fingerprint did not change with file content")
	}
}

func TestRemoteTemplateDisablesSubscriptionCache(t *testing.T) {
	config := `{"clash":"https://example.com/clash.yaml"}`
	if _, cacheable := templateFingerprint(config, "clash"); cacheable {
		t.Fatal("remote templates must not use the generated subscription cache")
	}
}

func TestNodeFingerprintTracksNodeChanges(t *testing.T) {
	nodes := []models.Node{{ID: 1, Name: "node", Link: "ss://first"}}
	first := subscriptionNodesFingerprint(nodes)
	nodes[0].Link = "ss://second"
	second := subscriptionNodesFingerprint(nodes)
	if first == second {
		t.Fatal("node fingerprint did not change with node content")
	}
}
