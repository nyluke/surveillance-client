package go2rtc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"surveillance-client/internal/camera"
)

// GenerateConfig creates a go2rtc.yaml config file from the camera list
func GenerateConfig(cameras []camera.Camera, configPath string) error {
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}

	var b strings.Builder

	b.WriteString("log:\n")
	b.WriteString("  level: trace\n")
	b.WriteString("\n")
	b.WriteString("api:\n")
	b.WriteString("  listen: \":1984\"\n")
	b.WriteString("  origin: \"*\"\n")
	b.WriteString("\n")

	b.WriteString("streams:\n")
	for _, cam := range cameras {
		rtspURL := cam.RTSPMain
		if cam.Username != nil && cam.Password != nil && *cam.Username != "" {
			rtspURL = injectCredentials(rtspURL, *cam.Username, *cam.Password)
		}
		b.WriteString(fmt.Sprintf("  cam_%s: \"exec:ffmpeg -rtsp_transport tcp -i %s -c:v copy -an -f mpegts -\"\n", cam.ID, rtspURL))

		if cam.RTSPSub != nil && *cam.RTSPSub != "" {
			subURL := *cam.RTSPSub
			if cam.Username != nil && cam.Password != nil && *cam.Username != "" {
				subURL = injectCredentials(subURL, *cam.Username, *cam.Password)
			}
			b.WriteString(fmt.Sprintf("  cam_%s_sub: \"exec:ffmpeg -rtsp_transport tcp -i %s -c:v copy -an -f mpegts -\"\n", cam.ID, subURL))
		}
	}

	return os.WriteFile(configPath, []byte(b.String()), 0644)
}

// injectCredentials adds user:pass to an RTSP URL
func injectCredentials(rtspURL, username, password string) string {
	if strings.Contains(rtspURL, "@") {
		return rtspURL
	}
	return strings.Replace(rtspURL, "rtsp://", fmt.Sprintf("rtsp://%s:%s@", username, password), 1)
}
