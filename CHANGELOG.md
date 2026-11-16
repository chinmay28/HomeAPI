# Changelog

Releases are `vYEAR.MONTH.PATCH` — a calendar version, where YEAR.MONTH is the
month of the release's commit and the patch number is the repository's commit
count, so `v2026.11.42` is the 42nd commit, made in November 2026. See
[`internal/version/version.go`](./internal/version/version.go).

Each section below is the body of the corresponding GitHub release, and is what
publishes it: a push to `main` whose commit builds a version named by a
`## vX.Y.Z` heading here is released, any other push is not, and a tag (or a
manual run) whose version has no section fails. A heading must name the tag
exactly. Work not yet released goes under `## Unreleased`, which never matches a
version; rename it to the version when cutting the release.

## v2026.11.25

**Downloads:** **`homeapi.apk`** (new: HomeAPI for Android — on the phone, open
`https://github.com/chinmay28/homeapi/releases/latest/download/homeapi.apk`
and install it; see [`docs/ANDROID.md`](./docs/ANDROID.md)),
`homeapi-linux-arm64` and `homeapi-linux-amd64` (the server: static, with the
web GUI built in). All are in `SHA256SUMS`.

### HomeAPI for Android

HomeAPI now has a phone app, which works one of two ways:

- **Connect to a server** — the app opens your HomeAPI, wherever it runs.
  **Find on this network** lists the servers on your Wi-Fi, so there is no
  address to type.
- **Run HomeAPI on this phone** — the whole server runs on the phone, with its
  database in the app's private storage. Turn on **Serve other devices on this
  network** and scripts, browsers and Minion displays on your Wi-Fi can use the
  phone's API at `http://<phone>:9999`.

It is HomeAPI's own web GUI, so everything works as in a browser. Export saves to
*Downloads/HomeAPI*, import opens Android's file picker, and links that leave
the server open in your browser. A **home-screen widget** shows one entry's value
by key — a watchlist, a note a script keeps current — and reads it again when you
tap it. The app follows the phone's light or dark theme, and its first screen
carries its version and the developer badge.

### The server

- **Find on this network.** A server answers the app's discovery broadcast (UDP
  port 9998) with its port, name and version — only on a private network, never
  when bound to loopback. Turn it off with `--discovery-port 0`.
- **`--host`** (or `HOMEAPI_HOST`) binds one address, e.g. `127.0.0.1`. The
  default is still every interface.
- **A clean stop.** On SIGTERM or Ctrl-C the server finishes the requests it is
  serving and closes the database before exiting.
- **No C compiler needed.** The SQLite driver is now pure Go, so the binary is
  built static (`CGO_ENABLED=0`) for any platform. Existing databases are read
  and written exactly as before.
- **Versions take their month from the commit**, so there is nothing to bump;
  `make version` prints what a tree builds as.
