package serve

import (
	"context"
	"io"
	"net"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/chinmay28/homeapi/internal/discovery"
)

func env(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

// The defaults are what HomeAPI always did: every interface, port 9999.
// Discovery is the one addition, on its own port.
func TestDefaultsAreTheExistingBehaviour(t *testing.T) {
	c, err := Resolve(Flags{}, env(nil))
	if err != nil {
		t.Fatal(err)
	}
	if c.Addr() != ":9999" || c.Loopback() {
		t.Errorf("addr %q loopback %v, want :9999 on every interface", c.Addr(), c.Loopback())
	}
	if c.Discovery() != discovery.DefaultPort {
		t.Errorf("discovery = %d, want %d", c.Discovery(), discovery.DefaultPort)
	}
	if c.Name == "" {
		t.Error("no name to announce")
	}
}

func TestFlagBeatsEnvironmentBeatsDefault(t *testing.T) {
	e := env(map[string]string{EnvHost: "0.0.0.0", EnvPort: "8000", EnvDiscoveryPort: "7000", EnvName: "Pixel"})
	c, err := Resolve(Flags{}, e)
	if err != nil {
		t.Fatal(err)
	}
	if c.Addr() != "0.0.0.0:8000" || c.Discovery() != 7000 || c.Name != "Pixel" {
		t.Errorf("from env: %+v", c)
	}
	c, err = Resolve(Flags{Host: "127.0.0.1", Port: "9100", DiscoveryPort: "0"}, e)
	if err != nil {
		t.Fatal(err)
	}
	if c.Addr() != "127.0.0.1:9100" || c.DiscoveryPort != 0 {
		t.Errorf("from flags: %+v", c)
	}
}

func TestLoopbackNeverAnswersDiscovery(t *testing.T) {
	for _, host := range []string{"127.0.0.1", "::1", "[::1]", "localhost"} {
		c, err := Resolve(Flags{Host: host}, env(nil))
		if err != nil {
			t.Fatalf("%s: %v", host, err)
		}
		if !c.Loopback() || c.Discovery() != 0 {
			t.Errorf("%s: loopback %v discovery %d", host, c.Loopback(), c.Discovery())
		}
	}
	if c, _ := Resolve(Flags{Host: "::1"}, env(nil)); c.Addr() != "[::1]:9999" {
		t.Errorf("IPv6 addr = %q", c.Addr())
	}
}

func TestBadValuesAreRefused(t *testing.T) {
	for _, f := range []Flags{
		{Port: "0"}, {Port: "70000"}, {Port: "http"},
		{DiscoveryPort: "-1"}, {DiscoveryPort: "x"},
		{Host: "example.com"}, {Host: "127.0.0.1:9999"},
	} {
		if _, err := Resolve(f, env(nil)); err == nil {
			t.Errorf("Resolve(%+v) accepted it", f)
		}
	}
}

// Run serves until its context ends, lets a request that is already running
// finish, and then returns nil.
func TestRunDrainsInFlightRequestsOnStop(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	started := make(chan struct{})
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(started)
		time.Sleep(200 * time.Millisecond)
		io.WriteString(w, "done")
	})
	ctx, cancel := context.WithCancel(context.Background())
	c := Config{Host: "127.0.0.1", Port: 0}
	done := make(chan error, 1)
	go func() { done <- Run(ctx, c, ln, h, func() string { return "v0" }, t.Logf) }()

	body := make(chan string, 1)
	go func() {
		resp, err := http.Get("http://" + ln.Addr().String() + "/")
		if err != nil {
			body <- "error: " + err.Error()
			return
		}
		b, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		body <- string(b)
	}()
	<-started
	cancel()
	if got := <-body; got != "done" {
		t.Errorf("in-flight request got %q, want it to finish", got)
	}
	select {
	case err := <-done:
		if err != nil {
			t.Errorf("Run = %v, want nil after a clean stop", err)
		}
	case <-time.After(ShutdownGrace + time.Second):
		t.Fatal("Run did not return")
	}
	if _, err := http.Get("http://" + ln.Addr().String() + "/"); err == nil {
		t.Error("still serving after stop")
	}
}

// With discovery on, a probe finds the web server's real port.
func TestRunAnswersDiscoveryWithTheWebPort(t *testing.T) {
	udp, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	dport := udp.LocalAddr().(*net.UDPAddr).Port
	udp.Close()

	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	// Host "" = every interface, so discovery runs; the listener is loopback
	// only because that is all a test should open.
	c := Config{Port: 9999, DiscoveryPort: dport, Name: "phone"}
	go Run(ctx, c, ln, http.NotFoundHandler(), func() string { return "v2026.11.1" }, t.Logf)

	client, err := net.ListenPacket("udp4", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	target := &net.UDPAddr{IP: net.IPv4(127, 0, 0, 1), Port: dport}
	var found []discovery.Found
	for i := 0; i < 20 && len(found) == 0; i++ { // the responder starts asynchronously
		pctx, pcancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
		found, err = discovery.Probe(pctx, client, target)
		pcancel()
		if err != nil && !strings.Contains(err.Error(), "refused") {
			t.Fatal(err)
		}
	}
	if len(found) != 1 {
		t.Fatalf("found %+v", found)
	}
	want := ln.Addr().(*net.TCPAddr).Port
	if f := found[0]; f.Port != want || f.Name != "phone" || f.Version != "v2026.11.1" {
		t.Errorf("found %+v, want port %d", f, want)
	}
}
