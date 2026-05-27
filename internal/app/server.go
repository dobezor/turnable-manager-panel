package app

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type App struct {
	cfg   Config
	store *Store
}

type pageData struct {
	Title         string
	Active        string
	State         State
	Message       string
	Error         string
	ServiceOutput string
	Exported      string
	QueryUUID     string
}

func New(cfg Config) (*App, error) {
	store, generatedPassword, err := OpenStore(cfg.StateFile)
	if err != nil {
		return nil, err
	}
	if generatedPassword != "" {
		log.Printf("FIRST RUN ADMIN PASSWORD: %s", generatedPassword)
		log.Printf("set TURNABLE_MANAGER_ADMIN_PASSWORD before first start to avoid auto-generated password")
	}
	return &App{cfg: cfg, store: store}, nil
}

func (a *App) Routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/static/app.css", a.css)
	mux.HandleFunc("/admin/login", a.login)
	mux.HandleFunc("/admin/logout", a.logout)
	mux.HandleFunc("/admin/export", a.auth(a.exportConfig))
	mux.HandleFunc("/admin/apply", a.auth(a.apply))
	mux.HandleFunc("/admin/service", a.auth(a.service))
	mux.HandleFunc("/admin/settings", a.auth(a.settings))
	mux.HandleFunc("/admin/routes/delete", a.auth(a.deleteRoute))
	mux.HandleFunc("/admin/routes", a.auth(a.routes))
	mux.HandleFunc("/admin/users/delete", a.auth(a.deleteUser))
	mux.HandleFunc("/admin/users", a.auth(a.users))
	mux.HandleFunc("/admin", a.auth(a.dashboard))
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/admin", http.StatusFound)
	})
	return secureHeaders(mux)
}

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func (a *App) css(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/css; charset=utf-8")
	_, _ = w.Write([]byte(appCSS))
}

func (a *App) auth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !a.validSession(r) {
			http.Redirect(w, r, "/admin/login", http.StatusFound)
			return
		}
		next(w, r)
	}
}

func (a *App) validSession(r *http.Request) bool {
	c, err := r.Cookie("tmp_session")
	if err != nil || c.Value == "" {
		return false
	}
	parts := strings.Split(c.Value, ".")
	if len(parts) != 3 {
		return false
	}
	if parts[0] != "admin" {
		return false
	}
	exp, err := strconv.ParseInt(parts[1], 10, 64)
	if err != nil || time.Now().Unix() > exp {
		return false
	}
	return VerifySigned(parts[0]+"."+parts[1], parts[2], a.cfg.SessionSecret) == nil
}

func (a *App) setSession(w http.ResponseWriter) {
	exp := time.Now().Add(12 * time.Hour).Unix()
	value := fmt.Sprintf("admin.%d", exp)
	sig := Sign(value, a.cfg.SessionSecret)
	http.SetCookie(w, &http.Cookie{
		Name:     "tmp_session",
		Value:    value + "." + sig,
		Path:     "/",
		HttpOnly: true,
		Secure:   a.cfg.CookieSecure,
		SameSite: http.SameSiteStrictMode,
		Expires:  time.Unix(exp, 0),
	})
}

func (a *App) login(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		a.render(w, "login", pageData{Title: "Login"}, loginTpl)
		return
	}
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	username := r.FormValue("username")
	password := r.FormValue("password")
	st := a.store.Snapshot()
	if username == st.AdminUsername && VerifyPassword(st.AdminPassword, password) {
		a.setSession(w)
		http.Redirect(w, r, "/admin", http.StatusFound)
		return
	}
	a.render(w, "login", pageData{Title: "Login", Error: "Bad username or password"}, loginTpl)
}

func (a *App) logout(w http.ResponseWriter, r *http.Request) {
	http.SetCookie(w, &http.Cookie{Name: "tmp_session", Value: "", Path: "/", MaxAge: -1, HttpOnly: true})
	http.Redirect(w, r, "/admin/login", http.StatusFound)
}

func (a *App) dashboard(w http.ResponseWriter, r *http.Request) {
	st := a.store.Snapshot()
	out := ""
	if a.cfg.AllowServiceControl && st.Settings.TurnableService != "" {
		if s, err := RunSystemctl("is-active", st.Settings.TurnableService); err == nil {
			out = s
		} else {
			out = "inactive/unknown"
		}
	}
	a.render(w, "dashboard", pageData{Title: "Dashboard", Active: "dashboard", State: st, ServiceOutput: out}, dashboardTpl)
}

func (a *App) settings(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		err := a.store.Mutate(func(st *State) error {
			st.Settings.PlatformID = strings.TrimSpace(r.FormValue("platform_id"))
			st.Settings.CallID = strings.TrimSpace(r.FormValue("call_id"))
			st.Settings.PrivateKey = strings.TrimSpace(r.FormValue("priv_key"))
			st.Settings.PublicKey = strings.TrimSpace(r.FormValue("pub_key"))
			st.Settings.RelayEnabled = r.FormValue("relay_enabled") == "on"
			st.Settings.RelayProto = strings.TrimSpace(r.FormValue("relay_proto"))
			st.Settings.RelayCloak = strings.TrimSpace(r.FormValue("relay_cloak"))
			st.Settings.RelayPublicIP = strings.TrimSpace(r.FormValue("relay_public_ip"))
			st.Settings.RelayPort = atoiDefault(r.FormValue("relay_port"), 56000)
			st.Settings.P2PEnabled = r.FormValue("p2p_enabled") == "on"
			st.Settings.P2PUsername = strings.TrimSpace(r.FormValue("p2p_username"))
			st.Settings.P2PCloak = strings.TrimSpace(r.FormValue("p2p_cloak"))
			st.Settings.WorkDir = strings.TrimSpace(r.FormValue("work_dir"))
			st.Settings.TurnableBinary = strings.TrimSpace(r.FormValue("turnable_binary"))
			st.Settings.TurnableService = strings.TrimSpace(r.FormValue("turnable_service"))
			st.Settings.XrayService = strings.TrimSpace(r.FormValue("xray_service"))
			return nil
		})
		if err != nil {
			a.render(w, "settings", pageData{Title: "Settings", Active: "settings", State: a.store.Snapshot(), Error: err.Error()}, settingsTpl)
			return
		}
		http.Redirect(w, r, "/admin/settings?ok=1", http.StatusFound)
		return
	}
	msg := ""
	if r.URL.Query().Get("ok") == "1" {
		msg = "Settings saved"
	}
	a.render(w, "settings", pageData{Title: "Settings", Active: "settings", State: a.store.Snapshot(), Message: msg}, settingsTpl)
}

func (a *App) routes(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		route := Route{
			ID:         slug(r.FormValue("id")),
			Name:       strings.TrimSpace(r.FormValue("name")),
			Address:    strings.TrimSpace(r.FormValue("address")),
			Port:       atoiDefault(r.FormValue("port"), 10808),
			Socket:     or(strings.TrimSpace(r.FormValue("socket")), "tcp"),
			Transport:  or(strings.TrimSpace(r.FormValue("transport")), "kcp"),
			Encryption: or(strings.TrimSpace(r.FormValue("encryption")), "handshake"),
			Conn:       strings.TrimSpace(r.FormValue("conn")),
			Enabled:    r.FormValue("enabled") == "on",
			Notes:      strings.TrimSpace(r.FormValue("notes")),
		}
		if route.ID == "" || route.Name == "" || route.Address == "" || route.Port <= 0 {
			a.render(w, "routes", pageData{Title: "Routes", Active: "routes", State: a.store.Snapshot(), Error: "Route ID, name, address and port are required"}, routesTpl)
			return
		}
		err := a.store.Mutate(func(st *State) error {
			for i := range st.Routes {
				if st.Routes[i].ID == route.ID {
					st.Routes[i] = route
					return nil
				}
			}
			st.Routes = append(st.Routes, route)
			return nil
		})
		if err != nil {
			a.render(w, "routes", pageData{Title: "Routes", Active: "routes", State: a.store.Snapshot(), Error: err.Error()}, routesTpl)
			return
		}
		http.Redirect(w, r, "/admin/routes?ok=1", http.StatusFound)
		return
	}
	msg := ""
	if r.URL.Query().Get("ok") == "1" {
		msg = "Route saved"
	}
	a.render(w, "routes", pageData{Title: "Routes", Active: "routes", State: a.store.Snapshot(), Message: msg}, routesTpl)
}

func (a *App) deleteRoute(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	id := r.FormValue("id")
	_ = a.store.Mutate(func(st *State) error {
		filtered := st.Routes[:0]
		for _, rt := range st.Routes {
			if rt.ID != id {
				filtered = append(filtered, rt)
			}
		}
		st.Routes = filtered
		for i := range st.Users {
			st.Users[i].AllowedRoutes = removeString(st.Users[i].AllowedRoutes, id)
		}
		return nil
	})
	http.Redirect(w, r, "/admin/routes", http.StatusFound)
}

func (a *App) users(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodPost {
		u := User{
			UUID:          strings.TrimSpace(r.FormValue("uuid")),
			Username:      strings.TrimSpace(r.FormValue("username")),
			Type:          or(strings.TrimSpace(r.FormValue("type")), "relay"),
			Peers:         atoiDefault(r.FormValue("peers"), 1),
			AllowedRoutes: r.Form["allowed_routes"],
			Disabled:      r.FormValue("disabled") == "on",
			Notes:         strings.TrimSpace(r.FormValue("notes")),
			CreatedAt:     time.Now().UTC(),
		}
		if u.UUID == "" {
			u.UUID = newUUID()
		}
		if u.Username == "" {
			u.Username = "user-" + u.UUID[:8]
		}
		if len(u.AllowedRoutes) == 0 {
			a.render(w, "users", pageData{Title: "Users", Active: "users", State: a.store.Snapshot(), Error: "Select at least one route"}, usersTpl)
			return
		}
		err := a.store.Mutate(func(st *State) error {
			for i := range st.Users {
				if st.Users[i].UUID == u.UUID {
					created := st.Users[i].CreatedAt
					st.Users[i] = u
					st.Users[i].CreatedAt = created
					return nil
				}
			}
			st.Users = append(st.Users, u)
			return nil
		})
		if err != nil {
			a.render(w, "users", pageData{Title: "Users", Active: "users", State: a.store.Snapshot(), Error: err.Error()}, usersTpl)
			return
		}
		http.Redirect(w, r, "/admin/users?ok=1", http.StatusFound)
		return
	}
	msg := ""
	if r.URL.Query().Get("ok") == "1" {
		msg = "User saved"
	}
	a.render(w, "users", pageData{Title: "Users", Active: "users", State: a.store.Snapshot(), Message: msg}, usersTpl)
}

func (a *App) deleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	uuid := r.FormValue("uuid")
	_ = a.store.Mutate(func(st *State) error {
		filtered := st.Users[:0]
		for _, u := range st.Users {
			if u.UUID != uuid {
				filtered = append(filtered, u)
			}
		}
		st.Users = filtered
		return nil
	})
	http.Redirect(w, r, "/admin/users", http.StatusFound)
}

func (a *App) apply(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	err := a.store.Mutate(func(st *State) error { return ApplyTurnableFiles(st) })
	if err != nil {
		a.render(w, "dashboard", pageData{Title: "Dashboard", Active: "dashboard", State: a.store.Snapshot(), Error: err.Error()}, dashboardTpl)
		return
	}
	http.Redirect(w, r, "/admin?applied=1", http.StatusFound)
}

func (a *App) exportConfig(w http.ResponseWriter, r *http.Request) {
	uuid := r.URL.Query().Get("uuid")
	st := a.store.Snapshot()
	var user User
	found := false
	for _, u := range st.Users {
		if u.UUID == uuid {
			user = u
			found = true
			break
		}
	}
	if !found {
		http.Error(w, "user not found", http.StatusNotFound)
		return
	}
	routeIDs := r.URL.Query()["route"]
	result, err := GenerateClientConfig(st, user, routeIDs)
	if err != nil {
		a.render(w, "users", pageData{Title: "Users", Active: "users", State: st, Error: err.Error()}, usersTpl)
		return
	}
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	_, _ = w.Write([]byte(result + "\n"))
}

func (a *App) service(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if !a.cfg.AllowServiceControl {
		http.Error(w, "service control disabled", http.StatusForbidden)
		return
	}
	st := a.store.Snapshot()
	target := r.FormValue("target")
	action := r.FormValue("action")
	service := st.Settings.TurnableService
	if target == "xray" {
		service = st.Settings.XrayService
	}
	out, err := RunSystemctl(action, service)
	if err != nil {
		out = out + "\n" + err.Error()
	}
	a.render(w, "dashboard", pageData{Title: "Dashboard", Active: "dashboard", State: a.store.Snapshot(), ServiceOutput: out}, dashboardTpl)
}

func (a *App) render(w http.ResponseWriter, name string, data pageData, content string) {
	if data.State.Version == 0 && name != "login" {
		data.State = a.store.Snapshot()
	}
	funcs := template.FuncMap{
		"join": strings.Join,
		"checked": func(v bool) string {
			if v {
				return "checked"
			}
			return ""
		},
		"active": func(current, item string) string {
			if current == item {
				return "active"
			}
			return ""
		},
		"hasRoute": func(routes []string, id string) bool {
			for _, r := range routes {
				if r == id {
					return true
				}
			}
			return false
		},
		"formatTime": func(t time.Time) string {
			if t.IsZero() {
				return "-"
			}
			return t.Local().Format("2006-01-02 15:04:05")
		},
	}
	tpl := template.Must(template.New("layout").Funcs(funcs).Parse(layoutTpl + content))
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tpl.ExecuteTemplate(w, "layout", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func atoiDefault(v string, fallback int) int {
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return fallback
	}
	return n
}

func slug(v string) string {
	v = strings.ToLower(strings.TrimSpace(v))
	replacer := strings.NewReplacer(" ", "-", "_", "-", ".", "-", "/", "-", "\\", "-")
	v = replacer.Replace(v)
	var b strings.Builder
	for _, r := range v {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			b.WriteRune(r)
		}
	}
	return strings.Trim(b.String(), "-")
}

func removeString(values []string, target string) []string {
	out := values[:0]
	for _, v := range values {
		if v != target {
			out = append(out, v)
		}
	}
	return out
}

func newUUID() string {
	h := RandomHex(16)
	return fmt.Sprintf("%s-%s-%s-%s-%s", h[0:8], h[8:12], h[12:16], h[16:20], h[20:32])
}
