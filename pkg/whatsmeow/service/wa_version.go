package whatsmeow_service

import (
	"context"
	"net/http"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/store"

	logger_wrapper "github.com/evolution-foundation/evolution-go/pkg/logger"
)

// WhatsApp rejects clients whose web version is too old with a 405
// (events.ClientOutdated): sessions drop and no QR code is ever generated.
// The version compiled into whatsmeow goes stale between library updates, so
// it is refreshed from web.whatsapp.com at startup, periodically and whenever
// WhatsApp reports the client as outdated.
const (
	waVersionRefreshInterval = 6 * time.Hour
	waVersionMinRetry        = time.Minute
	waVersionFetchTimeout    = 15 * time.Second
	waVersionLogID           = "wa-version"
)

var (
	waVersionMu        sync.Mutex
	waVersionLastFetch time.Time
)

// refreshWAVersion fetches the current WhatsApp web version and applies it when
// it is newer than the one in use. Returns true when the version changed.
func refreshWAVersion(loggerWrapper *logger_wrapper.LoggerManager) bool {
	waVersionMu.Lock()
	defer waVersionMu.Unlock()

	if time.Since(waVersionLastFetch) < waVersionMinRetry {
		return false
	}
	waVersionLastFetch = time.Now()

	log := loggerWrapper.GetLogger(waVersionLogID)

	ctx, cancel := context.WithTimeout(context.Background(), waVersionFetchTimeout)
	defer cancel()

	latest, err := whatsmeow.GetLatestVersion(ctx, &http.Client{Timeout: waVersionFetchTimeout})
	if err != nil {
		log.LogWarn("Failed to fetch latest WhatsApp web version (using %s): %v", store.GetWAVersion(), err)
		return false
	}

	current := store.GetWAVersion()
	if !current.LessThan(*latest) {
		return false
	}

	store.SetWAVersion(*latest)
	log.LogInfo("WhatsApp web version updated from %s to %s", current, *latest)
	return true
}

func startWAVersionRefresher(loggerWrapper *logger_wrapper.LoggerManager) {
	refreshWAVersion(loggerWrapper)

	go func() {
		ticker := time.NewTicker(waVersionRefreshInterval)
		defer ticker.Stop()
		for range ticker.C {
			refreshWAVersion(loggerWrapper)
		}
	}()
}
