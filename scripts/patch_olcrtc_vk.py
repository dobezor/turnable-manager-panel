#!/usr/bin/env python3
import argparse
from pathlib import Path


def replace_once(path: Path, old: str, new: str) -> None:
    text = path.read_text(encoding="utf-8")
    if new in text:
        return
    if old not in text:
        raise RuntimeError(f"pattern not found in {path}: {old[:120]!r}")
    path.write_text(text.replace(old, new, 1), encoding="utf-8")


def patch_panel(panel: Path) -> None:
    main_go = panel / "cmd" / "olcrtc-manager" / "main.go"
    main_tsx = panel / "src" / "main.tsx"

    replace_once(
        main_go,
        '"jitsi": {\n\t\t\t"datachannel":  true,\n\t\t\t"vp8channel":   true,\n\t\t\t"seichannel":   true,\n\t\t\t"videochannel": true,\n\t\t},',
        '"jitsi": {\n\t\t\t"datachannel":  true,\n\t\t\t"vp8channel":   true,\n\t\t\t"seichannel":   true,\n\t\t\t"videochannel": true,\n\t\t},\n\t\t"vk": {\n\t\t\t"datachannel":  false,\n\t\t\t"vp8channel":   true,\n\t\t\t"seichannel":   true,\n\t\t\t"videochannel": true,\n\t\t},',
    )

    replace_once(
        main_tsx,
        'const carriers = ["jitsi", "wbstream", "telemost", "jazz"];',
        'const carriers = ["jitsi", "wbstream", "telemost", "jazz", "vk"];',
    )

    replace_once(
        main_tsx,
        '  jazz: ["datachannel"],\n};',
        '  jazz: ["datachannel"],\n  vk: ["vp8channel", "seichannel", "videochannel"],\n};',
    )

    replace_once(
        main_tsx,
        '  return carrier === "jitsi" ? "https://meet.example.org/room" : "room-id";\n}',
        '  if (carrier === "jitsi") return "https://meet.example.org/room";\n  if (carrier === "vk") return "https://vk.ru/call/join/<call-id>";\n  return "room-id";\n}',
    )


def patch_olcrtc(olcrtc: Path) -> None:
    vk_dir = olcrtc / "internal" / "auth" / "vk"
    vk_dir.mkdir(parents=True, exist_ok=True)

    (vk_dir / "vk.go").write_text(r'''package vk

import (
	"context"
	"fmt"
	"strings"

	"github.com/openlibrecommunity/olcrtc/internal/auth"
)

const roomURLPrefix = "https://vk.ru/call/join/"

// Provider produces LiveKit-like credentials for VK Calls when the VK page
// exposes a media WebSocket URL and token in its boot payload. The shape mirrors
// the WB Stream provider: normalize room -> fetch connection details -> return
// URL/token to the LiveKit engine.
type Provider struct{}

func (Provider) Engine() string { return "livekit" }

func (Provider) DefaultServiceURL() string { return "https://vk.ru" }

func (Provider) Issue(ctx context.Context, cfg auth.Config) (auth.Credentials, error) {
	if cfg.RoomURL == "" || cfg.RoomURL == "any" {
		return auth.Credentials{}, auth.ErrRoomIDRequired
	}

	roomURL := normalizeRoomURL(cfg.RoomURL)
	info, err := GetConnectionInfo(ctx, roomURL, cfg.Name)
	if err != nil {
		return auth.Credentials{}, fmt.Errorf("get vk call connection info: %w", err)
	}

	roomID := info.RoomID
	if roomID == "" {
		roomID = extractCallID(roomURL)
	}

	return auth.Credentials{
		URL:   info.ServerURL,
		Token: info.RoomToken,
		Extra: map[string]string{"roomID": roomID, "roomURL": roomURL},
	}, nil
}

func normalizeRoomURL(value string) string {
	value = strings.TrimSpace(value)
	if strings.HasPrefix(value, "https://") || strings.HasPrefix(value, "http://") {
		return value
	}
	return roomURLPrefix + strings.TrimLeft(value, "/")
}

func init() {
	auth.Register("vk", Provider{})
}
''', encoding="utf-8")

    (vk_dir / "api.go").write_text(r'''package vk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/openlibrecommunity/olcrtc/internal/protect"
)

var errVKConnectionInfo = errors.New("vk connection info not found")

type connectionInfo struct {
	RoomID    string
	RoomToken string
	ServerURL string
}

func GetConnectionInfo(ctx context.Context, roomURL, displayName string) (connectionInfo, error) {
	page, finalURL, err := fetchCallPage(ctx, roomURL, displayName)
	if err != nil {
		return connectionInfo{}, err
	}

	info := parseConnectionInfo(page)
	if info.RoomID == "" {
		info.RoomID = extractCallID(finalURL)
	}
	if info.ServerURL == "" || info.RoomToken == "" {
		return connectionInfo{}, fmt.Errorf("%w: page did not contain server url/token; capture Network HAR for %s", errVKConnectionInfo, finalURL)
	}
	return info, nil
}

func fetchCallPage(ctx context.Context, roomURL, displayName string) (string, string, error) {
	u, err := url.Parse(roomURL)
	if err != nil {
		return "", "", fmt.Errorf("parse room url: %w", err)
	}
	q := u.Query()
	if displayName != "" && q.Get("display_name") == "" {
		q.Set("display_name", displayName)
	}
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return "", "", fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 Chrome/125 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "ru-RU,ru;q=0.9,en;q=0.8")

	client := protect.NewHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return "", "", fmt.Errorf("do request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("vk call page status: %w", protect.StatusError(errVKConnectionInfo, resp, 8192))
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 8*1024*1024))
	if err != nil {
		return "", "", fmt.Errorf("read response: %w", err)
	}
	return string(body), resp.Request.URL.String(), nil
}

func parseConnectionInfo(page string) connectionInfo {
	page = htmlUnescape(page)
	return connectionInfo{
		RoomID:    firstJSONLikeValue(page, "roomId", "room_id", "callId", "call_id", "conferenceId", "conference_id"),
		RoomToken: firstJSONLikeValue(page, "roomToken", "room_token", "token", "accessToken", "access_token", "jwt"),
		ServerURL: firstWSSValue(page, "serverUrl", "server_url", "mediaServerUrl", "media_server_url", "wsUrl", "ws_url", "websocketUrl", "websocket_url", "url"),
	}
}

func htmlUnescape(s string) string {
	replacer := strings.NewReplacer("&quot;", "\"", "&#34;", "\"", "&amp;", "&", "\\/", "/")
	return replacer.Replace(s)
}

func firstJSONLikeValue(body string, keys ...string) string {
	for _, key := range keys {
		patterns := []string{
			fmt.Sprintf(`"%s"\s*:\s*"([^"\\]*(?:\\.[^"\\]*)*)"`, regexp.QuoteMeta(key)),
			fmt.Sprintf(`%s\s*:\s*"([^"\\]*(?:\\.[^"\\]*)*)"`, regexp.QuoteMeta(key)),
		}
		for _, pattern := range patterns {
			re := regexp.MustCompile(pattern)
			m := re.FindStringSubmatch(body)
			if len(m) == 2 {
				return unquoteJSONString(m[1])
			}
		}
	}
	return ""
}

func firstWSSValue(body string, keys ...string) string {
	for _, key := range keys {
		value := firstJSONLikeValue(body, key)
		if strings.HasPrefix(value, "wss://") || strings.HasPrefix(value, "ws://") || strings.HasPrefix(value, "https://") {
			return value
		}
	}
	re := regexp.MustCompile(`wss://[^"'<>\\\s]+`)
	if m := re.FindString(body); m != "" {
		return strings.ReplaceAll(m, `\/`, `/`)
	}
	return ""
}

func unquoteJSONString(s string) string {
	var out string
	if err := json.Unmarshal([]byte(`"`+s+`"`), &out); err == nil {
		return out
	}
	return strings.ReplaceAll(s, `\/`, `/`)
}

func extractCallID(roomURL string) string {
	u, err := url.Parse(roomURL)
	if err != nil {
		return strings.TrimSpace(roomURL)
	}
	parts := strings.Split(strings.Trim(u.Path, "/"), "/")
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "join" && parts[i+1] != "" {
			return parts[i+1]
		}
	}
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return roomURL
}
''', encoding="utf-8")

    builtin = olcrtc / "internal" / "engine" / "builtin" / "builtin.go"
    replace_once(
        builtin,
        'authTelemost "github.com/openlibrecommunity/olcrtc/internal/auth/telemost"\n\tauthWBStream "github.com/openlibrecommunity/olcrtc/internal/auth/wbstream"',
        'authTelemost "github.com/openlibrecommunity/olcrtc/internal/auth/telemost"\n\tauthVK "github.com/openlibrecommunity/olcrtc/internal/auth/vk"\n\tauthWBStream "github.com/openlibrecommunity/olcrtc/internal/auth/wbstream"',
    )
    replace_once(
        builtin,
        'registerEngineAuth("jitsi", authJitsi.Provider{})\n\tregisterDirect("none")',
        'registerEngineAuth("jitsi", authJitsi.Provider{})\n\tregisterEngineAuth("vk", authVK.Provider{})\n\tregisterDirect("none")',
    )

    session_go = olcrtc / "internal" / "app" / "session" / "session.go"
    text = session_go.read_text(encoding="utf-8")
    text = text.replace(
        'auth provider required (set auth.provider to jitsi, telemost, wbstream or none)',
        'auth provider required (set auth.provider to jitsi, telemost, wbstream, vk or none)',
    )
    session_go.write_text(text, encoding="utf-8")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--panel", required=True, help="path to olcrtc-manager-panel checkout")
    parser.add_argument("--olcrtc", required=True, help="path to olcrtc checkout")
    args = parser.parse_args()

    panel = Path(args.panel).resolve()
    olcrtc = Path(args.olcrtc).resolve()
    patch_panel(panel)
    patch_olcrtc(olcrtc)
    print("patched panel and olcrtc for carrier vk")


if __name__ == "__main__":
    main()
