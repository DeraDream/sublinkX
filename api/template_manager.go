package api

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"sublink/utils"

	"github.com/gin-gonic/gin"
)

// VisualTemplate is deliberately separate from Temp.  Visual templates live
// under rule_templates and are never considered by the existing subscription
// template list.
type VisualTemplate struct {
	File      string `json:"file"`
	UpdatedAt string `json:"updated_at"`
}

type visualTemplatePayload struct {
	Filename string `json:"filename"`
	OldName  string `json:"old_name"`
	Content  string `json:"content"`
}

var visualTemplateDir string
var visualTemplateDirOnce sync.Once
var visualTemplateDirErr error

func ensureVisualTemplateDir() error {
	visualTemplateDirOnce.Do(func() {
		cwd, err := os.Getwd()
		if err != nil {
			visualTemplateDirErr = fmt.Errorf("无法获取当前工作目录: %w", err)
			return
		}
		visualTemplateDir, err = filepath.Abs(filepath.Join(cwd, "rule_templates"))
		if err != nil {
			visualTemplateDirErr = fmt.Errorf("无法解析可视化模板目录: %w", err)
			return
		}
		visualTemplateDirErr = os.MkdirAll(visualTemplateDir, 0755)
	})
	return visualTemplateDirErr
}

func visualTemplatePath(filename string) (string, error) {
	if err := ensureVisualTemplateDir(); err != nil {
		return "", err
	}
	filename = strings.TrimSpace(filename)
	if filepath.Ext(filename) == "" {
		filename += ".yaml"
	}
	if !isYAMLTemplateFilename(filename) {
		return "", errors.New("可视化模板仅支持 .yaml 或 .yml 文件")
	}
	clean := filepath.Clean(filename)
	if clean == "." || clean == ".." || strings.ContainsAny(clean, "/\\") {
		return "", errors.New("文件名无效")
	}
	path := filepath.Join(visualTemplateDir, clean)
	rel, err := filepath.Rel(visualTemplateDir, path)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return "", errors.New("文件名无效")
	}
	return path, nil
}

func visualTemplateJSON(c *gin.Context) (visualTemplatePayload, bool) {
	var payload visualTemplatePayload
	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(400, gin.H{"msg": "请求数据无效"})
		return payload, false
	}
	payload.Filename = strings.TrimSpace(payload.Filename)
	payload.OldName = strings.TrimSpace(payload.OldName)
	return payload, true
}

func ListVisualTemplates(c *gin.Context) {
	if err := ensureVisualTemplateDir(); err != nil {
		c.JSON(500, gin.H{"msg": err.Error()})
		return
	}
	entries, err := os.ReadDir(visualTemplateDir)
	if err != nil {
		c.JSON(500, gin.H{"msg": "无法读取可视化模板目录"})
		return
	}
	items := make([]VisualTemplate, 0)
	for _, entry := range entries {
		if entry.IsDir() || !isYAMLTemplateFilename(entry.Name()) {
			continue
		}
		info, err := entry.Info()
		if err != nil || info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		items = append(items, VisualTemplate{File: entry.Name(), UpdatedAt: utils.FormatBeijingTime(info.ModTime())})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].File < items[j].File })
	c.JSON(200, gin.H{"code": "00000", "data": items, "msg": "ok"})
}

func GetVisualTemplate(c *gin.Context) {
	path, err := visualTemplatePath(c.Query("filename"))
	if err != nil {
		c.JSON(400, gin.H{"msg": err.Error()})
		return
	}
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 {
		c.JSON(404, gin.H{"msg": "模板不存在"})
		return
	}
	content, err := os.ReadFile(path)
	if err != nil {
		c.JSON(500, gin.H{"msg": "无法读取模板"})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "data": gin.H{"file": filepath.Base(path), "content": string(content)}, "msg": "ok"})
}

func CreateVisualTemplate(c *gin.Context) {
	payload, ok := visualTemplateJSON(c)
	if !ok {
		return
	}
	path, err := visualTemplatePath(payload.Filename)
	if err != nil {
		c.JSON(400, gin.H{"msg": err.Error()})
		return
	}
	if strings.TrimSpace(payload.Content) == "" {
		c.JSON(400, gin.H{"msg": "模板内容不能为空"})
		return
	}
	if err := validateYAMLTemplate(path, payload.Content); err != nil {
		c.JSON(400, gin.H{"msg": err.Error()})
		return
	}
	if _, err := os.Lstat(path); err == nil {
		c.JSON(409, gin.H{"msg": "模板已存在"})
		return
	} else if !errors.Is(err, os.ErrNotExist) {
		c.JSON(500, gin.H{"msg": "无法创建模板"})
		return
	}
	if err := os.WriteFile(path, []byte(payload.Content), 0644); err != nil {
		c.JSON(500, gin.H{"msg": "保存模板失败"})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "msg": "创建成功"})
}

func UpdateVisualTemplate(c *gin.Context) {
	payload, ok := visualTemplateJSON(c)
	if !ok {
		return
	}
	if payload.OldName == "" {
		c.JSON(400, gin.H{"msg": "原模板名不能为空"})
		return
	}
	oldPath, err := visualTemplatePath(payload.OldName)
	if err != nil {
		c.JSON(400, gin.H{"msg": err.Error()})
		return
	}
	newPath, err := visualTemplatePath(payload.Filename)
	if err != nil {
		c.JSON(400, gin.H{"msg": err.Error()})
		return
	}
	if strings.TrimSpace(payload.Content) == "" {
		c.JSON(400, gin.H{"msg": "模板内容不能为空"})
		return
	}
	if err := validateYAMLTemplate(newPath, payload.Content); err != nil {
		c.JSON(400, gin.H{"msg": err.Error()})
		return
	}
	info, err := os.Lstat(oldPath)
	if err != nil || info.Mode()&os.ModeSymlink != 0 {
		c.JSON(404, gin.H{"msg": "原模板不存在"})
		return
	}
	if oldPath != newPath {
		if _, err := os.Lstat(newPath); err == nil {
			c.JSON(409, gin.H{"msg": "新模板名已存在"})
			return
		} else if !errors.Is(err, os.ErrNotExist) {
			c.JSON(500, gin.H{"msg": "无法更新模板"})
			return
		}
		if err := os.Rename(oldPath, newPath); err != nil {
			c.JSON(500, gin.H{"msg": "重命名模板失败"})
			return
		}
	}
	if err := os.WriteFile(newPath, []byte(payload.Content), 0644); err != nil {
		c.JSON(500, gin.H{"msg": "保存模板失败"})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "msg": "保存成功"})
}

func DeleteVisualTemplate(c *gin.Context) {
	payload, ok := visualTemplateJSON(c)
	if !ok {
		return
	}
	path, err := visualTemplatePath(payload.Filename)
	if err != nil {
		c.JSON(400, gin.H{"msg": err.Error()})
		return
	}
	info, err := os.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 {
		c.JSON(404, gin.H{"msg": "模板不存在"})
		return
	}
	if err := os.Remove(path); err != nil {
		c.JSON(500, gin.H{"msg": "删除模板失败"})
		return
	}
	c.JSON(200, gin.H{"code": "00000", "msg": "删除成功"})
}
