package api

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sublink/models"
	"sublink/node"
	"time"

	"github.com/gin-gonic/gin"
)

var SunName string

// md5加密
func Md5(src string) string {
	m := md5.New()
	m.Write([]byte(src))
	res := hex.EncodeToString(m.Sum(nil))
	return res
}
func GetClient(c *gin.Context) {
	// 获取协议头
	token := c.Query("token")
	ClientIndex := c.Query("client") // 客户端标识
	if token == "" {
		log.Println("token为空")
		c.Writer.WriteString("token为空")
		return
	}
	sub, ok := findSubscriptionByToken(token)
	if !ok {
		c.Writer.WriteHeader(http.StatusNotFound)
		c.Writer.WriteString("订阅不存在或 token 无效")
		return
	}
	if available, reason := sub.IsAvailable(time.Now()); !available {
		c.Writer.WriteHeader(http.StatusForbidden)
		c.Writer.WriteString(reason)
		return
	}
	models.DB.Model(&models.Subcription{}).Where("id = ?", sub.ID).
		UpdateColumn("access_count", sub.AccessCount+1)
	SunName = sub.Name
	c.Set("subname", sub.Name)

	switch ClientIndex {
	case "clash":
		GetClash(c)
		return
	case "surge":
		GetSurge(c)
		return
	case "v2ray":
		GetV2ray(c)
		return
	}
	for _, userAgent := range c.Request.Header.Values("User-Agent") {
		ua := strings.ToLower(userAgent)
		if strings.Contains(ua, "clash") {
			GetClash(c)
			return
		}
		if strings.Contains(ua, "surge") {
			GetSurge(c)
			return
		}
	}
	GetV2ray(c)
}

func findSubscriptionByToken(token string) (models.Subcription, bool) {
	token = strings.ToLower(strings.TrimSpace(token))
	var sub models.Subcription
	if err := models.DB.Preload("Nodes").Preload("SubLogs").Where("token = ?", token).First(&sub).Error; err == nil {
		return sub, true
	}
	var subs []models.Subcription
	if err := models.DB.Preload("Nodes").Preload("SubLogs").Find(&subs).Error; err != nil {
		return models.Subcription{}, false
	}
	for _, item := range subs {
		if item.LegacyTokenDisabled {
			continue
		}
		if models.LegacySubscriptionToken(item.Name) == token {
			if strings.TrimSpace(item.Token) == "" {
				item.EnsureToken()
				_ = models.DB.Model(&item).Update("token", item.Token).Error
			}
			return item, true
		}
	}
	return models.Subcription{}, false
}

func subscriptionNodeLinks(n models.Node) []string {
	link := strings.TrimSpace(n.Link)
	if link == "" {
		return nil
	}
	if strings.Contains(link, "http://") || strings.Contains(link, "https://") {
		return []string{link}
	}
	if !strings.Contains(link, ",") {
		return []string{linkWithDisplayName(link, n.Name)}
	}
	links := strings.Split(link, ",")
	result := make([]string, 0, len(links))
	for _, item := range links {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func linkWithDisplayName(link, displayName string) string {
	displayName = strings.TrimSpace(displayName)
	if displayName == "" {
		return link
	}
	scheme := strings.ToLower(strings.SplitN(link, "://", 2)[0])
	switch scheme {
	case "vmess":
		vmess, err := node.DecodeVMESSURL(link)
		if err != nil {
			log.Println("重写 vmess 节点名称失败:", err)
			return link
		}
		vmess.Ps = displayName
		if vmess.V == "" {
			vmess.V = "2"
		}
		payload, err := json.Marshal(vmess)
		if err != nil {
			log.Println("重写 vmess 节点名称失败:", err)
			return link
		}
		return "vmess://" + node.Base64Encode(string(payload))
	default:
		u, err := url.Parse(link)
		if err != nil {
			log.Println("重写节点名称失败:", err)
			return link
		}
		u.Fragment = displayName
		return u.String()
	}
}

func GetV2ray(c *gin.Context) {
	var sub models.Subcription
	if SunName == "" {
		c.Writer.WriteString("订阅名为空")
		return
	}
	// subname := c.Param("subname")
	// subname := SunName
	// subname = node.Base64Decode(subname)
	sub.Name = SunName
	err := sub.Find()
	if err != nil {
		c.Writer.WriteString("找不到这个订阅:" + SunName)
		return
	}
	err = sub.Find()
	if err != nil {
		c.Writer.WriteString("读取错误")
		return
	}
	baselist := ""
	for _, v := range sub.ActiveNodes() {
		nodeLinks := subscriptionNodeLinks(v)
		switch {
		// 如果包含多条节点
		case strings.Contains(v.Link, ","):
			links := nodeLinks
			baselist += strings.Join(links, "\n") + "\n"
			continue
		//如果是订阅转换
		case strings.Contains(v.Link, "http://") || strings.Contains(v.Link, "https://"):
			resp, err := http.Get(v.Link)
			if err != nil {
				log.Println(err)
				return
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			nodes := node.Base64Decode(string(body))
			baselist += nodes + "\n"
		// 默认
		default:
			baselist += strings.Join(nodeLinks, "\n") + "\n"
		}
	}
	c.Set("subname", SunName)
	filename := fmt.Sprintf("%s.txt", SunName)
	encodedFilename := url.QueryEscape(filename)
	c.Writer.Header().Set("Content-Disposition", "inline; filename*=utf-8''"+encodedFilename)
	c.Writer.Header().Set("Content-Type", "text/html; charset=utf-8")
	c.Writer.WriteString(node.Base64Encode(baselist))
}
func GetClash(c *gin.Context) {
	var sub models.Subcription
	// subname := c.Param("subname")
	// subname := node.Base64Decode(SunName)
	sub.Name = SunName
	err := sub.Find()
	if err != nil {
		c.Writer.WriteString("找不到这个订阅:" + SunName)
		return
	}
	// err = sub.Find()

	urls := []string{}

	models.DB.Model(sub).Preload("Nodes").Find(&sub)
	log.Println("订阅名:", sub.Nodes)
	for _, v := range sub.ActiveNodes() {
		log.Println("节点信息:", v)
		log.Println("节点链接:", v.Link)
		nodeLinks := subscriptionNodeLinks(v)
		switch {
		// 如果包含多条节点
		case strings.Contains(v.Link, ","):
			links := nodeLinks
			urls = append(urls, links...)
			continue
		//如果是订阅转换
		case strings.Contains(v.Link, "http://") || strings.Contains(v.Link, "https://"):
			resp, err := http.Get(v.Link)
			if err != nil {
				log.Println(err)
				return
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			nodes := node.Base64Decode(string(body))
			links := strings.Split(nodes, "\n")
			urls = append(urls, links...)
		// 默认
		default:
			urls = append(urls, nodeLinks...)
		}
	}
	log.Println("urls", urls)
	var configs node.SqlConfig
	err = json.Unmarshal([]byte(sub.Config), &configs)
	if err != nil {
		c.Writer.WriteString("配置读取错误")
		return
	}
	DecodeClash, err := node.EncodeClash(urls, configs)
	if err != nil {
		c.Writer.WriteString(err.Error())
		return
	}
	c.Set("subname", SunName)
	filename := fmt.Sprintf("%s.yaml", SunName)
	encodedFilename := url.QueryEscape(filename)
	c.Writer.Header().Set("Content-Disposition", "inline; filename*=utf-8''"+encodedFilename)
	c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	c.Writer.WriteString(string(DecodeClash))
}
func GetSurge(c *gin.Context) {
	var sub models.Subcription
	// subname := c.Param("subname")
	// subname := node.Base64Decode(SunName)
	sub.Name = SunName
	err := sub.Find()
	if err != nil {
		c.Writer.WriteString("找不到这个订阅:" + SunName)
		return
	}
	err = sub.Find()
	if err != nil {
		c.Writer.WriteString("读取错误")
		return
	}
	urls := []string{}
	for _, v := range sub.ActiveNodes() {
		nodeLinks := subscriptionNodeLinks(v)
		switch {
		// 如果包含多条节点
		case strings.Contains(v.Link, ","):
			links := nodeLinks
			urls = append(urls, links...)
			continue
		//如果是订阅转换
		case strings.Contains(v.Link, "http://") || strings.Contains(v.Link, "https://"):
			resp, err := http.Get(v.Link)
			if err != nil {
				log.Println(err)
				return
			}
			defer resp.Body.Close()
			body, _ := io.ReadAll(resp.Body)
			nodes := node.Base64Decode(string(body))
			links := strings.Split(nodes, "\n")
			urls = append(urls, links...)
		// 默认
		default:
			urls = append(urls, nodeLinks...)
		}
	}

	var configs node.SqlConfig
	err = json.Unmarshal([]byte(sub.Config), &configs)
	if err != nil {
		c.Writer.WriteString("配置读取错误")
		return
	}
	// log.Println("surge路径:", configs)
	DecodeClash, err := node.EncodeSurge(urls, configs)
	if err != nil {
		c.Writer.WriteString(err.Error())
		return
	}
	c.Set("subname", SunName)
	filename := fmt.Sprintf("%s.conf", SunName)
	encodedFilename := url.QueryEscape(filename)
	c.Writer.Header().Set("Content-Disposition", "inline; filename*=utf-8''"+encodedFilename)
	c.Writer.Header().Set("Content-Type", "text/plain; charset=utf-8")
	host := c.Request.Host
	url := c.Request.URL.String()
	// 如果包含头部更新信息
	if strings.Contains(DecodeClash, "#!MANAGED-CONFIG") {
		c.Writer.WriteString(DecodeClash)
		return
	}
	// 否则就插入头部更新信息
	interval := fmt.Sprintf("#!MANAGED-CONFIG %s interval=86400 strict=false", host+url)
	c.Writer.WriteString(string(interval + "\n" + DecodeClash))
}
