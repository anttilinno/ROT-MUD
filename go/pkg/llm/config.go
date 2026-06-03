package llm

import (
	"os"
	"strconv"
	"time"
)

// Config controls the LLM enhancement layer. Off by default; opt-in per server.
type Config struct {
	Enabled  bool          // master switch
	Endpoint string        // llama.cpp base URL, e.g. http://127.0.0.1:8080
	Model    string        // model name passed through (llama.cpp ignores it but logs it)
	Workers  int           // worker goroutines draining the inbox
	Timeout  time.Duration // per-call latency budget before fallback
	Queue    int           // inbox/result channel capacity
}

// ConfigFromEnv builds a Config from environment variables. The feature is off
// unless ROTMUD_LLM is truthy. Config loading proper (data/config.toml) is not
// wired in the server yet, so env vars keep the blast radius small.
//
//	ROTMUD_LLM=1
//	ROTMUD_LLM_ENDPOINT=http://127.0.0.1:8080
//	ROTMUD_LLM_MODEL=qwen
//	ROTMUD_LLM_TIMEOUT_MS=5000
func ConfigFromEnv() Config {
	c := Config{
		Enabled:  envBool("ROTMUD_LLM"),
		Endpoint: envStr("ROTMUD_LLM_ENDPOINT", "http://127.0.0.1:8080"),
		Model:    envStr("ROTMUD_LLM_MODEL", "qwen"),
		Workers:  envInt("ROTMUD_LLM_WORKERS", 4),
		Timeout:  time.Duration(envInt("ROTMUD_LLM_TIMEOUT_MS", 5000)) * time.Millisecond,
		Queue:    envInt("ROTMUD_LLM_QUEUE", 64),
	}
	return c
}

func envBool(key string) bool {
	switch os.Getenv(key) {
	case "1", "true", "TRUE", "yes", "on":
		return true
	}
	return false
}

func envStr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
