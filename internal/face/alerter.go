package face

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"
)

type Alerter struct {
	webhookURL string
	store      *Store
	mu         sync.Mutex
	lastAlert  map[string]time.Time // subject_id -> last alert time
	debounce   time.Duration
}

func NewAlerter(webhookURL string, store *Store) *Alerter {
	return &Alerter{
		webhookURL: webhookURL,
		store:      store,
		lastAlert:  make(map[string]time.Time),
		debounce:   5 * time.Minute,
	}
}

func (a *Alerter) Enabled() bool {
	return a.webhookURL != ""
}

func (a *Alerter) SendAlert(sighting *Sighting, subjectName, cameraName string) {
	if !a.Enabled() {
		return
	}

	a.mu.Lock()
	last, exists := a.lastAlert[sighting.SubjectID]
	if exists && time.Since(last) < a.debounce {
		a.mu.Unlock()
		return
	}
	a.lastAlert[sighting.SubjectID] = time.Now()
	a.mu.Unlock()

	msg := fmt.Sprintf("Face Alert: *%s* spotted on camera *%s* (confidence: %.1f%%)",
		subjectName, cameraName, sighting.Confidence*100)

	payload := map[string]string{"text": msg}
	body, _ := json.Marshal(payload)

	resp, err := http.Post(a.webhookURL, "application/json", bytes.NewReader(body))
	success := err == nil && resp.StatusCode == 200
	errMsg := ""
	if err != nil {
		errMsg = err.Error()
	} else if resp.StatusCode != 200 {
		errMsg = fmt.Sprintf("slack returned %d", resp.StatusCode)
	}
	if resp != nil {
		resp.Body.Close()
	}

	if !success {
		log.Printf("slack alert failed for subject %s: %s", sighting.SubjectID, errMsg)
	}

	if a.store != nil {
		a.store.CreateAlert(sighting.ID, success, errMsg)
	}
}
