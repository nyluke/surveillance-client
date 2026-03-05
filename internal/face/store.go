package face

import (
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"

	"github.com/google/uuid"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Embedding conversion helpers

func EncodeEmbedding(floats []float64) []byte {
	buf := make([]byte, len(floats)*8)
	for i, f := range floats {
		binary.LittleEndian.PutUint64(buf[i*8:], math.Float64bits(f))
	}
	return buf
}

func DecodeEmbedding(b []byte) []float64 {
	n := len(b) / 8
	floats := make([]float64, n)
	for i := 0; i < n; i++ {
		floats[i] = math.Float64frombits(binary.LittleEndian.Uint64(b[i*8:]))
	}
	return floats
}

// Subject CRUD

func (s *Store) ListSubjects() ([]Subject, error) {
	rows, err := s.db.Query(`SELECT id, name, notes, alert_enabled, created_at FROM face_subjects ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subjects []Subject
	for rows.Next() {
		var sub Subject
		if err := rows.Scan(&sub.ID, &sub.Name, &sub.Notes, &sub.AlertEnabled, &sub.CreatedAt); err != nil {
			return nil, err
		}
		subjects = append(subjects, sub)
	}
	if subjects == nil {
		subjects = []Subject{}
	}
	return subjects, rows.Err()
}

func (s *Store) GetSubject(id string) (*Subject, error) {
	var sub Subject
	err := s.db.QueryRow(`SELECT id, name, notes, alert_enabled, created_at FROM face_subjects WHERE id = ?`, id).
		Scan(&sub.ID, &sub.Name, &sub.Notes, &sub.AlertEnabled, &sub.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &sub, nil
}

func (s *Store) CreateSubject(name, notes string) (*Subject, error) {
	id := uuid.New().String()
	_, err := s.db.Exec(`INSERT INTO face_subjects (id, name, notes) VALUES (?, ?, ?)`, id, name, notes)
	if err != nil {
		return nil, err
	}
	return s.GetSubject(id)
}

func (s *Store) DeleteSubject(id string) error {
	result, err := s.db.Exec("DELETE FROM face_subjects WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("subject not found: %s", id)
	}
	return nil
}

// Embedding CRUD

func (s *Store) AddEmbedding(subjectID string, embedding []byte, cropPath string) (*Embedding, error) {
	id := uuid.New().String()
	_, err := s.db.Exec(`INSERT INTO face_embeddings (id, subject_id, embedding, crop_path) VALUES (?, ?, ?, ?)`,
		id, subjectID, embedding, cropPath)
	if err != nil {
		return nil, err
	}
	return &Embedding{ID: id, SubjectID: subjectID, Embedding: embedding, CropPath: cropPath}, nil
}

func (s *Store) GetSubjectEmbeddings(subjectID string) ([]Embedding, error) {
	rows, err := s.db.Query(`SELECT id, subject_id, embedding, crop_path, created_at FROM face_embeddings WHERE subject_id = ?`, subjectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var embeddings []Embedding
	for rows.Next() {
		var e Embedding
		if err := rows.Scan(&e.ID, &e.SubjectID, &e.Embedding, &e.CropPath, &e.CreatedAt); err != nil {
			return nil, err
		}
		embeddings = append(embeddings, e)
	}
	return embeddings, rows.Err()
}

func (s *Store) GetFirstEmbeddingCrop(subjectID string) (string, error) {
	var cropPath string
	err := s.db.QueryRow(`SELECT crop_path FROM face_embeddings WHERE subject_id = ? ORDER BY created_at LIMIT 1`, subjectID).Scan(&cropPath)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return cropPath, err
}

// Gallery - all embeddings with subject info for Python service

func (s *Store) GetGallery() ([]GalleryEntry, error) {
	rows, err := s.db.Query(`
		SELECT e.subject_id, e.id, s.name, s.alert_enabled, e.embedding
		FROM face_embeddings e
		JOIN face_subjects s ON e.subject_id = s.id
		ORDER BY s.name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var gallery []GalleryEntry
	for rows.Next() {
		var g GalleryEntry
		var embBytes []byte
		if err := rows.Scan(&g.SubjectID, &g.EmbeddingID, &g.Name, &g.AlertEnabled, &embBytes); err != nil {
			return nil, err
		}
		g.Embedding = DecodeEmbedding(embBytes)
		gallery = append(gallery, g)
	}
	if gallery == nil {
		gallery = []GalleryEntry{}
	}
	return gallery, rows.Err()
}

// Sighting CRUD

func (s *Store) CreateSighting(subjectID, cameraID string, confidence float64, cropPath string) (*Sighting, error) {
	id := uuid.New().String()
	_, err := s.db.Exec(`INSERT INTO face_sightings (id, subject_id, camera_id, confidence, crop_path) VALUES (?, ?, ?, ?, ?)`,
		id, subjectID, cameraID, confidence, cropPath)
	if err != nil {
		return nil, err
	}
	return &Sighting{ID: id, SubjectID: subjectID, CameraID: cameraID, Confidence: confidence, CropPath: cropPath}, nil
}

func (s *Store) ListSightings(limit int) ([]SightingWithDetails, error) {
	if limit <= 0 {
		limit = 100
	}
	rows, err := s.db.Query(`
		SELECT si.id, si.subject_id, si.camera_id, si.confidence, si.crop_path, si.seen_at,
		       COALESCE(su.name, 'Unknown') as subject_name,
		       COALESCE(c.name, 'Unknown') as camera_name
		FROM face_sightings si
		LEFT JOIN face_subjects su ON si.subject_id = su.id
		LEFT JOIN cameras c ON si.camera_id = c.id
		ORDER BY si.seen_at DESC
		LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var sightings []SightingWithDetails
	for rows.Next() {
		var si SightingWithDetails
		if err := rows.Scan(&si.ID, &si.SubjectID, &si.CameraID, &si.Confidence, &si.CropPath, &si.SeenAt,
			&si.SubjectName, &si.CameraName); err != nil {
			return nil, err
		}
		sightings = append(sightings, si)
	}
	if sightings == nil {
		sightings = []SightingWithDetails{}
	}
	return sightings, rows.Err()
}

// Visitor CRUD

func (s *Store) CreateVisitor(cameraID string, embedding []byte, cropPath string) (*Visitor, error) {
	id := uuid.New().String()
	_, err := s.db.Exec(`INSERT INTO face_visitors (id, camera_id, embedding, crop_path) VALUES (?, ?, ?, ?)`,
		id, cameraID, embedding, cropPath)
	if err != nil {
		return nil, err
	}
	return &Visitor{ID: id, CameraID: cameraID, Embedding: embedding, CropPath: cropPath}, nil
}

func (s *Store) ListVisitorsForClustering() ([]Visitor, error) {
	rows, err := s.db.Query(`SELECT id, cluster_id, camera_id, embedding, crop_path, seen_at FROM face_visitors ORDER BY seen_at DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var visitors []Visitor
	for rows.Next() {
		var v Visitor
		var clusterID sql.NullString
		if err := rows.Scan(&v.ID, &clusterID, &v.CameraID, &v.Embedding, &v.CropPath, &v.SeenAt); err != nil {
			return nil, err
		}
		if clusterID.Valid {
			v.ClusterID = clusterID.String
		}
		visitors = append(visitors, v)
	}
	return visitors, rows.Err()
}

func (s *Store) UpdateVisitorCluster(visitorID, clusterID string) error {
	_, err := s.db.Exec(`UPDATE face_visitors SET cluster_id = ? WHERE id = ?`, clusterID, visitorID)
	return err
}

// Cluster CRUD

func (s *Store) UpsertCluster(id, label, firstSeen, lastSeen string, visitCount int, representativeCrop string) error {
	_, err := s.db.Exec(`
		INSERT INTO face_clusters (id, label, first_seen, last_seen, visit_count, representative_crop)
		VALUES (?, ?, ?, ?, ?, ?)
		ON CONFLICT(id) DO UPDATE SET
			label = COALESCE(excluded.label, face_clusters.label),
			first_seen = excluded.first_seen,
			last_seen = excluded.last_seen,
			visit_count = excluded.visit_count,
			representative_crop = excluded.representative_crop`,
		id, label, firstSeen, lastSeen, visitCount, representativeCrop)
	return err
}

func (s *Store) ListClusters() ([]Cluster, error) {
	rows, err := s.db.Query(`SELECT id, label, first_seen, last_seen, visit_count, representative_crop FROM face_clusters ORDER BY last_seen DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var clusters []Cluster
	for rows.Next() {
		var c Cluster
		var label, repCrop sql.NullString
		if err := rows.Scan(&c.ID, &label, &c.FirstSeen, &c.LastSeen, &c.VisitCount, &repCrop); err != nil {
			return nil, err
		}
		if label.Valid {
			c.Label = label.String
		}
		if repCrop.Valid {
			c.RepresentativeCrop = repCrop.String
		}
		clusters = append(clusters, c)
	}
	if clusters == nil {
		clusters = []Cluster{}
	}
	return clusters, rows.Err()
}

func (s *Store) UpdateClusterLabel(id, label string) error {
	result, err := s.db.Exec(`UPDATE face_clusters SET label = ? WHERE id = ?`, label, id)
	if err != nil {
		return err
	}
	n, _ := result.RowsAffected()
	if n == 0 {
		return fmt.Errorf("cluster not found: %s", id)
	}
	return nil
}

func (s *Store) ClearClusters() error {
	_, err := s.db.Exec(`DELETE FROM face_clusters`)
	if err != nil {
		return err
	}
	_, err = s.db.Exec(`UPDATE face_visitors SET cluster_id = NULL`)
	return err
}

// Monitor Config CRUD

func (s *Store) ListMonitorConfigs() ([]MonitorConfig, error) {
	rows, err := s.db.Query(`SELECT camera_id, monitor_type, interval_seconds FROM face_monitor_config`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []MonitorConfig
	for rows.Next() {
		var c MonitorConfig
		if err := rows.Scan(&c.CameraID, &c.MonitorType, &c.IntervalSeconds); err != nil {
			return nil, err
		}
		configs = append(configs, c)
	}
	if configs == nil {
		configs = []MonitorConfig{}
	}
	return configs, rows.Err()
}

func (s *Store) SetMonitorConfigs(configs []MonitorConfig) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`DELETE FROM face_monitor_config`); err != nil {
		return err
	}

	for _, c := range configs {
		interval := c.IntervalSeconds
		if interval <= 0 {
			interval = 2
		}
		if _, err := tx.Exec(`INSERT INTO face_monitor_config (camera_id, monitor_type, interval_seconds) VALUES (?, ?, ?)`,
			c.CameraID, c.MonitorType, interval); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// Alert CRUD

func (s *Store) CreateAlert(sightingID string, success bool, errorMsg string) error {
	id := uuid.New().String()
	_, err := s.db.Exec(`INSERT INTO face_alerts (id, sighting_id, success, error_message) VALUES (?, ?, ?, ?)`,
		id, sightingID, success, errorMsg)
	return err
}
