package camera

import (
	"database/sql"
	"fmt"

	"github.com/google/uuid"
)

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// Camera CRUD

func (s *Store) ListCameras() ([]Camera, error) {
	rows, err := s.db.Query(`
		SELECT id, name, onvif_address, rtsp_main, rtsp_sub, username, password,
		       enabled, sort_order, created_at, updated_at
		FROM cameras ORDER BY sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cameras []Camera
	for rows.Next() {
		var c Camera
		if err := rows.Scan(&c.ID, &c.Name, &c.ONVIFAddress, &c.RTSPMain, &c.RTSPSub,
			&c.Username, &c.Password, &c.Enabled, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		cameras = append(cameras, c)
	}
	if cameras == nil {
		cameras = []Camera{}
	}
	return cameras, rows.Err()
}

func (s *Store) GetCamera(id string) (*Camera, error) {
	var c Camera
	err := s.db.QueryRow(`
		SELECT id, name, onvif_address, rtsp_main, rtsp_sub, username, password,
		       enabled, sort_order, created_at, updated_at
		FROM cameras WHERE id = ?`, id).Scan(
		&c.ID, &c.Name, &c.ONVIFAddress, &c.RTSPMain, &c.RTSPSub,
		&c.Username, &c.Password, &c.Enabled, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &c, nil
}

func (s *Store) CreateCamera(req CreateCameraRequest) (*Camera, error) {
	id := uuid.New().String()
	enabled := true
	if req.Enabled != nil {
		enabled = *req.Enabled
	}
	sortOrder := 0
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}

	_, err := s.db.Exec(`
		INSERT INTO cameras (id, name, onvif_address, rtsp_main, rtsp_sub, username, password, enabled, sort_order)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		id, req.Name, req.ONVIFAddress, req.RTSPMain, req.RTSPSub, req.Username, req.Password, enabled, sortOrder)
	if err != nil {
		return nil, err
	}

	return s.GetCamera(id)
}

func (s *Store) UpdateCamera(id string, req UpdateCameraRequest) (*Camera, error) {
	existing, err := s.GetCamera(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("camera not found: %s", id)
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.ONVIFAddress != nil {
		existing.ONVIFAddress = req.ONVIFAddress
	}
	if req.RTSPMain != nil {
		existing.RTSPMain = *req.RTSPMain
	}
	if req.RTSPSub != nil {
		existing.RTSPSub = req.RTSPSub
	}
	if req.Username != nil {
		existing.Username = req.Username
	}
	if req.Password != nil {
		existing.Password = req.Password
	}
	if req.Enabled != nil {
		existing.Enabled = *req.Enabled
	}
	if req.SortOrder != nil {
		existing.SortOrder = *req.SortOrder
	}

	_, err = s.db.Exec(`
		UPDATE cameras SET name=?, onvif_address=?, rtsp_main=?, rtsp_sub=?,
		       username=?, password=?, enabled=?, sort_order=?, updated_at=datetime('now')
		WHERE id=?`,
		existing.Name, existing.ONVIFAddress, existing.RTSPMain, existing.RTSPSub,
		existing.Username, existing.Password, existing.Enabled, existing.SortOrder, id)
	if err != nil {
		return nil, err
	}

	return s.GetCamera(id)
}

func (s *Store) DeleteCamera(id string) error {
	result, err := s.db.Exec("DELETE FROM cameras WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("camera not found: %s", id)
	}
	return nil
}

// Group CRUD

func (s *Store) ListGroups() ([]Group, error) {
	rows, err := s.db.Query("SELECT id, name, sort_order FROM groups ORDER BY sort_order, name")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groups []Group
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.ID, &g.Name, &g.SortOrder); err != nil {
			return nil, err
		}
		groups = append(groups, g)
	}
	if groups == nil {
		groups = []Group{}
	}
	return groups, rows.Err()
}

func (s *Store) GetGroup(id string) (*Group, error) {
	var g Group
	err := s.db.QueryRow("SELECT id, name, sort_order FROM groups WHERE id = ?", id).
		Scan(&g.ID, &g.Name, &g.SortOrder)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &g, nil
}

func (s *Store) CreateGroup(req CreateGroupRequest) (*Group, error) {
	id := uuid.New().String()
	sortOrder := 0
	if req.SortOrder != nil {
		sortOrder = *req.SortOrder
	}

	_, err := s.db.Exec("INSERT INTO groups (id, name, sort_order) VALUES (?, ?, ?)",
		id, req.Name, sortOrder)
	if err != nil {
		return nil, err
	}

	return s.GetGroup(id)
}

func (s *Store) UpdateGroup(id string, req UpdateGroupRequest) (*Group, error) {
	existing, err := s.GetGroup(id)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		return nil, fmt.Errorf("group not found: %s", id)
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.SortOrder != nil {
		existing.SortOrder = *req.SortOrder
	}

	_, err = s.db.Exec("UPDATE groups SET name=?, sort_order=? WHERE id=?",
		existing.Name, existing.SortOrder, id)
	if err != nil {
		return nil, err
	}

	return s.GetGroup(id)
}

func (s *Store) DeleteGroup(id string) error {
	result, err := s.db.Exec("DELETE FROM groups WHERE id = ?", id)
	if err != nil {
		return err
	}
	n, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return fmt.Errorf("group not found: %s", id)
	}
	return nil
}

// Camera-Group associations

func (s *Store) GetCameraGroups(cameraID string) ([]string, error) {
	rows, err := s.db.Query("SELECT group_id FROM camera_groups WHERE camera_id = ?", cameraID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

func (s *Store) GetGroupCameras(groupID string) ([]Camera, error) {
	rows, err := s.db.Query(`
		SELECT c.id, c.name, c.onvif_address, c.rtsp_main, c.rtsp_sub, c.username, c.password,
		       c.enabled, c.sort_order, c.created_at, c.updated_at
		FROM cameras c
		JOIN camera_groups cg ON c.id = cg.camera_id
		WHERE cg.group_id = ?
		ORDER BY c.sort_order, c.name`, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cameras []Camera
	for rows.Next() {
		var c Camera
		if err := rows.Scan(&c.ID, &c.Name, &c.ONVIFAddress, &c.RTSPMain, &c.RTSPSub,
			&c.Username, &c.Password, &c.Enabled, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		cameras = append(cameras, c)
	}
	if cameras == nil {
		cameras = []Camera{}
	}
	return cameras, rows.Err()
}

func (s *Store) SetCameraGroups(cameraID string, groupIDs []string) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec("DELETE FROM camera_groups WHERE camera_id = ?", cameraID); err != nil {
		return err
	}

	for _, gid := range groupIDs {
		if _, err := tx.Exec("INSERT INTO camera_groups (camera_id, group_id) VALUES (?, ?)",
			cameraID, gid); err != nil {
			return err
		}
	}

	return tx.Commit()
}

// ListEnabled returns only enabled cameras (used by go2rtc sync)
func (s *Store) ListEnabled() ([]Camera, error) {
	rows, err := s.db.Query(`
		SELECT id, name, onvif_address, rtsp_main, rtsp_sub, username, password,
		       enabled, sort_order, created_at, updated_at
		FROM cameras WHERE enabled = 1 ORDER BY sort_order, name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var cameras []Camera
	for rows.Next() {
		var c Camera
		if err := rows.Scan(&c.ID, &c.Name, &c.ONVIFAddress, &c.RTSPMain, &c.RTSPSub,
			&c.Username, &c.Password, &c.Enabled, &c.SortOrder, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		cameras = append(cameras, c)
	}
	if cameras == nil {
		cameras = []Camera{}
	}
	return cameras, rows.Err()
}
