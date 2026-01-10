package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"time"

	"easy_proxies/internal/app"
	"easy_proxies/internal/config"

	"gopkg.in/yaml.v3"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "path to config file")
	flag.Parse()

	if err := ensureDefaultConfig(configPath); err != nil {
		log.Fatalf("ensure default config: %v", err)
	}

	logStartupHints(configPath)

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	logSubscriptionURL(cfg)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := app.Run(ctx, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "proxy pool exited with error: %v\n", err)
		os.Exit(1)
	}
}

func ensureDefaultConfig(path string) error {
	if _, err := os.Stat(path); err == nil {
		return ensurePlaceholders(path)
	} else if !os.IsNotExist(err) {
		return err
	}

	defaultConfig := `mode: pool
listener:
  address: "0.0.0.0"
  port: 60009
  username: "admin"
  password: "Admin@123.."
nodes:
  - name: "local-placeholder"
    uri: "vless://00000000-0000-0000-0000-000000000000@127.0.0.1:1"
subscriptions:
  - "https://example.com/subscription"
management:
  listen: "127.0.0.1:9090"
  password: "Admin@123.."
`
	if err := os.WriteFile(path, []byte(defaultConfig), 0o644); err != nil {
		return err
	}
	log.Printf("config not found, created default config at %s", path)
	return nil
}

func ensurePlaceholders(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	var doc map[string]any
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return err
	}

	nodesEmpty := isEmptySlice(doc["nodes"])
	subsEmpty := isEmptySlice(doc["subscriptions"])
	nodesFileEmpty := isEmptyString(doc["nodes_file"])

	if !(nodesEmpty && subsEmpty && nodesFileEmpty) {
		return nil
	}

	doc["nodes"] = []any{
		map[string]any{
			"name": "local-placeholder",
			"uri":  "vless://00000000-0000-0000-0000-000000000000@127.0.0.1:1",
		},
	}
	doc["subscriptions"] = []any{"https://example.com/subscription"}

	newData, err := yaml.Marshal(doc)
	if err != nil {
		return err
	}
	if err := os.WriteFile(path, newData, 0o644); err != nil {
		return err
	}
	log.Printf("config had no nodes/subscriptions; wrote placeholders to %s", path)
	return nil
}

func isEmptySlice(value any) bool {
	if value == nil {
		return true
	}
	switch v := value.(type) {
	case []any:
		return len(v) == 0
	case []string:
		return len(v) == 0
	default:
		return false
	}
}

func isEmptyString(value any) bool {
	if value == nil {
		return true
	}
	if s, ok := value.(string); ok {
		return s == ""
	}
	return false
}

func logStartupHints(path string) {
	log.Printf("using config: %s", path)
	log.Printf("quick start: edit nodes in %s or set subscriptions/nodes_file", path)
	log.Printf("defaults: listener port 60009, password Admin@123..")
}

func logSubscriptionURL(cfg *config.Config) {
	if cfg == nil || !cfg.ManagementEnabled() {
		return
	}

	host, port := splitListenHostPort(cfg.Management.Listen)
	if port == "" {
		port = "9090"
	}

	exportHost := host
	if isLoopbackOrWildcard(host) {
		exportHost = strings.TrimSpace(cfg.ExternalIP)
		if exportHost == "" {
			if autoIP := fetchExternalIP(); autoIP != "" {
				exportHost = autoIP
			}
		}
	}

	if exportHost == "" {
		log.Printf("subscription url: unavailable (set external_ip in config or web settings)")
		return
	}

	log.Printf("subscription url: http://%s:%s/api/export", exportHost, port)
}

func fetchExternalIP() string {
	client := &http.Client{Timeout: 3 * time.Second}
	req, err := http.NewRequest(http.MethodGet, "https://api.ipify.org?format=text", nil)
	if err != nil {
		return ""
	}
	resp, err := client.Do(req)
	if err != nil {
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return ""
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return ""
	}
	ip := strings.TrimSpace(string(body))
	if net.ParseIP(ip) == nil {
		return ""
	}
	return ip
}

func splitListenHostPort(value string) (string, string) {
	host, port, err := net.SplitHostPort(strings.TrimSpace(value))
	if err == nil {
		return host, port
	}
	return strings.TrimSpace(value), ""
}

func isLoopbackOrWildcard(host string) bool {
	if host == "" || host == "localhost" {
		return true
	}
	if host == "0.0.0.0" || host == "::" {
		return true
	}
	if ip := net.ParseIP(host); ip != nil {
		return ip.IsLoopback() || ip.IsUnspecified()
	}
	return false
}
