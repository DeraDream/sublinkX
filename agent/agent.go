package agent

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sublink/node"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

const Version = "0.1.0"

type Config struct {
	Server string `yaml:"server"`
	Token  string `yaml:"token"`
}

type pollResponse struct {
	Code string `json:"code"`
	Data struct {
		Mode      string `json:"mode"`
		PollAfter int    `json:"poll_after"`
		Task      *struct {
			ID       uint   `json:"id"`
			Type     string `json:"type"`
			NodeLink string `json:"node_link"`
		} `json:"task"`
	} `json:"data"`
	Msg string `json:"msg"`
}

type taskReport struct {
	TaskID       uint    `json:"task_id"`
	Success      bool    `json:"success"`
	LatencyMs    int64   `json:"latency_ms"`
	DownloadMbps float64 `json:"download_mbps"`
	TestBytes    int64   `json:"test_bytes"`
	EgressIP     string  `json:"egress_ip"`
	Error        string  `json:"error"`
}

type speedResult struct {
	LatencyMs    int64
	DownloadMbps float64
	TestBytes    int64
	EgressIP     string
}

func Main(args []string) error {
	if len(args) == 0 {
		return errors.New("用法: sublink agent run|install --server URL --token TOKEN")
	}
	switch args[0] {
	case "run":
		return runCommand(args[1:])
	case "install":
		return installCommand(args[1:])
	default:
		return fmt.Errorf("未知 agent 命令: %s", args[0])
	}
}

func parseFlags(name string, args []string) (Config, error) {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	server := fs.String("server", "", "sublinkX 服务地址")
	token := fs.String("token", "", "家宽测速端令牌")
	configPath := fs.String("config", "", "配置文件路径")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	if *configPath != "" {
		data, err := os.ReadFile(*configPath)
		if err != nil {
			return Config{}, err
		}
		var cfg Config
		if err := yaml.Unmarshal(data, &cfg); err != nil {
			return Config{}, err
		}
		return cfg, validateConfig(cfg)
	}
	cfg := Config{Server: strings.TrimRight(*server, "/"), Token: *token}
	return cfg, validateConfig(cfg)
}

func validateConfig(cfg Config) error {
	if cfg.Server == "" || cfg.Token == "" {
		return errors.New("server 和 token 必填")
	}
	if _, err := url.ParseRequestURI(cfg.Server); err != nil {
		return errors.New("server 地址无效")
	}
	return nil
}

func runCommand(args []string) error {
	cfg, err := parseFlags("agent run", args)
	if err != nil {
		return err
	}
	return Run(context.Background(), cfg)
}

func installCommand(args []string) error {
	if runtime.GOOS != "linux" {
		return errors.New("自动安装目前仅支持 Linux；Windows 请使用 agent run")
	}
	cfg, err := parseFlags("agent install", args)
	if err != nil {
		return err
	}
	configDir := "/etc/sublink-agent"
	if err := os.MkdirAll(configDir, 0750); err != nil {
		return err
	}
	data, _ := yaml.Marshal(cfg)
	if err := os.WriteFile(filepath.Join(configDir, "config.yaml"), data, 0600); err != nil {
		return err
	}
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	target := "/usr/local/bin/sublink-agent"
	if filepath.Clean(exe) != target {
		input, err := os.Open(exe)
		if err != nil {
			return err
		}
		defer input.Close()
		output, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
		if err != nil {
			return err
		}
		if _, err := io.Copy(output, input); err != nil {
			output.Close()
			return err
		}
		if err := output.Close(); err != nil {
			return err
		}
	}
	service := `[Unit]
Description=sublinkX Home Speed Test Agent
After=network-online.target
Wants=network-online.target

[Service]
Type=simple
ExecStart=/usr/local/bin/sublink-agent agent run --config /etc/sublink-agent/config.yaml
Restart=always
RestartSec=10
NoNewPrivileges=true

[Install]
WantedBy=multi-user.target
`
	if err := os.WriteFile("/etc/systemd/system/sublink-agent.service", []byte(service), 0644); err != nil {
		return err
	}
	if err := exec.Command("systemctl", "daemon-reload").Run(); err != nil {
		return err
	}
	if output, err := exec.Command("systemctl", "enable", "--now", "sublink-agent").CombinedOutput(); err != nil {
		return fmt.Errorf("启动服务失败: %s", strings.TrimSpace(string(output)))
	}
	fmt.Println("sublink-agent 已安装并启动")
	return nil
}

func Run(ctx context.Context, cfg Config) error {
	client := &http.Client{Timeout: 75 * time.Second}
	fmt.Printf("sublink-agent %s 已启动，服务端 %s\n", Version, cfg.Server)
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}
		poll, err := pollTask(ctx, client, cfg)
		if err != nil {
			fmt.Println("获取任务失败:", err)
			if !sleepContext(ctx, 30*time.Second) {
				return ctx.Err()
			}
			continue
		}
		if poll.Data.Task != nil {
			task := poll.Data.Task
			report := executeTask(ctx, task.ID, task.Type, task.NodeLink)
			if err := reportTask(ctx, client, cfg, report); err != nil {
				fmt.Println("上报任务失败:", err)
			}
		}
		delay := poll.Data.PollAfter
		if delay < 3 {
			delay = 3
		}
		if !sleepContext(ctx, time.Duration(delay)*time.Second) {
			return ctx.Err()
		}
	}
}

func agentRequest(ctx context.Context, client *http.Client, cfg Config, path string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, cfg.Server+path, body)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-Agent-Token", cfg.Token)
	req.Header.Set("X-Agent-Version", Version)
	req.Header.Set("X-Agent-Platform", runtime.GOOS+"/"+runtime.GOARCH)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	return client.Do(req)
}

func pollTask(ctx context.Context, client *http.Client, cfg Config) (pollResponse, error) {
	resp, err := agentRequest(ctx, client, cfg, "/api/v1/agent/poll", nil)
	if err != nil {
		return pollResponse{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		data, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return pollResponse{}, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	var out pollResponse
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return out, err
	}
	return out, nil
}

func reportTask(ctx context.Context, client *http.Client, cfg Config, report taskReport) error {
	data, _ := json.Marshal(report)
	resp, err := agentRequest(ctx, client, cfg, "/api/v1/agent/report", bytes.NewReader(data))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	return nil
}

func sleepContext(ctx context.Context, delay time.Duration) bool {
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-timer.C:
		return true
	case <-ctx.Done():
		return false
	}
}

func executeTask(ctx context.Context, id uint, testType, nodeLink string) taskReport {
	report := taskReport{TaskID: id}
	xray, err := ensureXray(ctx)
	if err != nil {
		report.Error = err.Error()
		return report
	}
	result, err := runXrayTest(ctx, xray, nodeLink, testType == "speed")
	report.LatencyMs = result.LatencyMs
	report.DownloadMbps = result.DownloadMbps
	report.TestBytes = result.TestBytes
	report.EgressIP = result.EgressIP
	if err != nil {
		report.Error = err.Error()
		return report
	}
	report.Success = true
	return report
}

func runXrayTest(parent context.Context, binary, nodeLink string, download bool) (speedResult, error) {
	ctx, cancel := context.WithTimeout(parent, 2*time.Minute)
	defer cancel()
	port, err := freePort()
	if err != nil {
		return speedResult{}, err
	}
	cfg, err := buildXrayConfig(nodeLink, port)
	if err != nil {
		return speedResult{}, err
	}
	data, err := json.Marshal(cfg)
	if err != nil {
		return speedResult{}, err
	}
	workdir, err := os.MkdirTemp("", "sublink-speedtest-*")
	if err != nil {
		return speedResult{}, err
	}
	defer os.RemoveAll(workdir)
	configPath := filepath.Join(workdir, "config.json")
	if err := os.WriteFile(configPath, data, 0600); err != nil {
		return speedResult{}, err
	}
	cmd := exec.CommandContext(ctx, binary, "run", "-c", configPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Start(); err != nil {
		return speedResult{}, err
	}
	defer stopProcess(cmd)
	if err := waitPort(ctx, port); err != nil {
		return speedResult{}, fmt.Errorf("%w: %s", err, strings.TrimSpace(stderr.String()))
	}
	proxyURL, _ := url.Parse(fmt.Sprintf("http://127.0.0.1:%d", port))
	transport := &http.Transport{
		Proxy:               http.ProxyURL(proxyURL),
		DisableCompression:  true,
		ForceAttemptHTTP2:   false,
		MaxIdleConns:        16,
		IdleConnTimeout:     30 * time.Second,
		TLSHandshakeTimeout: 10 * time.Second,
	}
	defer transport.CloseIdleConnections()
	client := &http.Client{Transport: transport}
	result := speedResult{
		LatencyMs: measureLatency(ctx, client),
		EgressIP:  measureEgressIP(ctx, client),
	}
	if result.LatencyMs < 0 {
		return result, errors.New("节点延迟测试失败")
	}
	if download {
		bytesRead, duration, err := measureDownload(ctx, client)
		if err != nil {
			return result, err
		}
		result.TestBytes = bytesRead
		result.DownloadMbps = float64(bytesRead) * 8 / duration.Seconds() / 1e6
	}
	return result, nil
}

func buildXrayConfig(link string, port int) (map[string]any, error) {
	scheme := strings.ToLower(strings.SplitN(link, "://", 2)[0])
	var outbound map[string]any
	switch scheme {
	case "ss":
		ss, err := node.DecodeSSURL(link)
		if err != nil {
			return nil, fmt.Errorf("解析 SS 节点失败: %w", err)
		}
		outbound = map[string]any{
			"protocol": "shadowsocks",
			"tag":      "node",
			"settings": map[string]any{"servers": []any{map[string]any{
				"address": ss.Server, "port": ss.Port,
				"method": ss.Param.Cipher, "password": ss.Param.Password,
			}}},
		}
	case "vless":
		vless, err := node.DecodeVLESSURL(link)
		if err != nil {
			return nil, fmt.Errorf("解析 VLESS 节点失败: %w", err)
		}
		user := map[string]any{"id": vless.Uuid, "encryption": "none"}
		if vless.Query.Flow != "" {
			user["flow"] = vless.Query.Flow
		}
		network := defaultString(vless.Query.Type, "tcp")
		stream := map[string]any{"network": network}
		switch vless.Query.Security {
		case "tls":
			tls := map[string]any{
				"serverName":  vless.Query.Sni,
				"fingerprint": defaultString(vless.Query.Fp, "chrome"),
			}
			if len(vless.Query.Alpn) > 0 {
				tls["alpn"] = vless.Query.Alpn
			}
			stream["security"] = "tls"
			stream["tlsSettings"] = tls
		case "reality":
			stream["security"] = "reality"
			stream["realitySettings"] = map[string]any{
				"serverName":  vless.Query.Sni,
				"fingerprint": defaultString(vless.Query.Fp, "chrome"),
				"publicKey":   vless.Query.Pbk,
				"shortId":     vless.Query.Sid,
				"spiderX":     vless.Query.Path,
			}
		case "", "none":
		default:
			return nil, fmt.Errorf("暂不支持 VLESS security=%s", vless.Query.Security)
		}
		switch network {
		case "ws":
			stream["wsSettings"] = map[string]any{
				"path":    vless.Query.Path,
				"headers": map[string]any{"Host": vless.Query.Host},
			}
		case "grpc":
			stream["grpcSettings"] = map[string]any{
				"serviceName": vless.Query.ServiceName,
				"multiMode":   vless.Query.Mode == "multi",
			}
		case "tcp", "raw":
			stream["network"] = "raw"
			if vless.Query.HeaderType != "" && vless.Query.HeaderType != "none" {
				stream["rawSettings"] = map[string]any{
					"header": map[string]any{"type": vless.Query.HeaderType},
				}
			}
		default:
			return nil, fmt.Errorf("暂不支持 VLESS transport=%s", network)
		}
		outbound = map[string]any{
			"protocol": "vless",
			"tag":      "node",
			"settings": map[string]any{"vnext": []any{map[string]any{
				"address": vless.Server, "port": vless.Port, "users": []any{user},
			}}},
			"streamSettings": stream,
		}
	default:
		return nil, fmt.Errorf("暂不支持 %s 节点测速", scheme)
	}
	return map[string]any{
		"log": map[string]any{"loglevel": "warning"},
		"inbounds": []any{map[string]any{
			"listen": "127.0.0.1", "port": port,
			"protocol": "http", "settings": map[string]any{},
		}},
		"outbounds": []any{
			outbound,
			map[string]any{"protocol": "freedom", "tag": "direct"},
		},
	}, nil
}

func defaultString(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func measureLatency(ctx context.Context, client *http.Client) int64 {
	var samples []int64
	for i := 0; i < 3; i++ {
		probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
		req, _ := http.NewRequestWithContext(probeCtx, http.MethodGet, "https://cp.cloudflare.com/generate_204", nil)
		start := time.Now()
		resp, err := client.Do(req)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			samples = append(samples, time.Since(start).Milliseconds())
		}
		cancel()
	}
	if len(samples) == 0 {
		return -1
	}
	sort.Slice(samples, func(i, j int) bool { return samples[i] < samples[j] })
	keep := 2
	if len(samples) < keep {
		keep = len(samples)
	}
	var sum int64
	for i := 0; i < keep; i++ {
		sum += samples[i]
	}
	return sum / int64(keep)
}

func measureEgressIP(ctx context.Context, client *http.Client) string {
	probeCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(probeCtx, http.MethodGet, "https://api.ipify.org", nil)
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(io.LimitReader(resp.Body, 64))
	return strings.TrimSpace(string(data))
}

func measureDownload(ctx context.Context, client *http.Client) (int64, time.Duration, error) {
	downloadCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
	defer cancel()
	req, _ := http.NewRequestWithContext(downloadCtx, http.MethodGet, "https://speed.cloudflare.com/__down?bytes=100000000", nil)
	req.Header.Set("Accept-Encoding", "identity")
	start := time.Now()
	resp, err := client.Do(req)
	if err != nil {
		return 0, 0, fmt.Errorf("下载测速失败: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode/100 != 2 {
		return 0, 0, fmt.Errorf("下载测速 HTTP %d", resp.StatusCode)
	}
	n, copyErr := io.Copy(io.Discard, resp.Body)
	duration := time.Since(start)
	if n == 0 {
		return 0, duration, errors.New("下载测速未收到数据")
	}
	if copyErr != nil && downloadCtx.Err() != context.DeadlineExceeded {
		return n, duration, copyErr
	}
	return n, duration, nil
}

func freePort() (int, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return 0, err
	}
	defer listener.Close()
	return listener.Addr().(*net.TCPAddr).Port, nil
}

func waitPort(ctx context.Context, port int) error {
	address := fmt.Sprintf("127.0.0.1:%d", port)
	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", address, 300*time.Millisecond)
		if err == nil {
			conn.Close()
			return nil
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(200 * time.Millisecond):
		}
	}
	return errors.New("节点协议引擎启动超时")
}

func stopProcess(cmd *exec.Cmd) {
	if cmd == nil || cmd.Process == nil {
		return
	}
	if runtime.GOOS != "windows" {
		_ = cmd.Process.Signal(syscall.SIGTERM)
		time.Sleep(500 * time.Millisecond)
	}
	_ = cmd.Process.Kill()
	_ = cmd.Wait()
}

func ensureXray(ctx context.Context) (string, error) {
	if custom := os.Getenv("XRAY_BIN"); custom != "" {
		if _, err := os.Stat(custom); err == nil {
			return custom, nil
		}
	}
	if path, err := exec.LookPath("xray"); err == nil {
		return path, nil
	}
	cacheDir, err := os.UserCacheDir()
	if err != nil {
		cacheDir = os.TempDir()
	}
	targetDir := filepath.Join(cacheDir, "sublink-agent")
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return "", err
	}
	name := "xray"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	target := filepath.Join(targetDir, name)
	if _, err := os.Stat(target); err == nil {
		return target, nil
	}
	return downloadXray(ctx, target)
}

func downloadXray(ctx context.Context, target string) (string, error) {
	assetName := ""
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		assetName = "Xray-linux-64.zip"
	case "linux/arm64":
		assetName = "Xray-linux-arm64-v8a.zip"
	case "windows/amd64":
		assetName = "Xray-windows-64.zip"
	default:
		return "", fmt.Errorf("暂不支持 %s/%s 自动安装节点协议引擎", runtime.GOOS, runtime.GOARCH)
	}
	downloadURL := "https://github.com/XTLS/Xray-core/releases/latest/download/" + assetName
	dlReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	dlResp, err := http.DefaultClient.Do(dlReq)
	if err != nil {
		return "", err
	}
	defer dlResp.Body.Close()
	if dlResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载节点协议引擎失败: HTTP %d", dlResp.StatusCode)
	}
	archive, err := os.CreateTemp("", "xray-*.zip")
	if err != nil {
		return "", err
	}
	archivePath := archive.Name()
	defer os.Remove(archivePath)
	if _, err := io.Copy(archive, dlResp.Body); err != nil {
		archive.Close()
		return "", err
	}
	if err := archive.Close(); err != nil {
		return "", err
	}
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	expected := "xray"
	if runtime.GOOS == "windows" {
		expected = "xray.exe"
	}
	for _, entry := range reader.File {
		if !strings.EqualFold(filepath.Base(entry.Name), expected) {
			continue
		}
		input, err := entry.Open()
		if err != nil {
			return "", err
		}
		output, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
		if err != nil {
			input.Close()
			return "", err
		}
		_, copyErr := io.Copy(output, input)
		input.Close()
		output.Close()
		if copyErr != nil {
			return "", copyErr
		}
		return target, nil
	}
	return "", errors.New("下载包中没有找到 xray 可执行文件")
}
