package camera

type Camera struct {
	ID           string  `json:"id"`
	Name         string  `json:"name"`
	ONVIFAddress *string `json:"onvif_address,omitempty"`
	RTSPMain     string  `json:"rtsp_main"`
	RTSPSub      *string `json:"rtsp_sub,omitempty"`
	Username     *string `json:"username,omitempty"`
	Password     *string `json:"password,omitempty"`
	Enabled      bool    `json:"enabled"`
	SortOrder    int     `json:"sort_order"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

type CreateCameraRequest struct {
	Name         string  `json:"name"`
	ONVIFAddress *string `json:"onvif_address,omitempty"`
	RTSPMain     string  `json:"rtsp_main"`
	RTSPSub      *string `json:"rtsp_sub,omitempty"`
	Username     *string `json:"username,omitempty"`
	Password     *string `json:"password,omitempty"`
	Enabled      *bool   `json:"enabled,omitempty"`
	SortOrder    *int    `json:"sort_order,omitempty"`
}

type UpdateCameraRequest struct {
	Name         *string `json:"name,omitempty"`
	ONVIFAddress *string `json:"onvif_address,omitempty"`
	RTSPMain     *string `json:"rtsp_main,omitempty"`
	RTSPSub      *string `json:"rtsp_sub,omitempty"`
	Username     *string `json:"username,omitempty"`
	Password     *string `json:"password,omitempty"`
	Enabled      *bool   `json:"enabled,omitempty"`
	SortOrder    *int    `json:"sort_order,omitempty"`
}

type Group struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SortOrder int    `json:"sort_order"`
}

type CreateGroupRequest struct {
	Name      string `json:"name"`
	SortOrder *int   `json:"sort_order,omitempty"`
}

type UpdateGroupRequest struct {
	Name      *string `json:"name,omitempty"`
	SortOrder *int    `json:"sort_order,omitempty"`
}
