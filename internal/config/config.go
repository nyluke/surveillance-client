package config

import (
	"os"
)

type Config struct {
	Port           string
	DBPath         string
	Go2RTCPath     string
	Go2RTCAPI      string
	DVRHost        string
	DVRUsername    string
	DVRPassword    string
	AuthPassword   string
	FaceServiceURL string
	SlackWebhookURL string
	FaceDataDir    string
}

func Load() *Config {
	return &Config{
		Port:       getEnv("PORT", "8080"),
		DBPath:     getEnv("DB_PATH", "data/surveillance.db"),
		Go2RTCPath: getEnv("GO2RTC_PATH", "./go2rtc"),
		Go2RTCAPI:  getEnv("GO2RTC_API", "http://localhost:1984"),
		DVRHost:     getEnv("DVR_HOST", ""),
		DVRUsername: getEnv("DVR_USERNAME", "admin"),
		DVRPassword:  getEnv("DVR_PASSWORD", ""),
		AuthPassword:    getEnv("AUTH_PASSWORD", ""),
		FaceServiceURL:  getEnv("FACE_SERVICE_URL", ""),
		SlackWebhookURL: getEnv("SLACK_WEBHOOK_URL", ""),
		FaceDataDir:     getEnv("FACE_DATA_DIR", "data/faces"),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
