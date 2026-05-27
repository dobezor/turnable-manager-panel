package app

const layoutTpl = `{{define "layout"}}
<!doctype html>
<html lang="ru">
<head>
  <meta charset="utf-8">
  <meta name="viewport" content="width=device-width, initial-scale=1">
  <title>{{.Title}} · Turnable Manager</title>
  <link rel="stylesheet" href="/static/app.css">
</head>
<body>
{{if ne .Title "Login"}}
<header class="topbar">
  <div class="brand">Turnable Manager Panel</div>
  <nav>
    <a class="{{active .Active "dashboard"}}" href="/admin">Dashboard</a>
    <a class="{{active .Active "users"}}" href="/admin/users">Users</a>
    <a class="{{active .Active "routes"}}" href="/admin/routes">Routes</a>
    <a class="{{active .Active "settings"}}" href="/admin/settings">Settings</a>
    <a href="/admin/logout">Logout</a>
  </nav>
</header>
{{end}}
<main class="page">
  {{if .Message}}<div class="notice ok">{{.Message}}</div>{{end}}
  {{if .Error}}<div class="notice err">{{.Error}}</div>{{end}}
  {{template "content" .}}
</main>
</body>
</html>
{{end}}
`

const loginTpl = `{{define "content"}}
<section class="login-card">
  <h1>Admin login</h1>
  <form method="post" action="/admin/login">
    <label>Username<input name="username" value="admin" autocomplete="username"></label>
    <label>Password<input name="password" type="password" autocomplete="current-password" autofocus></label>
    <button type="submit">Login</button>
  </form>
</section>
{{end}}
`

const dashboardTpl = `{{define "content"}}
<h1>Dashboard</h1>
<div class="grid cards">
  <section class="card"><span class="muted">Users</span><strong>{{len .State.Users}}</strong></section>
  <section class="card"><span class="muted">Routes</span><strong>{{len .State.Routes}}</strong></section>
  <section class="card"><span class="muted">Turnable service</span><strong>{{.ServiceOutput}}</strong></section>
  <section class="card"><span class="muted">Last apply</span><strong>{{formatTime .State.LastApplyAt}}</strong></section>
</div>

<section class="card">
  <h2>Apply generated files</h2>
  <p class="muted">Writes <code>config.json</code> and <code>store.json</code> into <code>{{.State.Settings.WorkDir}}</code>. Restart Turnable after applying.</p>
  <form method="post" action="/admin/apply" class="inline">
    <button type="submit">Apply config/store</button>
  </form>
</section>

<section class="card">
  <h2>Service control</h2>
  <div class="row-actions">
    <form method="post" action="/admin/service"><input type="hidden" name="target" value="turnable"><button name="action" value="restart">Restart Turnable</button></form>
    <form method="post" action="/admin/service"><input type="hidden" name="target" value="turnable"><button name="action" value="status">Status Turnable</button></form>
    <form method="post" action="/admin/service"><input type="hidden" name="target" value="xray"><button name="action" value="restart">Restart Xray</button></form>
    <form method="post" action="/admin/service"><input type="hidden" name="target" value="xray"><button name="action" value="status">Status Xray</button></form>
  </div>
  {{if .ServiceOutput}}<pre>{{.ServiceOutput}}</pre>{{end}}
</section>

<section class="card">
  <h2>RU → NL relay idea</h2>
  <p>Client connects to Turnable through TURN/SFU. RU server receives the tunnel. A route can forward traffic to a local Xray inbound, and Xray sends it to the NL VLESS outbound.</p>
  <pre>Phone → Turnable carrier → RU Turnable → 127.0.0.1:10808 Xray → NL VLESS → Internet</pre>
</section>
{{end}}
`

const settingsTpl = `{{define "content"}}
<h1>Settings</h1>
<form method="post" action="/admin/settings" class="card form-grid">
  <h2>Turnable server</h2>
  <label>Platform ID<input name="platform_id" value="{{.State.Settings.PlatformID}}" placeholder="vk.com"></label>
  <label>Call ID<input name="call_id" value="{{.State.Settings.CallID}}" placeholder="meeting/call id"></label>
  <label>Private key<textarea name="priv_key" rows="3">{{.State.Settings.PrivateKey}}</textarea></label>
  <label>Public key<textarea name="pub_key" rows="3">{{.State.Settings.PublicKey}}</textarea></label>

  <h2>Relay mode</h2>
  <label class="check"><input type="checkbox" name="relay_enabled" {{checked .State.Settings.RelayEnabled}}> Relay enabled</label>
  <label>Relay proto<select name="relay_proto"><option>dtls</option><option>srtp</option><option>none</option></select><small>Current: {{.State.Settings.RelayProto}}</small></label>
  <label>Relay cloak<input name="relay_cloak" value="{{.State.Settings.RelayCloak}}" placeholder="none"></label>
  <label>Public IP<input name="relay_public_ip" value="{{.State.Settings.RelayPublicIP}}" placeholder="RU server public IP"></label>
  <label>Relay UDP port<input name="relay_port" type="number" value="{{.State.Settings.RelayPort}}"></label>

  <h2>P2P mode</h2>
  <label class="check"><input type="checkbox" name="p2p_enabled" {{checked .State.Settings.P2PEnabled}}> P2P enabled</label>
  <label>P2P username<input name="p2p_username" value="{{.State.Settings.P2PUsername}}"></label>
  <label>P2P cloak<input name="p2p_cloak" value="{{.State.Settings.P2PCloak}}" placeholder="none"></label>

  <h2>Paths and services</h2>
  <label>Work dir<input name="work_dir" value="{{.State.Settings.WorkDir}}"></label>
  <label>Turnable binary<input name="turnable_binary" value="{{.State.Settings.TurnableBinary}}"></label>
  <label>Turnable systemd service<input name="turnable_service" value="{{.State.Settings.TurnableService}}"></label>
  <label>Xray systemd service<input name="xray_service" value="{{.State.Settings.XrayService}}"></label>
  <button type="submit">Save settings</button>
</form>

<section class="card">
  <h2>Key generation</h2>
  <p class="muted">Run on the RU server, then paste keys above:</p>
  <pre>{{.State.Settings.TurnableBinary}} config keygen</pre>
</section>
{{end}}
`

const routesTpl = `{{define "content"}}
<h1>Routes</h1>
<section class="card">
  <h2>Add or update route</h2>
  <form method="post" action="/admin/routes" class="form-grid compact">
    <label>ID<input name="id" placeholder="xray-tcp"></label>
    <label>Name<input name="name" placeholder="Local Xray client TCP"></label>
    <label>Address<input name="address" value="127.0.0.1"></label>
    <label>Port<input name="port" type="number" value="10808"></label>
    <label>Socket<select name="socket"><option>tcp</option><option>udp</option></select></label>
    <label>Transport<select name="transport"><option>kcp</option><option>none</option><option>sctp</option></select></label>
    <label>Encryption<select name="encryption"><option>handshake</option><option>full</option></select></label>
    <label>Connection override<select name="conn"><option value="">user default</option><option>relay</option><option>p2p</option></select></label>
    <label class="check"><input type="checkbox" name="enabled" checked> Enabled</label>
    <label class="wide">Notes<textarea name="notes" rows="2"></textarea></label>
    <button type="submit">Save route</button>
  </form>
</section>

<section class="card">
  <h2>Existing routes</h2>
  <table>
    <thead><tr><th>ID</th><th>Name</th><th>Destination</th><th>Socket</th><th>Transport</th><th>Enabled</th><th></th></tr></thead>
    <tbody>{{range .State.Routes}}
      <tr>
        <td><code>{{.ID}}</code></td><td>{{.Name}}</td><td>{{.Address}}:{{.Port}}</td><td>{{.Socket}}</td><td>{{.Transport}}</td><td>{{.Enabled}}</td>
        <td><form method="post" action="/admin/routes/delete" onsubmit="return confirm('Delete route?')"><input type="hidden" name="id" value="{{.ID}}"><button class="danger">Delete</button></form></td>
      </tr>
    {{else}}<tr><td colspan="7">No routes</td></tr>{{end}}</tbody>
  </table>
</section>
{{end}}
`

const usersTpl = `{{define "content"}}
<h1>Users</h1>
<section class="card">
  <h2>Add or update user</h2>
  <form method="post" action="/admin/users" class="form-grid compact">
    <label>UUID<input name="uuid" placeholder="leave empty to generate"></label>
    <label>Username<input name="username" placeholder="client01"></label>
    <label>Type<select name="type"><option>relay</option><option>p2p</option></select></label>
    <label>Peers<input name="peers" type="number" value="4"></label>
    <label class="wide">Allowed routes
      <div class="checks">{{range .State.Routes}}<label><input type="checkbox" name="allowed_routes" value="{{.ID}}" checked> {{.ID}} — {{.Name}}</label>{{end}}</div>
    </label>
    <label class="check"><input type="checkbox" name="disabled"> Disabled</label>
    <label class="wide">Notes<textarea name="notes" rows="2"></textarea></label>
    <button type="submit">Save user</button>
  </form>
</section>

<section class="card">
  <h2>Existing users</h2>
  <table>
    <thead><tr><th>UUID</th><th>Username</th><th>Type</th><th>Peers</th><th>Routes</th><th>Disabled</th><th>Config</th><th></th></tr></thead>
    <tbody>{{range .State.Users}}
      <tr>
        <td><code>{{.UUID}}</code></td><td>{{.Username}}</td><td>{{.Type}}</td><td>{{.Peers}}</td><td>{{join .AllowedRoutes ", "}}</td><td>{{.Disabled}}</td>
        <td><a class="button" href="/admin/export?uuid={{.UUID}}">Export</a></td>
        <td><form method="post" action="/admin/users/delete" onsubmit="return confirm('Delete user?')"><input type="hidden" name="uuid" value="{{.UUID}}"><button class="danger">Delete</button></form></td>
      </tr>
    {{else}}<tr><td colspan="8">No users</td></tr>{{end}}</tbody>
  </table>
</section>
{{end}}
`

const appCSS = `
:root{--bg:#0d1117;--panel:#161b22;--panel2:#1f2630;--text:#e6edf3;--muted:#8b949e;--line:#30363d;--accent:#58a6ff;--danger:#f85149;--ok:#3fb950}
*{box-sizing:border-box}body{margin:0;background:var(--bg);color:var(--text);font:14px/1.5 system-ui,-apple-system,Segoe UI,Roboto,Arial,sans-serif}.topbar{height:62px;border-bottom:1px solid var(--line);display:flex;align-items:center;justify-content:space-between;padding:0 28px;background:#010409}.brand{font-weight:700;letter-spacing:.2px}.topbar nav{display:flex;gap:10px}.topbar a{color:var(--muted);text-decoration:none;padding:8px 10px;border-radius:8px}.topbar a.active,.topbar a:hover{background:var(--panel2);color:var(--text)}.page{max-width:1180px;margin:0 auto;padding:28px}h1{font-size:28px;margin:0 0 18px}h2{font-size:18px;margin:0 0 14px}.grid{display:grid;grid-template-columns:repeat(4,minmax(0,1fr));gap:14px}.card{background:var(--panel);border:1px solid var(--line);border-radius:14px;padding:18px;margin-bottom:18px}.cards .card{display:flex;flex-direction:column;gap:8px}.card strong{font-size:24px}.muted,small{color:var(--muted)}code,pre{font-family:ui-monospace,SFMono-Regular,Consolas,monospace}pre{background:#0b0f14;border:1px solid var(--line);border-radius:10px;padding:14px;overflow:auto}table{width:100%;border-collapse:collapse}th,td{border-bottom:1px solid var(--line);text-align:left;padding:10px;vertical-align:top}th{color:var(--muted);font-weight:600}.form-grid{display:grid;grid-template-columns:repeat(2,minmax(0,1fr));gap:14px}.form-grid.compact{grid-template-columns:repeat(3,minmax(0,1fr))}.form-grid h2,.wide{grid-column:1/-1}label{display:flex;flex-direction:column;gap:6px;color:var(--muted)}label.check{flex-direction:row;align-items:center;color:var(--text)}input,select,textarea{width:100%;background:#0b0f14;color:var(--text);border:1px solid var(--line);border-radius:10px;padding:10px}button,.button{display:inline-block;background:var(--accent);color:#06101f;border:0;border-radius:10px;padding:10px 13px;font-weight:700;text-decoration:none;cursor:pointer}button.danger{background:var(--danger);color:#fff}.row-actions{display:flex;flex-wrap:wrap;gap:10px}.inline{display:inline}.checks{display:flex;flex-direction:column;gap:6px;background:#0b0f14;border:1px solid var(--line);border-radius:10px;padding:10px}.checks label{flex-direction:row;color:var(--text)}.notice{padding:12px 14px;border-radius:12px;margin-bottom:16px}.notice.ok{background:rgba(63,185,80,.12);border:1px solid rgba(63,185,80,.35)}.notice.err{background:rgba(248,81,73,.12);border:1px solid rgba(248,81,73,.35)}.login-card{max-width:380px;margin:12vh auto;background:var(--panel);border:1px solid var(--line);border-radius:16px;padding:24px}.login-card form{display:flex;flex-direction:column;gap:14px}@media(max-width:900px){.grid,.form-grid,.form-grid.compact{grid-template-columns:1fr}.topbar{height:auto;align-items:flex-start;flex-direction:column;padding:14px}.topbar nav{flex-wrap:wrap}.page{padding:18px}table{display:block;overflow-x:auto}}
`
