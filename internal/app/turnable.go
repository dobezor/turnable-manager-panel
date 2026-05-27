package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

type turnableConfig struct {
	PlatformID string         `json:"platform_id"`
	CallID     string         `json:"call_id"`
	PrivateKey string         `json:"priv_key"`
	PublicKey  string         `json:"pub_key"`
	Relay      relayConfig    `json:"relay"`
	P2P        p2pConfig      `json:"p2p"`
	Provider   providerConfig `json:"provider"`
}

type relayConfig struct {
	Enabled  bool   `json:"enabled"`
	Proto    string `json:"proto"`
	Cloak    string `json:"cloak"`
	PublicIP string `json:"public_ip"`
	Port     int    `json:"port"`
}

type p2pConfig struct {
	Enabled  bool   `json:"enabled"`
	Username string `json:"username"`
	Cloak    string `json:"cloak"`
}

type providerConfig struct {
	Type string `json:"type"`
	Path string `json:"path"`
}

type turnableStore struct {
	Routes []RouteExport `json:"routes"`
	Users  []UserExport  `json:"users"`
}

type RouteExport struct {
	ID         string `json:"id"`
	Address    string `json:"address"`
	Port       int    `json:"port"`
	Socket     string `json:"socket"`
	Transport  string `json:"transport"`
	Encryption string `json:"encryption"`
	Name       string `json:"name"`
	Conn       string `json:"conn,omitempty"`
}

type UserExport struct {
	UUID          string   `json:"uuid"`
	AllowedRoutes []string `json:"allowed_routes"`
	Username      string   `json:"username"`
	Type          string   `json:"type"`
	Peers         int      `json:"peers"`
}

func ApplyTurnableFiles(st *State) error {
	if strings.TrimSpace(st.Settings.WorkDir) == "" {
		return fmt.Errorf("work dir is empty")
	}
	if err := os.MkdirAll(st.Settings.WorkDir, 0700); err != nil {
		return err
	}

	cfg := turnableConfig{
		PlatformID: st.Settings.PlatformID,
		CallID:     st.Settings.CallID,
		PrivateKey: st.Settings.PrivateKey,
		PublicKey:  st.Settings.PublicKey,
		Relay: relayConfig{
			Enabled:  st.Settings.RelayEnabled,
			Proto:    st.Settings.RelayProto,
			Cloak:    st.Settings.RelayCloak,
			PublicIP: st.Settings.RelayPublicIP,
			Port:     st.Settings.RelayPort,
		},
		P2P: p2pConfig{
			Enabled:  st.Settings.P2PEnabled,
			Username: st.Settings.P2PUsername,
			Cloak:    st.Settings.P2PCloak,
		},
		Provider: providerConfig{Type: "json", Path: "store.json"},
	}

	var exportedRoutes []RouteExport
	routeEnabled := map[string]bool{}
	for _, r := range st.Routes {
		if !r.Enabled {
			continue
		}
		routeEnabled[r.ID] = true
		exportedRoutes = append(exportedRoutes, RouteExport{
			ID:         r.ID,
			Address:    r.Address,
			Port:       r.Port,
			Socket:     or(r.Socket, "tcp"),
			Transport:  or(r.Transport, "kcp"),
			Encryption: or(r.Encryption, "handshake"),
			Name:       r.Name,
			Conn:       r.Conn,
		})
	}

	var exportedUsers []UserExport
	for _, u := range st.Users {
		if u.Disabled || u.UUID == "" {
			continue
		}
		if !u.ExpiresAt.IsZero() && time.Now().After(u.ExpiresAt) {
			continue
		}
		allowed := make([]string, 0, len(u.AllowedRoutes))
		for _, rid := range u.AllowedRoutes {
			if routeEnabled[rid] {
				allowed = append(allowed, rid)
			}
		}
		if len(allowed) == 0 {
			continue
		}
		exportedUsers = append(exportedUsers, UserExport{
			UUID:          u.UUID,
			AllowedRoutes: allowed,
			Username:      u.Username,
			Type:          or(u.Type, "relay"),
			Peers:         nonZero(u.Peers, 1),
		})
	}

	store := turnableStore{Routes: exportedRoutes, Users: exportedUsers}

	if err := writeJSON(filepath.Join(st.Settings.WorkDir, "config.json"), cfg, 0600); err != nil {
		return err
	}
	if err := writeJSON(filepath.Join(st.Settings.WorkDir, "store.json"), store, 0600); err != nil {
		return err
	}
	st.LastApplyAt = time.Now().UTC()
	return nil
}

func writeJSON(path string, v any, mode os.FileMode) error {
	raw, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	raw = append(raw, '\n')
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, raw, mode); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

func GenerateClientConfig(st State, user User, routeIDs []string) (string, error) {
	if len(routeIDs) == 0 {
		routeIDs = user.AllowedRoutes
	}
	configPath := filepath.Join(st.Settings.WorkDir, "config.json")
	if st.Settings.TurnableBinary != "" {
		args := append([]string{"config", "generate", user.UUID}, routeIDs...)
		args = append(args, "-c", configPath)
		cmd := exec.Command(st.Settings.TurnableBinary, args...)
		cmd.Dir = st.Settings.WorkDir
		out, err := cmd.CombinedOutput()
		if err == nil && strings.TrimSpace(string(out)) != "" {
			return strings.TrimSpace(string(out)), nil
		}
	}
	if st.Settings.PublicKey == "" {
		return "", fmt.Errorf("public key is empty; run: turnable config keygen and save keys in settings")
	}
	if st.Settings.CallID == "" {
		return "", fmt.Errorf("call id is empty")
	}
	rid := ""
	if len(routeIDs) > 0 {
		rid = routeIDs[0]
	}
	values := url.Values{}
	values.Set("pub_key", st.Settings.PublicKey)
	values.Set("type", or(user.Type, "relay"))
	values.Set("peers", fmt.Sprint(nonZero(user.Peers, 1)))
	values.Set("username", user.Username)
	return fmt.Sprintf("turnable://%s:%s@%s/%s?%s", url.PathEscape(user.UUID), url.PathEscape(st.Settings.CallID), st.Settings.PlatformID, url.PathEscape(rid), values.Encode()), nil
}

func RunSystemctl(action, service string) (string, error) {
	action = strings.TrimSpace(action)
	service = strings.TrimSpace(service)
	allowed := map[string]bool{"start": true, "stop": true, "restart": true, "status": true, "is-active": true}
	if !allowed[action] {
		return "", fmt.Errorf("unsupported systemctl action: %s", action)
	}
	if service == "" {
		return "", fmt.Errorf("empty service name")
	}
	cmd := exec.Command("systemctl", action, service)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	err := cmd.Run()
	return strings.TrimSpace(out.String()), err
}

func or(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func nonZero(v, fallback int) int {
	if v == 0 {
		return fallback
	}
	return v
}
