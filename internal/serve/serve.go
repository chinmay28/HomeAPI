// Package serve is how the homeapi binary turns its flags and environment into
// a listening server: where to bind, whether to answer discovery, and how to
// stop.
//
// Every setting is resolved flag > environment variable > default, and every
// default is what HomeAPI has always done — listen on every interface on port
// 9999 — so an existing deployment's unit file means exactly what it meant.
// The additions are opt-in or inert:
//
//   - --host / HOMEAPI_HOST binds one address (the Android app passes
//     127.0.0.1 unless the phone is serving other devices);
//   - --discovery-port / HOMEAPI_DISCOVERY_PORT is the UDP port
//     internal/discovery answers on (0 turns it off; a loopback bind never
//     starts it, since nobody it could answer could reach the server);
//   - HOMEAPI_NAME is the name discovery announces (default: the hostname);
//   - SIGTERM / SIGINT stop accepting, let in-flight requests finish, and only
//     then close the database, so its WAL is checkpointed.
package serve

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/chinmay28/homeapi/internal/discovery"
)

// Environment variable names, the fallbacks for the flags.
const (
	EnvHost          = "HOMEAPI_HOST"
	EnvPort          = "HOMEAPI_PORT"
	EnvDiscoveryPort = "HOMEAPI_DISCOVERY_PORT"
	EnvName          = "HOMEAPI_NAME"
)

// DefaultPort is the web server's port when nothing names one.
const DefaultPort = 9999

// ShutdownGrace is how long a stopping server waits for in-flight requests.
const ShutdownGrace = 5 * time.Second

// Flags are the command-line values as given; "" means "not given".
type Flags struct {
	Host          string
	Port          string
	DiscoveryPort string
}

// Config is the resolved configuration.
type Config struct {
	Host          string // "" = every interface
	Port          int
	DiscoveryPort int // 0 = off
	Name          string
}

// Resolve applies flag > environment > default to every setting.
func Resolve(f Flags, getenv func(string) string) (Config, error) {
	pick := func(flag, env string) string {
		if v := strings.TrimSpace(flag); v != "" {
			return v
		}
		return strings.TrimSpace(getenv(env))
	}
	c := Config{Host: pick(f.Host, EnvHost), Port: DefaultPort, DiscoveryPort: discovery.DefaultPort}

	if v := pick(f.Port, EnvPort); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 1 || p > 65535 {
			return Config{}, fmt.Errorf("port %q: want a number from 1 to 65535", v)
		}
		c.Port = p
	}
	if v := pick(f.DiscoveryPort, EnvDiscoveryPort); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil || p < 0 || p > 65535 {
			return Config{}, fmt.Errorf("discovery port %q: want 0 (off) or a number up to 65535", v)
		}
		c.DiscoveryPort = p
	}
	if c.Host != "" {
		if _, err := netip.ParseAddr(strings.Trim(c.Host, "[]")); err != nil && c.Host != "localhost" {
			return Config{}, fmt.Errorf("host %q: want an IP address to bind, such as 127.0.0.1 or 0.0.0.0", c.Host)
		}
	}
	c.Name = strings.TrimSpace(getenv(EnvName))
	if c.Name == "" {
		if h, err := os.Hostname(); err == nil {
			c.Name = h
		}
	}
	if c.Name == "" {
		c.Name = "HomeAPI"
	}
	return c, nil
}

// Addr is the TCP address to listen on.
func (c Config) Addr() string {
	return net.JoinHostPort(strings.Trim(c.Host, "[]"), strconv.Itoa(c.Port))
}

// Loopback reports whether the server is bound where only this machine can
// reach it.
func (c Config) Loopback() bool {
	if c.Host == "localhost" {
		return true
	}
	ip, err := netip.ParseAddr(strings.Trim(c.Host, "[]"))
	return err == nil && ip.IsLoopback()
}

// Discovery is the UDP port to answer discovery on, or 0 when it is off —
// asked to be, or bound where no other machine could reach the web server.
func (c Config) Discovery() int {
	if c.DiscoveryPort <= 0 || c.Loopback() {
		return 0
	}
	return c.DiscoveryPort
}

// Run serves handler on ln until ctx is done, answering discovery alongside
// when the config asks for it, then shuts down gracefully: no new
// connections, up to ShutdownGrace for requests already running. It returns
// nil after a clean stop and the server's error otherwise.
func Run(ctx context.Context, c Config, ln net.Listener, handler http.Handler, version func() string, logf func(string, ...any)) error {
	srv := &http.Server{Handler: handler, ReadHeaderTimeout: 30 * time.Second}

	if port := c.Discovery(); port > 0 {
		dctx, stop := context.WithCancel(ctx)
		defer stop()
		webPort := c.Port
		if tcp, ok := ln.Addr().(*net.TCPAddr); ok {
			webPort = tcp.Port
		}
		announce := func() discovery.Announcement {
			return discovery.Announcement{Name: c.Name, Port: webPort, Version: version()}
		}
		addr := net.JoinHostPort("", strconv.Itoa(port))
		go func() {
			logf("answering discovery on udp %s (--discovery-port 0 turns it off)", addr)
			if err := discovery.Listen(dctx, addr, announce); err != nil && dctx.Err() == nil {
				logf("discovery is off: %v — clients can still connect by address", err)
			}
		}()
	}

	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()

	select {
	case err := <-errc:
		return err
	case <-ctx.Done():
	}
	sctx, cancel := context.WithTimeout(context.Background(), ShutdownGrace)
	defer cancel()
	err := srv.Shutdown(sctx)
	if serr := <-errc; serr != nil && !errors.Is(serr, http.ErrServerClosed) && err == nil {
		err = serr
	}
	return err
}
