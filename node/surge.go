package node

import (
	"fmt"
	"log"
	"regexp"
	"strings"
)

func EncodeSurge(urls []string, sqlconfig SqlConfig) (string, error) {
	var proxys, groups []string
	for _, link := range urls {
		Scheme := strings.Split(link, "://")[0]
		switch {
		case Scheme == "ss":
			ss, err := DecodeSSURL(link)
			if err != nil {
				log.Println(err)
				continue
			}
			proxy := map[string]interface{}{
				"name":     ss.Name,
				"server":   ss.Server,
				"port":     ss.Port,
				"cipher":   ss.Param.Cipher,
				"password": ss.ClientPassword(),
				"udp":      sqlconfig.Udp,
			}
			ssproxy := fmt.Sprintf("%s = ss, %s, %d, encrypt-method=%s, password=%s, udp-relay=%t",
				proxy["name"], proxy["server"], proxy["port"], proxy["cipher"], proxy["password"], proxy["udp"])
			groups = append(groups, ss.Name)
			proxys = append(proxys, ssproxy)
		case Scheme == "vmess":
			vmess, err := DecodeVMESSURL(link)
			if err != nil {
				log.Println(err)
				continue
			}
			tls := false
			if vmess.Tls != "none" && vmess.Tls != "" {
				tls = true
			}
			port, err := convertToInt(vmess.Port)
			if err != nil || port <= 0 {
				log.Println("invalid vmess port")
				continue
			}
			proxy := map[string]interface{}{
				"name":             vmess.Ps,
				"server":           vmess.Add,
				"port":             port,
				"uuid":             vmess.Id,
				"tls":              tls,
				"network":          vmess.Net,
				"ws-path":          vmess.Path,
				"ws-host":          vmess.Host,
				"udp":              sqlconfig.Udp,
				"skip-cert-verify": sqlconfig.Cert,
			}
			vmessproxy := fmt.Sprintf("%s = vmess, %s, %d, username=%s , tls=%t, vmess-aead=true,  udp-relay=%t , skip-cert-verify=%t",
				proxy["name"], proxy["server"], proxy["port"], proxy["uuid"], proxy["tls"], proxy["udp"], proxy["skip-cert-verify"])
			if vmess.Net == "ws" {
				vmessproxy = fmt.Sprintf("%s, ws=true,ws-path=%s", vmessproxy, proxy["ws-path"])
				if vmess.Host != "" && vmess.Host != "none" {
					vmessproxy = fmt.Sprintf("%s, ws-headers=Host:%s", vmessproxy, proxy["ws-host"])
				}
			}
			if vmess.Sni != "" {
				vmessproxy = fmt.Sprintf("%s, sni=%s", vmessproxy, vmess.Sni)
			}
			groups = append(groups, vmess.Ps)
			proxys = append(proxys, vmessproxy)
		case Scheme == "trojan":
			trojan, err := DecodeTrojanURL(link)
			if err != nil {
				log.Println(err)
				continue
			}
			proxy := map[string]interface{}{
				"name":             trojan.Name,
				"server":           trojan.Hostname,
				"port":             trojan.Port,
				"password":         trojan.Password,
				"udp":              sqlconfig.Udp,
				"skip-cert-verify": sqlconfig.Cert,
			}
			trojanproxy := fmt.Sprintf("%s = trojan, %s, %d, password=%s, udp-relay=%t, skip-cert-verify=%t",
				proxy["name"], proxy["server"], proxy["port"], proxy["password"], proxy["udp"], proxy["skip-cert-verify"])
			if trojan.Query.Sni != "" {
				trojanproxy = fmt.Sprintf("%s, sni=%s", trojanproxy, trojan.Query.Sni)

			}
			groups = append(groups, trojan.Name)
			proxys = append(proxys, trojanproxy)
		case Scheme == "hysteria2" || Scheme == "hy2":
			hy2, err := DecodeHY2URL(link)
			if err != nil {
				log.Println(err)
				continue
			}
			proxy := map[string]interface{}{
				"name":             hy2.Name,
				"server":           hy2.Host,
				"port":             hy2.Port,
				"password":         hy2.Password,
				"udp":              sqlconfig.Udp,
				"skip-cert-verify": sqlconfig.Cert,
			}
			hy2proxy := fmt.Sprintf("%s = hysteria2, %s, %d, password=%s, udp-relay=%t, skip-cert-verify=%t",
				proxy["name"], proxy["server"], proxy["port"], proxy["password"], proxy["udp"], proxy["skip-cert-verify"])
			if hy2.Sni != "" {
				hy2proxy = fmt.Sprintf("%s, sni=%s", hy2proxy, hy2.Sni)

			}
			groups = append(groups, hy2.Name)
			proxys = append(proxys, hy2proxy)
		case Scheme == "tuic":
			tuic, err := DecodeTuicURL(link)
			if err != nil {
				log.Println(err)
				continue
			}
			proxy := map[string]interface{}{
				"name":             tuic.Name,
				"server":           tuic.Host,
				"port":             tuic.Port,
				"password":         tuic.Password,
				"udp":              sqlconfig.Udp,
				"skip-cert-verify": sqlconfig.Cert,
			}
			tuicproxy := fmt.Sprintf("%s = tuic, %s, %d, token=%s, udp-relay=%t, skip-cert-verify=%t",
				proxy["name"], proxy["server"], proxy["port"], proxy["password"], proxy["udp"], proxy["skip-cert-verify"])
			groups = append(groups, tuic.Name)
			proxys = append(proxys, tuicproxy)
		}
	}
	return DecodeSurge(proxys, groups, sqlconfig)
}
func DecodeSurge(proxys, groups []string, sqlconfig SqlConfig) (string, error) {
	surge, err := ReadTemplateSource(sqlconfig.Surge)
	if err != nil {
		log.Println(err)
		return "", err
	}

	proxyReg := regexp.MustCompile(`(?s)\[Proxy\](.*?)\[*]`)
	groupReg := regexp.MustCompile(`(?s)\[Proxy Group\](.*?)\[*]`)

	proxyPart := proxyReg.ReplaceAllStringFunc(string(surge), func(s string) string {

		text := strings.Join(proxys, "\n")
		return "[Proxy]\n" + text + s[len("[Proxy]"):]
	})
	groupPart := groupReg.ReplaceAllStringFunc(proxyPart, func(s string) string {
		lines := strings.Split(s, "\n")
		for i, line := range lines {

			if strings.Contains(line, "=") {
				parts := strings.SplitN(line, "=", 2)
				groupName := strings.TrimSpace(parts[0])
				groupNames := selectedProxyNamesForGroup(groupName, groups, sqlconfig.GroupNodes)
				if len(groupNames) == 0 {
					lines[i] = strings.TrimSpace(line)
					continue
				}
				existingItems := strings.Split(parts[1], ",")
				mergedItems := appendUniqueStringNames(existingItems, groupNames)
				lines[i] = groupName + " = " + strings.Join(mergedItems, ", ")
				// lines[i] = line + "," + grouplist
			}
		}
		return strings.Join(lines, "\n") + s[len("[Proxy Group]"):]
	})

	return groupPart, nil
}
