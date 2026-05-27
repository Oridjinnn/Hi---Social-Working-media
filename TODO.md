# HI POLISH — Implementation Progress

## Step 1 — Navigation + Detail polish (Tasks 1–3)
- [x] Update feed help bar text
- [x] Handle `esc` as quit in feed
- [x] Verify/ensure `↑/↓` cursor movement and no fallthrough on `→/enter`
- [x] Update detail help bar text
- [x] Add divider + one-line stats + left margin padding in detail view
- [x] Feed: verify empty/loading/error rendering is readable and styled


## Step 2 — Trend panel (Task 4)

- [x] Create `utils/cache.go`
- [x] Create `tui/trend.go`
- [x] Modify `tui/feed.go` to render trend panel above list

## Step 3 — P2P chat via GitHub Issue comments (Task 5)
- [x] Create `github/chat.go`
- [x] Create `tui/chat.go`
- [x] Wire `c` key from detail to open chat model
- [x] Enforce message limits via `~/.config/hi/connections.json`
- [x] Poll comments every 15s

## Step 6 — App navigation system (Tab UI)
- [x] Create `tui/app.go` — AppModel root with /home /feeds /grouphouse tabs
- [x] HomeModel — personal dashboard with embedded market intel + user stats
- [x] GroupModel — Group House placeholder with coming soon UI
- [x] Tab switching: `tab` key cycles, `1/2/3` jumps directly
- [x] Sub-view guard: tab switching disabled when detail/chat/wizard/market/filter active
- [x] Update `cmd/root.go` to launch AppModel instead of FeedModel
- [x] Fix connect: github/client.go now falls back to HI_SIGNAL_REPO_OWNER/NAME env vars

- [x] Create `market/intel.go` — fetch AI segment data from GitHub Search + HN Algolia API
- [x] Create `tui/market.go` — TUI view with sparkline chart + HTML export
- [x] Wire `m` key in feed to open market view
- [x] Add `esc/q` to return from market view to feed
- [x] HTML export opens in browser via `e` key inside market view
- [x] Fix `IsGhost bool` missing from `models/signal.go`

- [x] Add `IsGhost bool` to `models/signal.go`
- [x] Add `github/search.go` FetchGhostSignals
- [x] Modify feed rendering to show ghost items and handle `c/→` behavior

## Next 30 Days — Product + Hardening Backlog

### Domain Focus (Moat)
**Primary domain to dominate:** Developer collaboration matching + activation  
(`signal posted -> quality match -> chat started -> first build session`)

### P0 — Must ship first (Reliability + Trust)

#### 1) Realtime reliability hardening
- [x] Build connection state machine in `tui/realtime_bridge.go` (`connecting`, `connected`, `degraded`, `reconnecting`, `offline`)
- [x] Add exponential backoff + jitter reconnect strategy
- [x] Add dedupe for repeated events (same actor/signal/event in short window)
- [x] Add message ordering guard for out-of-order websocket events
- [x] Add visible status indicator in app shell (connection + last sync age)

**Acceptance criteria**
- Reconnects automatically after network drop without restart
- No duplicate toast/chat events during reconnect storms
- User can see when system is degraded/offline

#### 2) State consistency across tabs/views
- [x] Introduce app-level event bus/state sync path in `tui/app.go`
- [x] Standardize refresh triggers after signal create/connect/chat send
- [x] Prevent stale feed/detail/chat mismatch after updates
- [x] Add central "last updated" metadata for each tab model

**Acceptance criteria**
- Switching tabs never shows stale detail for deleted/updated signal
- New events reflect across relevant views within one update cycle

#### 3) Security + auth hygiene
- [x] Audit token usage in `config` + `github` clients (scope minimization)
- [x] Ensure no token/log leakage in errors or panic recovery output
- [x] Add startup auth validation with actionable remediation hints
- [x] Add secure storage fallback checks and clear warning when insecure

**Acceptance criteria**
- No token ever appears in logs/toasts/errors
- Invalid auth always yields deterministic recovery guidance

#### 4) Ranking trust baseline (anti-noise)
- [x] Add basic signal quality scoring (freshness, profile completeness, clarity)
- [x] Introduce low-quality/spam throttle in feed ordering
- [x] Add minimal reputation hints (response rate / accepted connections)

**Acceptance criteria**
- Top of feed is consistently actionable (less noise, more intent)

---

### P1 — Improve activation + usability

#### 5) Onboarding to first value in <2 min
- [ ] Add first-run guided flow in `tui/wizard.go`: create first signal + profile checks
- [ ] Add contextual helper prompts in empty states (`feed`, `grouphouse`, `market`)
- [ ] Add post-signal next-step CTA: "find matching collaborators now"

**Acceptance criteria**
- New user can post a signal and open first connection flow in one session

#### 6) Command discoverability + UX coherence
- [ ] Add global `?` command palette/help modal with per-tab controls
- [ ] Add consistent top context hints for all tabs (already started in `app.go`)
- [ ] Add compact/comfortable density toggle for feed and market panels

**Acceptance criteria**
- Users can discover core commands without reading docs

#### 7) Match quality improvements
- [ ] Add richer filter facets (intent, stack depth, timezone overlap, availability)
- [ ] Add compatibility badges in feed rows
- [ ] Add "why this match" explanation in detail view

**Acceptance criteria**
- Higher connect conversion from feed impression to chat start

---

### P2 — Insights, scale, and defensibility

#### 8) Funnel analytics and product telemetry
- [ ] Add event schema for funnel: `signal_created`, `signal_viewed`, `connect_clicked`, `chat_started`, `followup_sent`
- [ ] Build internal metrics view (TUI or export) for conversion/drop-off
- [ ] Add weekly report generator command (`hi digest` enhancement)

**Acceptance criteria**
- Team can quantify activation and identify exact funnel drop-offs

#### 9) Market intel differentiation
- [ ] Improve opportunity generation quality (less hardcoded templates)
- [ ] Add source provenance in market cards (top repos/HN links + recency)
- [ ] Add saved watchlist + alerting for tracked topics

**Acceptance criteria**
- Market Intel drives concrete actions, not just passive reading

#### 10) Ecosystem extensibility
- [ ] Add plugin-style provider interfaces for new data sources
- [ ] Add clear module contracts for feed ranking and recommendation strategies
- [ ] Add regression test harness for TUI flows + event scenarios

**Acceptance criteria**
- New intel/matching providers can be added without app-wide rewrites

---

## Execution Order (Recommended)
- [ ] Week 1: P0.1 + P0.2 (realtime and state consistency)
- [ ] Week 2: P0.3 + P0.4 (security and trust ranking)
- [ ] Week 3: P1.5 + P1.6 (onboarding and discoverability)
- [ ] Week 4: P1.7 + P2.8 baseline (match quality + funnel instrumentation)

## KPI Targets (Track weekly)
- [ ] Feed impression -> connect click rate
- [ ] Connect click -> chat started rate
- [ ] Chat started -> follow-up within 24h
- [ ] New user time-to-first-signal
- [ ] Realtime delivery success / reconnect success rate
