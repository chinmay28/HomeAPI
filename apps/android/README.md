# HomeAPI for Android — how it is built

The app (`apps/android`) is HomeAPI's own web GUI in a web view, in front of one
of two servers: a server on another machine (thin client), or the real
`homeapi` binary running on the phone (full stack). It is modelled file for file
on SAND Vault for Android. Using it is in the [guide](../../docs/ANDROID.md).

```
 ┌──────────────────────── the phone ──────────────────────────────┐
 │  MainActivity                                                    │
 │   WebView ── HomeAPI's web GUI, from Origin only                 │
 │     ▲  homeapi-bridge.js (document start, Origin only)           │
 │     │  HomeAPIAndroid ⇄ PageBridge: save, share, settings        │
 │     │                                                            │
 │     │ thin client            full stack                          │
 │     │     │               http://127.0.0.1:9999                  │
 │     │     │                     ▲                                │
 │     │     │  ServerService ── ServerSupervisor ── libhomeapi.so   │
 │     │     │  (foreground)       restart, log    --host 127.0.0.1 │
 │     │     │                                     HOMEAPI_TZ, _DB_PATH
 │  EntryWidgetProvider ── GET /api/entries/<key> ──┘ (either server)│
 └─────┼─────┼──────────────────────────────────────────────────────┘
       │     ▼
       │  http(s)://server:9999   ◄── UDP 9998 HOMEAPI-DISCOVER/1 (Find on this network)
```

## One interface, two servers

The web GUI already does everything, so the app does not reimplement it; it
shows the server's page and adds only what a web view lacks. The two modes
differ in one thing, the origin the web view opens: the configured server, or
`http://127.0.0.1:<port>`. The bridge, downloads, navigation rules and the
widget are the same code keyed on that origin.

## The server on the phone

- **The binary.** Gradle's `buildHomeapiServer-arm64-v8a` (before `preBuild`)
  runs `go build` with `CGO_ENABLED=0 GOOS=linux GOARCH=arm64 -tags timetzdata`,
  stamped with the version, into `lib/arm64-v8a/libhomeapi.so`. Android executes
  files only from an app's native library directory, and only when they are
  installed as files (`useLegacyPackaging`), hence the name. A static *linux*
  binary rather than `GOOS=android`: HomeAPI's SQLite is the pure-Go
  `modernc.org/sqlite`, whose C-to-Go libc targets linux, and a static linux
  binary runs on Android as it stands (CountRoster's MOBILE.md §2.1 has the
  same finding). The binary embeds the web GUI, so the build first runs
  `make frontend` when `cmd/homeapi/frontend_build` has no real build, and
  fails if there still is none. `-PappServer=false` builds a thin client
  without Go or Node; `-PappServerAbis=arm64-v8a,x86_64` adds an emulator build.
- **What Android lacks.** Go on Android starts in UTC, so the app passes the
  phone's zone in `HOMEAPI_TZ`, which `internal/hostenv` installs as
  `time.Local` (the Minion display draws local time); the zone database is
  linked in. HomeAPI makes no outbound connections, so unlike SAND Vault there
  is no DNS shim. `HOME`, `TMPDIR` and `HOMEAPI_DB_PATH` point into the app's
  private files; `HOMEAPI_NAME` is the name discovery announces (every Android
  app sees the hostname `localhost`). All of these are documented server
  settings, inert when unset. `core/ServerLaunch.kt` is the whole command line
  and environment, tested on its own.
- **Keeping it running.** `ServerService` is a foreground service
  (`specialUse`) owning one `core/ServerSupervisor`: it starts the process,
  waits for `/api/health`, restarts it with backoff if it exits, leaves it down
  after five exits in two minutes (with the log's tail), and stops it with
  SIGTERM — which `homeapi` answers by draining requests and closing the
  database — before SIGKILL. A server an earlier run left behind is found in
  `/proc` by its program path and ended first.
- **Binding.** `--host 127.0.0.1 --discovery-port 0` by default: only this
  phone's apps can reach it. *Serve other devices* passes `--host 0.0.0.0
  --discovery-port 9998`, which is what lets scripts on the LAN use the phone's
  API and other HomeAPI apps find it.

## Discovery

`internal/discovery` (Go, tested) answers a UDP broadcast of
`HOMEAPI-DISCOVER/1` on port 9998 with `{"service":"homeapi","name","port",
"version"}`, only to private, loopback, link-local and CGNAT senders, never on a
loopback bind, and off with `--discovery-port 0`. The app's `LanDiscovery`
broadcasts the query and `core/DiscoveryProtocol` reads the answers; its test
reads the three constants out of the Go source so the two cannot drift.

## The page bridge

`assets/homeapi-bridge.js` runs at document start on the configured origin only
(`addDocumentStartJavaScript` and `addWebMessageListener` with that origin as
the only rule) and talks to the app through `HomeAPIAndroid`:

| Page does | Bridge sends | App does |
|---|---|---|
| Settings → Export: `<a download href="blob:…">` | `begin`, `chunk`…, `end` (`save`) | writes it to *Downloads/HomeAPI* through MediaStore |
| `<a download href="/…">` on the origin | `fetch` | downloads it (redirects only within the origin) |
| `navigator.share({ files })` | `stage` each file, then `share` | the share sheet, *Save to Downloads* first |
| App settings button | `settings` | opens `SettingsActivity` |

It defines `window.homeapiApp`, which is how the web GUI knows to show its
**App settings** card (`frontend/src/nativeApp.js`, tested in
`Settings.test.js`); a browser has no such object and sees no change.
`core/Transfers.kt` holds the page to what it announced, so a half file never
lands in Downloads. Import is the web view's `<input type=file>`, answered by
`onShowFileChooser` with Android's picker (`core/AcceptTypes` maps `.json`).

## Native where it pays

- **Discovery** instead of typing an address (above).
- **Downloads and the share sheet** for export (above).
- **A home-screen widget** (`EntryWidgetProvider`): one entry's value by key,
  read with `GET /api/entries/<key>` from the app's current origin — a
  watchlist or a value a script keeps current, glanceable without opening
  anything. Refreshed on tap, by Android every 30 minutes, and when the server
  on the phone comes up; no WorkManager, no extra dependency. What it asks and
  how it reads the answer (`value_text` first, as the web GUI does) is
  `core/EntryWidget`, tested.
- **The developer badge** on the setup screen and in settings, with the web
  GUI's full-screen flash (`core/DevFlash` is the timeline, matching
  `frontend/src/App.css`'s keyframes).

Not built, on purpose: a native entry editor (the web GUI is the interface), a
share target that imports a JSON file sent from another app (it would need a
page-side hook for little gain over the picker), and full stack on 32-bit ARM.

## Navigation

`core/NavigationPolicy.kt` decides every navigation and `window.open`: the
origin's own pages and `blob:`/`data:` stay in the app; other `http(s)` goes to
the phone's browser; `intent:` is launched only after dropping any component or
selector and limiting it to browsable apps; `javascript:`, `file:`,
`content:` and the like are refused.

## Icons

Every launcher icon is derived from `frontend/public/icon.svg`, the mark the web
GUI already shows, by `cmd/homeapi-icons` (`make icons`): the 48dp
`mipmap-*/ic_launcher.png`, the adaptive icon's 108dp foreground with the
artwork in the 72dp safe zone over `ic_launcher_background` (the SVG's plate
colour), and `drawable-nodpi/homeapi_logo.png`. It reads the SVG's small subset
(rounded rects, round-capped lines and polylines) and renders it by signed
distance, supersampled; anything outside that subset is an error, not a guess.
`go test ./cmd/homeapi-icons` (part of `make test`) fails when a committed copy
has drifted, or a `@drawable`/`@mipmap` reference names nothing. Two drawables
are hand-drawn, not generated: `ic_stat_homeapi.xml`, the notification
silhouette (the roof, and the card with its rows cut out), and
`ic_launcher_monochrome.xml`, the same silhouette as Android 13's themed icon.
If the artwork changes shape, redraw them. The developer badge PNGs are copies
of `frontend/public/dev-badge*.png`.

## Code layout and checks

```
apps/android/
├── core/   plain Kotlin, JVM-tested, no Android:
│           Origin, NavigationPolicy, AppConfig, ModeSwitch, ServerLaunch
│           (+ MachineName), ServerSupervisor (+ RestartPolicy, StaleServers),
│           Health, Bridge, Transfers, FileNames, AcceptTypes,
│           DiscoveryProtocol, EntryWidget, DevFlash
└── app/    the Android shell: MainActivity, PageBridge + assets/homeapi-bridge.js,
            ServerService + LocalServer, Downloads, Sharing,
            SaveToDownloadsActivity, SetupActivity, SettingsActivity,
            LanDiscovery, EntryWidgetProvider + EntryWidgetConfigActivity,
            DevBadge, ConfigStore
```

```bash
cd apps/android
./gradlew --no-daemon -PcoreOnly :core:test          # no Android SDK needed
./gradlew --no-daemon :core:test :app:lintDebug :app:assembleRelease
```

Version: `-PappVersionName`, `-PappVersionCode` (the commit count) and
`-PappVersionDate`, from `scripts/version.mjs`, as the release workflow passes
them; without them the app is `dev`. The release build is signed with the debug
key so it sideloads as it stands. CI is `.github/workflows/android.yml`; the
release attaches the APK as `homeapi.apk`.

## Signing

Releases are signed with one fixed key, read from the `ANDROID_KEYSTORE_BASE64`
and `ANDROID_KEYSTORE_PASSWORD` repository secrets (a PKCS12 keystore, alias
`harbor`, shared by every Harbor app), so each release installs over the last.
A build without them, any local build included, falls back to the debug key.
