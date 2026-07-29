package api

import (
	"strings"
	"testing"
)

func TestValidateYAMLTemplateRejectsIndentTabs(t *testing.T) {
	err := validateYAMLTemplate("clash.yaml", "proxy-groups:\n\t- name: bad\n")
	if err == nil {
		t.Fatal("validateYAMLTemplate() expected an error")
	}
	if !strings.Contains(err.Error(), "第2行") || !strings.Contains(err.Error(), "Tab") {
		t.Fatalf("validateYAMLTemplate() error = %q, want line and tab message", err.Error())
	}
}

func TestValidateYAMLTemplateReportsParserLine(t *testing.T) {
	err := validateYAMLTemplate("clash.yaml", "proxy-groups:\n  - name: ok\n    proxies: [DIRECT\n")
	if err == nil {
		t.Fatal("validateYAMLTemplate() expected an error")
	}
	if !strings.Contains(err.Error(), "第2行") {
		t.Fatalf("validateYAMLTemplate() error = %q, want parser line", err.Error())
	}
}

func TestValidateYAMLTemplateSkipsNonYAMLFiles(t *testing.T) {
	if err := validateYAMLTemplate("surge.conf", "[Proxy Group]\nA = select,DIRECT\n"); err != nil {
		t.Fatalf("validateYAMLTemplate() error = %v, want nil for non-yaml", err)
	}
}
