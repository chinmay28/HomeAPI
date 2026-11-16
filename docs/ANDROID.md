# HomeAPI on Android

The HomeAPI app puts HomeAPI on an Android phone. It works one of two ways, and
you can switch between them at any time in the app's settings:

| Mode | What runs where | Use it when |
|---|---|---|
| **Connect to a server** (thin client) | HomeAPI runs on another machine — a Raspberry Pi, a desktop, a server. The phone opens it, as a browser would. | You already run HomeAPI at home and want it on your phone. |
| **Run HomeAPI on this phone** (full stack) | The whole server runs on the phone, with its database in the app's private storage. | You want a HomeAPI with no other machine — or a phone that scripts and displays on your Wi-Fi can read and write. |

Either way the app shows HomeAPI's own web GUI, so everything the browser does,
the app does: entries, categories, search, bulk delete, the dashboard, the
Minion 3 preview, export and import. This page covers what is different on a
phone. How it is built is in [`apps/android/README.md`](../apps/android/README.md).

## Install the app

Every [release](https://github.com/chinmay28/homeapi/releases) attaches it as
`homeapi.apk`, and the latest is always at:

```
https://github.com/chinmay28/homeapi/releases/latest/download/homeapi.apk
```

Open that address on the phone, download the APK, and open it. Android asks once
to let your browser install apps; allow it. A newer release installs over the
old one, keeping your settings and, in full-stack mode, your data.

It needs Android 10 or newer. Full-stack mode needs a 64-bit ARM phone (nearly
every phone sold since 2017); on any other, the app is a thin client only and
says so.

## The first screen

The app asks where HomeAPI is.

- **Connect to a server.** Type the server's address — `192.168.1.20`,
  `pi.local`, or `https://home.example.com` — or tap **Find on this network**,
  which lists every HomeAPI server that answers on your Wi-Fi. An address with no
  `http://` or `https://` means `http://` on HomeAPI's own port, 9999.
  Discovery needs a server from this release or later (older ones don't answer),
  not started with `--discovery-port 0`, with UDP port 9998 open.
- **Run HomeAPI on this phone.** The server starts on the phone and the app
  opens its GUI, empty and ready for entries.

The app's version is in small text under **Continue**. The round badge at the
bottom is the developer's mark; tap it to see it full screen.

## Settings

The app's settings are behind **App settings** on HomeAPI's Settings page (shown
only inside the app), and behind a long press on the app's icon, which works
whatever the server offers.

| Setting | Mode | What it does |
|---|---|---|
| Run HomeAPI on this phone | both | Full stack when on, thin client when off. If no server address is set yet, the app asks for one as you switch. Each mode keeps its own settings. |
| Server address, Find servers | thin client | Which server the app opens. |
| Serve other devices on this network | full stack | Off: only this phone can reach the API. On: the server listens on every interface, so `curl http://<phone's address>:9999/api/entries` works from a laptop, a script or a Minion display on the same Wi-Fi, and other HomeAPI apps can find the phone. HomeAPI has no password and plain HTTP is not encrypted — turn this on only on a network you trust, as with any HomeAPI server. |
| Keep running when the app is closed | full stack | Off: HomeAPI stops when you leave the app. On: it keeps serving other devices and the widget; a notification says it is running and has a **Stop** button. |
| Name on the network | full stack | What other HomeAPI apps list this phone as. The phone's own name unless you change it. |
| Port | full stack | 9999 unless another app on the phone has it. |
| Server log, Restart, Stop | full stack | What the server has said lately, and control over it. |

Switching to thin client stops the server on the phone but never deletes its
database; switching back opens it again.

## Export and import

- **Export** (HomeAPI's Settings page) saves `homeapi-export-<date>.json` to
  *Downloads/HomeAPI*, with a message when it lands. From there, any file
  manager or the Files app can share it.
- **Import** opens Android's file picker, which reaches the phone's storage and
  any app that offers files (Drive, Dropbox, a USB stick).

Moving data between the phone and a server is an export on one and an import on
the other; the format is the same everywhere.

## The home-screen widget

Long-press the home screen, choose **Widgets**, and drag **HomeAPI** out. It asks
for a key — `tickers`, say — and then shows that entry's value from whichever
server the app is set to, exactly as stored. Tap it to read it again; Android
also refreshes it every half hour, and in full-stack mode it refreshes the
moment the server on the phone comes up. On launchers that offer it, long-press
the widget to change its key. A key made only of digits is an entry id to the
API (`/api/entries/12`); the widget's title shows which entry it found.

In full-stack mode the widget can only read while the server runs: turn on
**Keep running when the app is closed** if you want it current with the app
closed.

## Full stack: what to know

- **Where the data lives.** `homeapi.db` in the app's private storage, which no
  other app can read and Android's backups leave out. **Uninstalling the app
  deletes it** — export first.
- **The time zone** is the phone's (the app passes it in `HOMEAPI_TZ`; Go on
  Android would otherwise run in UTC), so a Minion 3 display served by the phone
  shows local time.
- **When the phone sleeps**, Android may slow the server down even with *Keep
  running* on. A phone on a charger serves best.

## Thin client: what to know

- **HTTPS with a private certificate.** If your server's certificate is from
  your own CA, install that CA in Android (*Settings › Security › Encryption &
  credentials › Install a certificate › CA certificate*); the app trusts it. A
  certificate nothing on the phone trusts is refused, not clicked through.
- **Plain HTTP** is allowed, as for HomeAPI on a home network.
- **Can't reach the server?** The app says so, with **Retry** and **App
  settings**. Check the phone is on the same network, or on your VPN.
- Links that leave the server open in the phone's browser.

## Troubleshooting

| Symptom | What to do |
|---|---|
| *HomeAPI on this phone would not start*, with a log | Another app may hold the port: change **Port** in settings. Otherwise share the log (**Server log › Share**) with a bug report. |
| **Find on this network** finds nothing | The server is older than this app, was started with `--discovery-port 0`, a firewall blocks UDP 9998, or the phone is on another network (guest Wi-Fi, mobile data). Type the address instead. |
| Other devices can't reach the phone | Turn on **Serve other devices**, and use the address in the notification. Some Wi-Fi networks (guest, "client isolation") block devices from reaching each other. |
| The widget says it cannot reach the server | In full-stack mode, open the app (or turn on *Keep running*); in thin-client mode, check the phone's network. Then tap the widget. |
| **App settings** is missing from the Settings page | The server is older than the app. Long-press the app's icon and pick **Settings**. |
