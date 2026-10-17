package node

import (
	"fmt"
	"strings"
)

// EncodeLoon emits a Loon subscription file. Loon natively accepts the
// standard proxy URIs we store, avoiding lossy protocol conversion.
func EncodeLoon(urls []string, sqlconfig SqlConfig) (string, error) {
	proxies := make([]string, 0, len(urls))
	for _, raw := range urls {
		link := strings.TrimSpace(raw)
		if link == "" {
			continue
		}
		scheme := strings.ToLower(strings.SplitN(link, "://", 2)[0])
		switch scheme {
		case "ss", "ssr", "vmess", "vless", "trojan", "hysteria2", "hy2", "anytls":
			proxies = append(proxies, link)
		}
	}

	profile := "# Loon subscription\n[Proxy]\n"
	if source := strings.TrimSpace(sqlconfig.Loon); source != "" {
		data, err := ReadTemplateSource(source)
		if err != nil {
			return "", err
		}
		profile = string(data)
	}
	if !strings.Contains(profile, "[Proxy]") {
		return "", fmt.Errorf("Loon template must contain a [Proxy] section")
	}
	return replaceSurgeSection(profile, "Proxy", func(lines []string) []string {
		if len(proxies) == 0 {
			return lines
		}
		return append(proxies, lines...)
	}), nil
}
