package api

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/chinmay28/homeapi/internal/display"
	"github.com/chinmay28/homeapi/internal/models"
)

// DisplayName is the one display this server renders. The URL carries the
// name so a second panel is additive later.
const DisplayName = "minion3"

// PNG preview scale bounds and default.
const (
	minPNGScale     = 1
	maxPNGScale     = 8
	defaultPNGScale = 3
)

// Display routes /api/displays/{name}/{frame.bin|frame.png}. Anything else
// under the prefix is a 404 in the usual JSON error shape.
func (h *Handler) Display(w http.ResponseWriter, r *http.Request) {
	rest := strings.TrimPrefix(r.URL.Path, "/api/displays/")
	name, file, ok := strings.Cut(rest, "/")
	if !ok || name != DisplayName {
		writeError(w, http.StatusNotFound, "Display not found", "NOT_FOUND")
		return
	}
	switch file {
	case "frame.bin":
		h.serveFrame(w, r, false)
	case "frame.png":
		h.serveFrame(w, r, true)
	default:
		writeError(w, http.StatusNotFound, "API endpoint not found", "NOT_FOUND")
	}
}

// frameInputs is everything serveFrame pulled from the store for one render.
type frameInputs struct {
	cfg        display.Config
	in         display.Inputs
	quotesText string // raw minion-quotes value, "" when absent
	prevText   string // raw minion3-prev-quotes value, "" when absent
}

// loadFrameInputs assembles the render inputs from the store. Missing
// entries degrade the picture; only a failing store is an error.
func (h *Handler) loadFrameInputs(now time.Time) (*frameInputs, error) {
	fi := &frameInputs{}

	params, err := h.store.GetEntryByKey(display.ParamsKey)
	if err != nil {
		return nil, fmt.Errorf("loading %s: %w", display.ParamsKey, err)
	}
	if params == nil {
		fi.cfg = display.DefaultConfig()
		fi.in.ConfigInvalid = true
	} else {
		fi.cfg, err = display.ParseConfig([]byte(params.Value))
		if err != nil {
			log.Printf("display: %v (rendering the default)", err)
			fi.in.ConfigInvalid = true
		}
	}

	loc, err := time.LoadLocation(fi.cfg.Clock.TZ)
	if err != nil {
		loc = time.UTC
	}

	quotes, err := h.store.GetEntryByKey(display.QuotesKey)
	if err != nil {
		return nil, fmt.Errorf("loading %s: %w", display.QuotesKey, err)
	}
	if quotes != nil {
		fi.quotesText = quotes.Value
	}
	fi.in.Quotes = display.ParseQuotes(fi.quotesText, now, loc)

	prev, err := h.store.GetEntryByKey(display.PrevQuotesKey)
	if err != nil {
		return nil, fmt.Errorf("loading %s: %w", display.PrevQuotesKey, err)
	}
	if prev != nil {
		fi.prevText = prev.Value
	}
	fi.in.Prev = display.ParseQuotes(fi.prevText, now, loc)

	if fi.cfg.Sum != nil && fi.cfg.Sum.Key != "" {
		sum, err := h.store.GetEntryByKey(fi.cfg.Sum.Key)
		if err != nil {
			return nil, fmt.Errorf("loading %s: %w", fi.cfg.Sum.Key, err)
		}
		if sum != nil {
			fi.in.Sum = sum.Value
		}
	}
	return fi, nil
}

// serveFrame renders the frame and writes it as the native body or a PNG.
//
// Query parameters, all optional: at=<RFC3339> renders as if it were that
// instant and makes the request read-only; wake=<n> is the device's wake
// counter; scale=<1–8> sizes the PNG. Side effects — the previous-quotes
// snapshot and the last-seen record — happen only for a frame.bin request
// without `at`, i.e. a real device wake, so previews never disturb what the
// panel's change pills compare against.
func (h *Handler) serveFrame(w http.ResponseWriter, r *http.Request, asPNG bool) {
	q := r.URL.Query()

	now := time.Now()
	preview := false
	if at := q.Get("at"); at != "" {
		t, err := time.Parse(time.RFC3339, at)
		if err != nil {
			writeError(w, http.StatusBadRequest, "at must be an RFC 3339 timestamp", "VALIDATION_ERROR")
			return
		}
		now, preview = t, true
	}

	var wake int
	hasWake := false
	if ws := q.Get("wake"); ws != "" {
		n, err := strconv.Atoi(ws)
		if err != nil || n < 0 {
			writeError(w, http.StatusBadRequest, "wake must be a non-negative integer", "VALIDATION_ERROR")
			return
		}
		wake, hasWake = n, true
	}

	scale := defaultPNGScale
	if s := q.Get("scale"); s != "" && asPNG {
		n, err := strconv.Atoi(s)
		if err != nil || n < minPNGScale || n > maxPNGScale {
			writeError(w, http.StatusBadRequest, fmt.Sprintf("scale must be between %d and %d", minPNGScale, maxPNGScale), "VALIDATION_ERROR")
			return
		}
		scale = n
	}

	fi, err := h.loadFrameInputs(now)
	if err != nil {
		log.Printf("display: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to load display data", "INTERNAL_ERROR")
		return
	}
	fi.in.Wake, fi.in.HasWake = wake, hasWake

	frame, err := display.Render(fi.cfg, fi.in, now)
	if err != nil {
		log.Printf("display: render: %v", err)
		writeError(w, http.StatusInternalServerError, "Failed to render display", "INTERNAL_ERROR")
		return
	}

	next, sleep, err := display.NextWake(fi.cfg.Schedule, now)
	if err != nil {
		// The config validated, so this is unreachable in practice; still,
		// the device must get a sleep time.
		log.Printf("display: schedule: %v", err)
		sleep = display.DefaultSleepSeconds
		next = now.Add(time.Duration(sleep) * time.Second)
	}

	body := frame.Native()
	sum := sha256.Sum256(body)
	tag := hex.EncodeToString(sum[:])[:16]
	etag := `"` + tag + `"`
	if asPNG {
		etag = fmt.Sprintf(`"%s-s%d"`, tag, scale)
	}
	stale, _ := display.Staleness(fi.cfg, fi.in.Quotes, now)

	hdr := w.Header()
	hdr.Set("Cache-Control", "no-cache")
	hdr.Set("ETag", etag)
	hdr.Set("X-Minion-Format", "1")
	hdr.Set("X-Minion-Sleep-Seconds", strconv.Itoa(sleep))
	hdr.Set("X-Minion-Next-Wake", next.Format(time.RFC3339))
	hdr.Set("X-Minion-Rendered-At", now.In(next.Location()).Format(time.RFC3339))
	if stale {
		hdr.Set("X-Minion-Stale", "1")
	}

	if etagMatches(r.Header.Get("If-None-Match"), etag) {
		w.WriteHeader(http.StatusNotModified)
	} else if asPNG {
		data, err := frame.PNG(scale)
		if err != nil {
			log.Printf("display: png: %v", err)
			writeError(w, http.StatusInternalServerError, "Failed to encode display", "INTERNAL_ERROR")
			return
		}
		hdr.Set("Content-Type", "image/png")
		hdr.Set("Content-Length", strconv.Itoa(len(data)))
		w.WriteHeader(http.StatusOK)
		w.Write(data)
	} else {
		hdr.Set("Content-Type", "application/octet-stream")
		hdr.Set("Content-Length", strconv.Itoa(len(body)))
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	}

	if asPNG || preview {
		return
	}
	// Best-effort bookkeeping for a real device wake, after the response is
	// on its way. A 304 still counts: the device woke and the quotes it will
	// compare against next time are the ones it was offered now.
	h.recordDeviceWake(fi, now, wake, hasWake)
}

// recordDeviceWake replaces the previous-quotes snapshot when the quotes
// have moved on, and stores the device's wake counter. Errors are logged,
// never surfaced: the frame has already been served.
func (h *Handler) recordDeviceWake(fi *frameInputs, now time.Time, wake int, hasWake bool) {
	if fi.quotesText != "" && snapshotKey(fi.in.Quotes, fi.quotesText) != snapshotKey(fi.in.Prev, fi.prevText) {
		if err := h.upsertEntry(display.Category, display.PrevQuotesKey, fi.quotesText); err != nil {
			log.Printf("display: updating %s: %v", display.PrevQuotesKey, err)
		}
	}
	if hasWake {
		seen, _ := json.Marshal(map[string]interface{}{"wake": wake, "at": now.UTC().Format(time.RFC3339)})
		if err := h.upsertEntry(display.Category, display.LastSeenKey, string(seen)); err != nil {
			log.Printf("display: updating %s: %v", display.LastSeenKey, err)
		}
	}
}

// snapshotKey identifies a quotes snapshot for the "has it changed" check:
// the fetcher's timestamp when there is one, else the whole text.
func snapshotKey(q display.Quotes, raw string) string {
	if q.TimestampText != "" {
		return q.TimestampText
	}
	return raw
}

// upsertEntry sets an entry's value by key, creating it in category when it
// does not exist yet.
func (h *Handler) upsertEntry(category, key, value string) error {
	existing, err := h.store.GetEntryByKey(key)
	if err != nil {
		return err
	}
	if existing == nil {
		_, err = h.store.CreateEntry(&models.Entry{Category: category, Key: key, Value: value})
		return err
	}
	_, err = h.store.UpdateEntry(existing.ID, nil, nil, &value)
	return err
}

// etagMatches implements the If-None-Match comparison for a strong ETag:
// any listed tag equal to ours, ignoring a weak marker, or "*".
func etagMatches(header, etag string) bool {
	if header == "" {
		return false
	}
	for _, candidate := range strings.Split(header, ",") {
		candidate = strings.TrimSpace(candidate)
		candidate = strings.TrimPrefix(candidate, "W/")
		if candidate == "*" || candidate == etag {
			return true
		}
	}
	return false
}
