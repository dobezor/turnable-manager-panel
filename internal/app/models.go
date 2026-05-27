package app

import "time"

type State struct {
	Version       int       `json:"version"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
	AdminUsername string    `json:"admin_username"`
	AdminPassword string    `json:"admin_password"`
	Settings      Settings  `json:"settings"`
	Routes        []Route   `json:"routes"`
	Users         []User    `json:"users"`
	LastApplyAt   time.Time `json:"last_apply_at,omitempty"`
}

type Settings struct {
	PlatformID      string `json:"platform_id"`
	CallID          string `json:"call_id"`
	PrivateKey      string `json:"priv_key"`
	PublicKey       string `json:"pub_key"`
	RelayEnabled    bool   `json:"relay_enabled"`
	RelayProto      string `json:"relay_proto"`
	RelayCloak      string `json:"relay_cloak"`
	RelayPublicIP   string `json:"relay_public_ip"`
	RelayPort       int    `json:"relay_port"`
	P2PEnabled      bool   `json:"p2p_enabled"`
	P2PUsername     string `json:"p2p_username"`
	P2PCloak        string `json:"p2p_cloak"`
	WorkDir         string `json:"work_dir"`
	TurnableBinary  string `json:"turnable_binary"`
	TurnableService string `json:"turnable_service"`
	XrayService     string `json:"xray_service"`
}

type Route struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Address    string `json:"address"`
	Port       int    `json:"port"`
	Socket     string `json:"socket"`
	Transport  string `json:"transport"`
	Encryption string `json:"encryption"`
	Conn       string `json:"conn,omitempty"`
	Enabled    bool   `json:"enabled"`
	Notes      string `json:"notes,omitempty"`
}

type User struct {
	UUID          string    `json:"uuid"`
	Username      string    `json:"username"`
	Type          string    `json:"type"`
	Peers         int       `json:"peers"`
	AllowedRoutes []string  `json:"allowed_routes"`
	Disabled      bool      `json:"disabled"`
	ExpiresAt     time.Time `json:"expires_at,omitempty"`
	TrafficGB     int       `json:"traffic_gb,omitempty"`
	Notes         string    `json:"notes,omitempty"`
	CreatedAt     time.Time `json:"created_at"`
}

func DefaultState(adminHash string) State {
	now := time.Now().UTC()
	return State{
		Version:       1,
		CreatedAt:     now,
		UpdatedAt:     now,
		AdminUsername: "admin",
		AdminPassword: adminHash,
		Settings: Settings{
			PlatformID:      "vk.com",
			RelayEnabled:    true,
			RelayProto:      "dtls",
			RelayCloak:      "none",
			RelayPort:       56000,
			P2PEnabled:      false,
			P2PCloak:        "none",
			WorkDir:         "/etc/turnable",
			TurnableBinary:  "/usr/local/bin/turnable",
			TurnableService: "turnable",
			XrayService:     "xray",
		},
		Routes: []Route{
			{
				ID:         "xray-tcp",
				Name:       "Local Xray client TCP",
				Address:    "127.0.0.1",
				Port:       10808,
				Socket:     "tcp",
				Transport:  "kcp",
				Encryption: "handshake",
				Enabled:    true,
				Notes:      "Route to local Xray/SOCKS/TCP service on RU server, then Xray sends traffic to NL VLESS.",
			},
		},
		Users: []User{},
	}
}
