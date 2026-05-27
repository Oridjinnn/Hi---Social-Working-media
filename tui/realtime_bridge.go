package tui

import (
	"fmt"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/Oridjinnn/hi/models"
	"github.com/Oridjinnn/hi/supabase"
)

type RealtimeConnState string

const (
	RealtimeStateConnecting   RealtimeConnState = "connecting"
	RealtimeStateConnected    RealtimeConnState = "connected"
	RealtimeStateDegraded     RealtimeConnState = "degraded"
	RealtimeStateReconnecting RealtimeConnState = "reconnecting"
	RealtimeStateOffline      RealtimeConnState = "offline"
)

type realtimeBridgeStatusMsg struct {
	state    RealtimeConnState
	lastSync time.Time
	err      error
	attempt  int
}

type realtimeBridgeEventMsg struct {
	event models.ConnectionEvent
}

type RealtimeBridge struct {
	client   *supabase.Client
	username string

	eventCh  chan models.ConnectionEvent
	statusCh chan realtimeBridgeStatusMsg
	stopCh   chan struct{}

	seenIDs          map[string]time.Time
	seenFingerprints map[string]time.Time
	lastSeenAt       time.Time
}

func NewRealtimeBridge(client *supabase.Client, username string) *RealtimeBridge {
	return &RealtimeBridge{
		client:           client,
		username:         username,
		eventCh:          make(chan models.ConnectionEvent, 16),
		statusCh:         make(chan realtimeBridgeStatusMsg, 16),
		stopCh:           make(chan struct{}),
		seenIDs:          make(map[string]time.Time),
		seenFingerprints: make(map[string]time.Time),
	}
}

func (b *RealtimeBridge) Start() (<-chan models.ConnectionEvent, <-chan realtimeBridgeStatusMsg) {
	go b.loop()
	return b.eventCh, b.statusCh
}

func (b *RealtimeBridge) Stop() {
	select {
	case <-b.stopCh:
		return
	default:
		close(b.stopCh)
	}
}

func (b *RealtimeBridge) loop() {
	defer close(b.eventCh)
	defer close(b.statusCh)

	pollInterval := 12 * time.Second
	attempt := 0

	b.emitStatus(realtimeBridgeStatusMsg{state: RealtimeStateConnecting})

	for {
		select {
		case <-b.stopCh:
			b.emitStatus(realtimeBridgeStatusMsg{state: RealtimeStateOffline})
			return
		default:
		}

		events, err := b.client.GetPendingNotifications(b.username)
		now := time.Now()
		if err != nil {
			attempt++
			state := RealtimeStateDegraded
			if attempt > 1 {
				state = RealtimeStateReconnecting
			}
			b.emitStatus(realtimeBridgeStatusMsg{
				state:   state,
				err:     err,
				attempt: attempt,
			})

			backoff := backoffWithJitter(attempt)
			select {
			case <-time.After(backoff):
			case <-b.stopCh:
				b.emitStatus(realtimeBridgeStatusMsg{state: RealtimeStateOffline})
				return
			}
			continue
		}

		attempt = 0
		b.emitStatus(realtimeBridgeStatusMsg{
			state:    RealtimeStateConnected,
			lastSync: now,
		})
		b.emitNewEvents(events)

		select {
		case <-time.After(pollInterval):
		case <-b.stopCh:
			b.emitStatus(realtimeBridgeStatusMsg{state: RealtimeStateOffline})
			return
		}
	}
}

func (b *RealtimeBridge) emitStatus(msg realtimeBridgeStatusMsg) {
	select {
	case b.statusCh <- msg:
	default:
	}
}

func (b *RealtimeBridge) emitNewEvents(events []models.ConnectionEvent) {
	// API returns desc; process asc to preserve ordering for UI stream.
	for i := len(events) - 1; i >= 0; i-- {
		e := events[i]
		if !b.shouldEmit(e) {
			continue
		}
		select {
		case b.eventCh <- e:
		default:
		}
	}
	b.gcSeenCaches()
}

func (b *RealtimeBridge) shouldEmit(e models.ConnectionEvent) bool {
	now := time.Now()
	fp := eventFingerprint(e)

	if e.ID != "" {
		if _, exists := b.seenIDs[e.ID]; exists {
			return false
		}
	}

	if seenAt, exists := b.seenFingerprints[fp]; exists && now.Sub(seenAt) < 2*time.Minute {
		return false
	}

	// Ordering guard: avoid replaying stale events far behind the stream cursor.
	if !b.lastSeenAt.IsZero() && e.CreatedAt.Before(b.lastSeenAt.Add(-2*time.Second)) {
		return false
	}

	if e.ID != "" {
		b.seenIDs[e.ID] = now
	}
	b.seenFingerprints[fp] = now
	if e.CreatedAt.After(b.lastSeenAt) {
		b.lastSeenAt = e.CreatedAt
	}
	return true
}

func (b *RealtimeBridge) gcSeenCaches() {
	cutoff := time.Now().Add(-20 * time.Minute)
	for id, t := range b.seenIDs {
		if t.Before(cutoff) {
			delete(b.seenIDs, id)
		}
	}
	for fp, t := range b.seenFingerprints {
		if t.Before(cutoff) {
			delete(b.seenFingerprints, fp)
		}
	}
}

func eventFingerprint(e models.ConnectionEvent) string {
	return fmt.Sprintf("%d|%s|%s|%d", e.SignalID, e.ActorUsername, e.EventType, e.CreatedAt.Unix())
}

func backoffWithJitter(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	base := 2 * time.Second
	max := 45 * time.Second
	d := base * time.Duration(1<<(attempt-1))
	if d > max {
		d = max
	}
	// Deterministic, low-overhead jitter from wall clock.
	jitter := time.Duration(time.Now().UnixNano()%int64(600*time.Millisecond)) - 300*time.Millisecond
	if d+jitter > time.Second {
		d += jitter
	}
	return d
}

func waitRealtimeBridgeEvent(ch <-chan models.ConnectionEvent) tea.Cmd {
	return func() tea.Msg {
		ev, ok := <-ch
		if !ok {
			return nil
		}
		return realtimeBridgeEventMsg{event: ev}
	}
}

func waitRealtimeBridgeStatus(ch <-chan realtimeBridgeStatusMsg) tea.Cmd {
	return func() tea.Msg {
		status, ok := <-ch
		if !ok {
			return nil
		}
		return status
	}
}
