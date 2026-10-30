package whatsmeow_service

import "sync"

// /instance/connect starts the client in a goroutine and returns at once, so a
// /instance/qr (or any endpoint that auto-starts the instance) arriving before
// the client is registered in clientPointer used to start a second client.
// Both generated QR codes; after one was scanned, the other hit the QR limit
// and logged the freshly paired session out. Only one start per instance may
// be in flight until its client is registered.
var (
	startingClientsMu sync.Mutex
	startingClients   = map[string]struct{}{}
)

func beginClientStart(instanceID string) bool {
	startingClientsMu.Lock()
	defer startingClientsMu.Unlock()

	if _, ok := startingClients[instanceID]; ok {
		return false
	}
	startingClients[instanceID] = struct{}{}
	return true
}

func endClientStart(instanceID string) {
	startingClientsMu.Lock()
	defer startingClientsMu.Unlock()

	delete(startingClients, instanceID)
}
