package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

const (
	DefaultDeliveryCacheSize = 10000
	DefaultMaxBodyBytes      = int64(10 << 20)
	DefaultPort              = "8080"
)

type Config struct {
	Port                string
	GitHubWebhookSecret string
	TelegramBotToken    string
	TelegramChatID      string
	PublicBaseURL       string
	AllowedRepos        []string
	MaxBodyBytes        int64
	DeliveryCacheSize   int
}

func Load() (Config, error) {
	return LoadFromLookup(os.LookupEnv)
}

func LoadFromLookup(lookup func(string) (string, bool)) (Config, error) {
	cfg := Config{
		Port:              valueOrDefault(lookup, "PORT", DefaultPort),
		PublicBaseURL:     strings.TrimSpace(value(lookup, "PUBLIC_BASE_URL")),
		MaxBodyBytes:      DefaultMaxBodyBytes,
		DeliveryCacheSize: DefaultDeliveryCacheSize,
	}

	cfg.GitHubWebhookSecret = strings.TrimSpace(value(lookup, "GITHUB_WEBHOOK_SECRET"))
	cfg.TelegramBotToken = strings.TrimSpace(value(lookup, "TELEGRAM_BOT_TOKEN"))
	cfg.TelegramChatID = strings.TrimSpace(value(lookup, "TELEGRAM_CHAT_ID"))
	cfg.AllowedRepos = parseCSV(value(lookup, "ALLOWED_REPOS"))

	var missing []string
	if cfg.GitHubWebhookSecret == "" {
		missing = append(missing, "GITHUB_WEBHOOK_SECRET")
	}
	if cfg.TelegramBotToken == "" {
		missing = append(missing, "TELEGRAM_BOT_TOKEN")
	}
	if cfg.TelegramChatID == "" {
		missing = append(missing, "TELEGRAM_CHAT_ID")
	}
	if len(cfg.AllowedRepos) == 0 {
		missing = append(missing, "ALLOWED_REPOS")
	}
	if len(missing) > 0 {
		return Config{}, fmt.Errorf("missing required environment variables: %s", strings.Join(missing, ", "))
	}

	if raw := strings.TrimSpace(value(lookup, "MAX_BODY_BYTES")); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil || n <= 0 {
			return Config{}, fmt.Errorf("MAX_BODY_BYTES must be a positive integer")
		}
		cfg.MaxBodyBytes = n
	}

	if raw := strings.TrimSpace(value(lookup, "DELIVERY_CACHE_SIZE")); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil || n <= 0 {
			return Config{}, fmt.Errorf("DELIVERY_CACHE_SIZE must be a positive integer")
		}
		cfg.DeliveryCacheSize = n
	}

	return cfg, nil
}

func (c Config) Address() string {
	if strings.HasPrefix(c.Port, ":") {
		return c.Port
	}
	return ":" + c.Port
}

func (c Config) RepoAllowed(repo string) bool {
	repo = strings.TrimSpace(repo)
	if repo == "" {
		return false
	}

	repoLower := strings.ToLower(repo)
	for _, pattern := range c.AllowedRepos {
		pattern = strings.TrimSpace(pattern)
		if pattern == "*" {
			return true
		}

		patternLower := strings.ToLower(pattern)
		if strings.HasSuffix(patternLower, "/*") {
			owner := strings.TrimSuffix(patternLower, "/*")
			if strings.HasPrefix(repoLower, owner+"/") {
				return true
			}
			continue
		}

		if repoLower == patternLower {
			return true
		}
	}

	return false
}

func parseCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	seen := make(map[string]struct{}, len(parts))
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key := strings.ToLower(part)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, part)
	}
	return out
}

func value(lookup func(string) (string, bool), key string) string {
	v, _ := lookup(key)
	return v
}

func valueOrDefault(lookup func(string) (string, bool), key, fallback string) string {
	if v := strings.TrimSpace(value(lookup, key)); v != "" {
		return v
	}
	return fallback
}
