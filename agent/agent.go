package agent

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
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
	"strings"
	"sublink/node"
	"syscall"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	Version        = "0.2.0"
	singBoxVersion = "1.13.13"
)

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
			if err := reportTaskWithRetry(ctx, client, cfg, report); err != nil {
				fmt.Println("上报任务失败:", err)
			}
			// Give the web UI a short window to receive the result and enqueue
			// the next node, then poll immediately instead of entering idle sleep.
			if !sleepContext(ctx, 2*time.Second) {
				return ctx.Err()
			}
			continue
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

func reportTaskWithRetry(ctx context.Context, client *http.Client, cfg Config, report taskReport) error {
	var lastErr error
	for attempt := 0; attempt < 3; attempt++ {
		if err := reportTask(ctx, client, cfg, report); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if !sleepContext(ctx, time.Duration(attempt+1)*time.Second) {
			return ctx.Err()
		}
	}
	return lastErr
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
	singBox, err := ensureSingBox(ctx)
	if err != nil {
		report.Error = err.Error()
		return report
	}
	result, err := runSingBoxTest(ctx, singBox, nodeLink, testType == "speed")
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

func runSingBoxTest(parent context.Context, binary, nodeLink string, download bool) (speedResult, error) {
	ctx, cancel := context.WithTimeout(parent, 75*time.Second)
	defer cancel()
	port, err := freePort()
	if err != nil {
		return speedResult{}, err
	}
	cfg, err := buildSingBoxConfig(nodeLink, port)
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
	}
	if result.LatencyMs < 0 {
		return result, errors.New("节点延迟测试失败")
	}
	if download {
		result.EgressIP = measureEgressIP(ctx, client)
		bytesRead, duration, err := measureDownload(ctx, client)
		if err != nil {
			return result, err
		}
		result.TestBytes = bytesRead
		result.DownloadMbps = float64(bytesRead) * 8 / duration.Seconds() / 1e6
	}
	return result, nil
}

func buildSingBoxConfig(link string, port int) (map[string]any, error) {
	scheme := strings.ToLower(strings.SplitN(link, "://", 2)[0])
	var outbound map[string]any
	switch scheme {
	case "ss":
		ss, err := node.DecodeSSURL(link)
		if err != nil {
			return nil, fmt.Errorf("解析 SS 节点失败: %w", err)
		}
		outbound = map[string]any{
			"type":        "shadowsocks",
			"tag":         "node",
			"server":      ss.Server,
			"server_port": ss.Port,
			"method":      ss.Param.Cipher,
			"password":    ss.Param.Password,
		}
	case "vless":
		vless, err := node.DecodeVLESSURL(link)
		if err != nil {
			return nil, fmt.Errorf("解析 VLESS 节点失败: %w", err)
		}
		outbound = map[string]any{
			"type":        "vless",
			"tag":         "node",
			"server":      vless.Server,
			"server_port": vless.Port,
			"uuid":        vless.Uuid,
		}
		if vless.Query.Flow != "" {
			outbound["flow"] = vless.Query.Flow
		}
		network := defaultString(vless.Query.Type, "tcp")
		switch vless.Query.Security {
		case "tls":
			tls := map[string]any{
				"enabled":     true,
				"server_name": vless.Query.Sni,
				"utls": map[string]any{
					"enabled":     true,
					"fingerprint": defaultString(vless.Query.Fp, "chrome"),
				},
			}
			if len(vless.Query.Alpn) > 0 {
				tls["alpn"] = vless.Query.Alpn
			}
			outbound["tls"] = tls
		case "reality":
			outbound["tls"] = map[string]any{
				"enabled":     true,
				"server_name": vless.Query.Sni,
				"utls": map[string]any{
					"enabled":     true,
					"fingerprint": defaultString(vless.Query.Fp, "chrome"),
				},
				"reality": map[string]any{
					"enabled":    true,
					"public_key": vless.Query.Pbk,
					"short_id":   vless.Query.Sid,
				},
			}
		case "", "none":
		default:
			return nil, fmt.Errorf("暂不支持 VLESS security=%s", vless.Query.Security)
		}
		switch network {
		case "ws":
			outbound["transport"] = map[string]any{
				"type":    "ws",
				"path":    vless.Query.Path,
				"headers": map[string]any{"Host": vless.Query.Host},
			}
		case "grpc":
			outbound["transport"] = map[string]any{
				"type":         "grpc",
				"service_name": vless.Query.ServiceName,
			}
		case "tcp", "raw":
			if vless.Query.HeaderType != "" && vless.Query.HeaderType != "none" {
				return nil, fmt.Errorf("暂不支持 VLESS TCP header=%s", vless.Query.HeaderType)
			}
		default:
			return nil, fmt.Errorf("暂不支持 VLESS transport=%s", network)
		}
	default:
		return nil, fmt.Errorf("暂不支持 %s 节点测速", scheme)
	}
	return map[string]any{
		"log": map[string]any{"level": "warn"},
		"inbounds": []any{map[string]any{
			"type":        "mixed",
			"tag":         "local",
			"listen":      "127.0.0.1",
			"listen_port": port,
		}},
		"outbounds": []any{outbound},
		"route": map[string]any{
			"final": "node",
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
	probes := []string{
		"https://www.gstatic.com/generate_204",
		"https://cp.cloudflare.com/generate_204",
	}
	var lastErr error
	for _, probeURL := range probes {
		probeCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		req, _ := http.NewRequestWithContext(probeCtx, http.MethodGet, probeURL, nil)
		start := time.Now()
		resp, err := client.Do(req)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			elapsed := time.Since(start).Milliseconds()
			cancel()
			return elapsed
		}
		lastErr = err
		cancel()
	}
	if lastErr != nil {
		fmt.Println("延迟探测失败:", lastErr)
	}
	return -1
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
	urls := []string{
		"https://speed.cloudflare.com/__down?bytes=100000000",
		"https://proof.ovh.net/files/100Mb.dat",
	}
	var lastErr error
	for _, downloadURL := range urls {
		downloadCtx, cancel := context.WithTimeout(ctx, 8*time.Second)
		req, _ := http.NewRequestWithContext(downloadCtx, http.MethodGet, downloadURL, nil)
		req.Header.Set("Accept-Encoding", "identity")
		start := time.Now()
		resp, err := client.Do(req)
		if err != nil {
			lastErr = err
			cancel()
			continue
		}
		n, copyErr := io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		duration := time.Since(start)
		timedOut := downloadCtx.Err() == context.DeadlineExceeded
		cancel()
		if resp.StatusCode/100 != 2 {
			lastErr = fmt.Errorf("HTTP %d", resp.StatusCode)
			continue
		}
		if n == 0 {
			lastErr = errors.New("未收到数据")
			continue
		}
		if copyErr != nil && !timedOut {
			lastErr = copyErr
			continue
		}
		return n, duration, nil
	}
	return 0, 0, fmt.Errorf("下载测速失败: %w", lastErr)
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

func ensureSingBox(ctx context.Context) (string, error) {
	if custom := os.Getenv("SING_BOX_BIN"); custom != "" {
		if _, err := os.Stat(custom); err == nil {
			return custom, nil
		}
	}
	if path, err := exec.LookPath("sing-box"); err == nil {
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
	name := "sing-box"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	target := filepath.Join(targetDir, name)
	if _, err := os.Stat(target); err == nil {
		return target, nil
	}
	return downloadSingBox(ctx, target)
}

func downloadSingBox(ctx context.Context, target string) (string, error) {
	assetName := ""
	switch runtime.GOOS + "/" + runtime.GOARCH {
	case "linux/amd64":
		assetName = fmt.Sprintf("sing-box-%s-linux-amd64.tar.gz", singBoxVersion)
	case "linux/arm64":
		assetName = fmt.Sprintf("sing-box-%s-linux-arm64.tar.gz", singBoxVersion)
	case "windows/amd64":
		assetName = fmt.Sprintf("sing-box-%s-windows-amd64.zip", singBoxVersion)
	default:
		return "", fmt.Errorf("暂不支持 %s/%s 自动安装节点协议引擎", runtime.GOOS, runtime.GOARCH)
	}
	downloadURL := fmt.Sprintf(
		"https://github.com/SagerNet/sing-box/releases/download/v%s/%s",
		singBoxVersion,
		assetName,
	)
	dlReq, _ := http.NewRequestWithContext(ctx, http.MethodGet, downloadURL, nil)
	dlResp, err := http.DefaultClient.Do(dlReq)
	if err != nil {
		return "", err
	}
	defer dlResp.Body.Close()
	if dlResp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载节点协议引擎失败: HTTP %d", dlResp.StatusCode)
	}
	suffix := ".tar.gz"
	if runtime.GOOS == "windows" {
		suffix = ".zip"
	}
	archive, err := os.CreateTemp("", "sing-box-*"+suffix)
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
	expected := "sing-box"
	if runtime.GOOS == "windows" {
		expected = "sing-box.exe"
		reader, err := zip.OpenReader(archivePath)
		if err != nil {
			return "", err
		}
		defer reader.Close()
		for _, entry := range reader.File {
			if strings.EqualFold(filepath.Base(entry.Name), expected) {
				input, openErr := entry.Open()
				if openErr != nil {
					return "", openErr
				}
				defer input.Close()
				return writeExecutable(target, input)
			}
		}
		return "", errors.New("下载包中没有找到 sing-box 可执行文件")
	}

	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gz.Close()
	tarReader := tar.NewReader(gz)
	for {
		header, nextErr := tarReader.Next()
		if nextErr == io.EOF {
			break
		}
		if nextErr != nil {
			return "", nextErr
		}
		if header.Typeflag == tar.TypeReg && filepath.Base(header.Name) == expected {
			return writeExecutable(target, tarReader)
		}
	}
	return "", errors.New("下载包中没有找到 sing-box 可执行文件")
}

func writeExecutable(target string, input io.Reader) (string, error) {
	output, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0755)
	if err != nil {
		return "", err
	}
	if _, err := io.Copy(output, input); err != nil {
		output.Close()
		return "", err
	}
	if err := output.Close(); err != nil {
		return "", err
	}
	return target, nil
}
