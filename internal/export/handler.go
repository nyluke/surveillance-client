package export

import (
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
)

type Handler struct{}

func NewHandler() *Handler {
	return &Handler{}
}

func (h *Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Limit to 500MB
	r.Body = http.MaxBytesReader(w, r.Body, 500<<20)

	tmpDir, err := os.MkdirTemp("", "dvr-export-*")
	if err != nil {
		http.Error(w, "failed to create temp dir", http.StatusInternalServerError)
		return
	}
	defer os.RemoveAll(tmpDir)

	aviPath := filepath.Join(tmpDir, "input.avi")
	mp4Path := filepath.Join(tmpDir, "output.mp4")

	// Write uploaded AVI to temp file
	aviFile, err := os.Create(aviPath)
	if err != nil {
		http.Error(w, "failed to create temp file", http.StatusInternalServerError)
		return
	}
	n, err := io.Copy(aviFile, r.Body)
	aviFile.Close()
	if err != nil {
		http.Error(w, "failed to read upload", http.StatusInternalServerError)
		return
	}
	log.Printf("Export remux: received %d bytes AVI", n)

	// Remux AVI → MP4 with ffmpeg (copy streams, no re-encoding)
	cmd := exec.Command("ffmpeg",
		"-i", aviPath,
		"-c", "copy",
		"-movflags", "+faststart",
		"-y",
		mp4Path,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		log.Printf("ffmpeg remux failed: %v\n%s", err, output)
		http.Error(w, "ffmpeg remux failed — is ffmpeg installed?", http.StatusInternalServerError)
		return
	}

	mp4File, err := os.Open(mp4Path)
	if err != nil {
		http.Error(w, "failed to open converted file", http.StatusInternalServerError)
		return
	}
	defer mp4File.Close()

	stat, _ := mp4File.Stat()
	log.Printf("Export remux: %d bytes AVI → %d bytes MP4", n, stat.Size())

	w.Header().Set("Content-Type", "video/mp4")
	io.Copy(w, mp4File)
}
