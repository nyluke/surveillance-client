package face

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"time"
)

type SnapshotFetcher struct {
	go2rtcAPI string
}

func NewSnapshotFetcher(go2rtcAPI string) *SnapshotFetcher {
	return &SnapshotFetcher{go2rtcAPI: go2rtcAPI}
}

// FetchJPEGFromRTSP grabs a single JPEG frame from an RTSP stream via ffmpeg.
func (sf *SnapshotFetcher) FetchJPEGFromRTSP(rtspURL string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ffmpeg",
		"-rtsp_transport", "tcp",
		"-i", rtspURL,
		"-frames:v", "1",
		"-f", "image2",
		"-q:v", "3",
		"-y", "pipe:1",
	)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("ffmpeg snapshot: %w: %s", err, lastN(stderr.Bytes(), 200))
	}

	data := stdout.Bytes()
	if len(data) < 1000 {
		return nil, fmt.Errorf("ffmpeg snapshot too small: %d bytes", len(data))
	}
	return data, nil
}

func lastN(b []byte, n int) []byte {
	if len(b) <= n {
		return b
	}
	return b[len(b)-n:]
}
