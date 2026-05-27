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
    vk_go = vk_dir / "vk.go"
    vk_go.write_text(
        '''package vk

import (
    "context"
    "fmt"
    "strings"

    "github.com/openlibrecommunity/olcrtc/internal/auth"
)

const roomURLPrefix = "https://vk.ru/call/join/"

// Provider is a compile-ready placeholder for VK Calls support.
//
// The manager panel can now emit auth.provider: vk, but a real implementation
// still has to resolve a vk.ru/call/join room into SFU engine credentials.
// Keep this provider explicit instead of silently returning fake credentials.
type Provider struct{}

func (Provider) Engine() string { return "goolom" }

func (Provider) DefaultServiceURL() string { return "https://vk.ru" }

func (Provider) Issue(ctx context.Context, cfg auth.Config) (auth.Credentials, error) {
    if cfg.RoomURL == "" {
        return auth.Credentials{}, auth.ErrRoomIDRequired
    }
    roomURL := cfg.RoomURL
    if !strings.HasPrefix(roomURL, "https://") {
        roomURL = roomURLPrefix + strings.TrimLeft(roomURL, "/")
    }
    _ = ctx
    return auth.Credentials{}, fmt.Errorf("vk calls provider is registered, but signaling for %s is not implemented yet", roomURL)
}

func init() {
    auth.Register("vk", Provider{})
}
''',
        encoding="utf-8",
    )

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
