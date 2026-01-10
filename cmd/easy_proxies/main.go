package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"easy_proxies/internal/app"
	"easy_proxies/internal/config"
)

func main() {
	var configPath string
	flag.StringVar(&configPath, "config", "config.yaml", "path to config file")
	flag.Parse()

	if err := ensureDefaultConfig(configPath); err != nil {
		log.Fatalf("ensure default config: %v", err)
	}

	cfg, err := config.Load(configPath)
	if err != nil {
		log.Fatalf("load config: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := app.Run(ctx, cfg); err != nil {
		fmt.Fprintf(os.Stderr, "proxy pool exited with error: %v\n", err)
		os.Exit(1)
	}
}

func ensureDefaultConfig(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil
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
