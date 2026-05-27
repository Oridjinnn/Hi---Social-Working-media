package grouphouse

import (
	"encoding/json"
	"fmt"
	"net"

	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true // Allow all origins for local agent connections
	},
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// Agent represents a connected participant (housemaster, agent, or guest).
type Agent struct {
	Name        string
	Kind        ParticipantKind
	AgentID     string
	Conn        *websocket.Conn
	ConnectedAt time.Time
	LastActive  time.Time
	mu          sync.Mutex
}

func (a *Agent) Send(msg Message) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.Conn.WriteJSON(msg)
}

// Server is the group house WebSocket server.
type Server struct {
	name      string
	port      int
	workspace *Workspace

	agentsMu sync.RWMutex
	agents   map[string]*Agent // keyed by name

	housemaster *Agent

	broadcast  chan Message
	register   chan *Agent
	unregister chan *Agent

	eventLog   []Message
	eventLogMu sync.RWMutex
	maxLogSize int

	httpServer *http.Server
}

func NewServer(name string, port int, workspace *Workspace) *Server {
	return &Server{
		name:       name,
		port:       port,
		workspace:  workspace,
		agents:     make(map[string]*Agent),
		broadcast:  make(chan Message, 100),
		register:   make(chan *Agent),
		unregister: make(chan *Agent),
		eventLog:   make([]Message, 0, 100),
		maxLogSize: 1000,
	}
}

// Start begins listening for WebSocket connections.
func (s *Server) Start() error {
	mux := http.NewServeMux()
	mux.HandleFunc("/ws", s.handleWebSocket)
	mux.HandleFunc("/health", s.handleHealth)

	addr := fmt.Sprintf(":%d", s.port)
	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", addr, err)
	}

	s.httpServer = &http.Server{
		Handler:      mux,
		ReadTimeout:  60 * time.Second,
		WriteTimeout: 60 * time.Second,
	}

	go s.run()

	return s.httpServer.Serve(listener)
}

// Stop gracefully shuts down the server.
func (s *Server) Stop() {
	if s.httpServer != nil {
		s.httpServer.Close()
	}
}

// run is the main event loop for the server.
func (s *Server) run() {
	// Guardrail: Ensure the event loop stays alive even if a logic error occurs
	defer func() {
		if r := recover(); r != nil {
			// Restarting the loop is part of the 'self-fixing' strategy
			go s.run()
		}
	}()
	for {
		select {
		case agent := <-s.register:
			s.agentsMu.Lock()
			s.agents[agent.Name] = agent
			s.agentsMu.Unlock()

			// Announce to everyone
			joinMsg := NewMessage(MsgAgentJoined, agent.Name, agent.Kind, nil)
			s.broadcastToAll(joinMsg)

			// Send the current agent list to the new participant
			s.sendAgentList(agent)

			// Log the event
			s.logEvent(joinMsg)

		case agent := <-s.unregister:
			s.agentsMu.Lock()
			delete(s.agents, agent.Name)
			if s.housemaster == agent {
				s.housemaster = nil
			}
			s.agentsMu.Unlock()

			leaveMsg := NewMessage(MsgAgentLeft, agent.Name, agent.Kind, nil)
			s.broadcastToAll(leaveMsg)
			s.logEvent(leaveMsg)

		case msg := <-s.broadcast:
			s.broadcastToAll(msg)
			s.logEvent(msg)
		}
	}
}

func (s *Server) handleWebSocket(w http.ResponseWriter, r *http.Request) {
	defer func() {
		if r := recover(); r != nil {
			// Connection cleanup handled by net/http
		}
	}()
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		// Intentionally no stdout/stderr logging here; server runs behind a TUI.
		return
	}

	// Wait for join message
	var joinMsg Message
	if err := conn.ReadJSON(&joinMsg); err != nil {
		// No stdout/stderr logging in TUI mode.
		conn.Close()
		return
	}

	if joinMsg.Type != MsgJoin {
		sendError(conn, 400, "first message must be a join")
		conn.Close()
		return
	}

	payloadBytes, _ := json.Marshal(joinMsg.Payload)
	var join JoinPayload
	json.Unmarshal(payloadBytes, &join)

	agent := &Agent{
		Name:        join.Name,
		Kind:        join.Kind,
		AgentID:     join.AgentID,
		Conn:        conn,
		ConnectedAt: time.Now(),
		LastActive:  time.Now(),
	}

	// Validate
	if join.Name == "" {
		sendError(conn, 400, "name is required")
		conn.Close()
		return
	}

	// Check for duplicate names
	s.agentsMu.RLock()
	_, exists := s.agents[join.Name]
	s.agentsMu.RUnlock()
	if exists {
		sendError(conn, 409, fmt.Sprintf("name '%s' is already taken", join.Name))
		conn.Close()
		return
	}

	// First housemaster gets special status
	if join.Kind == KindHousemaster || s.housemaster == nil {
		agent.Kind = KindHousemaster
	}

	s.register <- agent

	if agent.Kind == KindHousemaster {
		s.agentsMu.Lock()
		s.housemaster = agent
		s.agentsMu.Unlock()
	}

	// Send welcome
	welcome := NewMessage(MsgAgentList, "system", KindHousemaster, AgentListPayload{
		HouseName:     s.name,
		WorkspacePath: s.workspace.Path,
	})
	agent.Send(welcome)

	// Main message loop
	defer func() {
		s.unregister <- agent
		conn.Close()
	}()

	for {
		var msg Message
		if err := conn.ReadJSON(&msg); err != nil {
			// Suppress websocket error logs to avoid corrupting the TUI output.
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				// no-op
			}
			break
		}

		msg.Sender = agent.Name
		msg.SenderKind = agent.Kind
		msg.Timestamp = time.Now()

		agent.LastActive = time.Now()

		if err := s.handleMessage(agent, msg); err != nil {
			// No stdout/stderr logging in TUI mode.
			sendError(conn, 500, err.Error())
		}
	}
}

func (s *Server) handleMessage(sender *Agent, msg Message) error {
	switch msg.Type {
	case MsgPing:
		return sender.Send(NewMessage(MsgPong, "system", KindHousemaster, nil))

	case MsgBroadcast:
		s.broadcast <- msg

	case MsgDirectMessage:
		var payload DirectMessagePayload
		payloadBytes, _ := json.Marshal(msg.Payload)
		json.Unmarshal(payloadBytes, &payload)

		s.agentsMu.RLock()
		target, ok := s.agents[payload.Target]
		s.agentsMu.RUnlock()

		if !ok {
			return sender.Send(NewMessage(MsgError, "system", KindHousemaster, ErrorPayload{
				Code:    404,
				Message: fmt.Sprintf("agent '%s' not found", payload.Target),
			}))
		}

		return target.Send(msg)

	case MsgFileWrite:
		var payload FileWritePayload
		payloadBytes, _ := json.Marshal(msg.Payload)
		json.Unmarshal(payloadBytes, &payload)

		_, err := s.workspace.WriteFile(payload.Path, payload.Content)
		if err != nil {

			return sender.Send(NewMessage(MsgError, "system", KindHousemaster, ErrorPayload{
				Code:    500,
				Message: fmt.Sprintf("write error: %v", err),
			}))
		}

		// Notify all agents of the change
		s.broadcast <- NewMessage(MsgFileChanged, sender.Name, sender.Kind, FileChangedPayload{
			Path:   payload.Path,
			Author: sender.Name,
			Action: "write",
		})

	case MsgFileRead:
		var payload FileReadPayload
		payloadBytes, _ := json.Marshal(msg.Payload)
		json.Unmarshal(payloadBytes, &payload)

		content, err := s.workspace.ReadFile(payload.Path)
		if err != nil {
			return sender.Send(NewMessage(MsgError, "system", KindHousemaster, ErrorPayload{
				Code:    404,
				Message: fmt.Sprintf("file not found: %s", payload.Path),
			}))
		}

		return sender.Send(NewMessage(MsgFileChanged, "system", KindHousemaster, FileWritePayload{
			Path:    payload.Path,
			Content: content,
		}))

	case MsgRun:
		var payload RunPayload
		payloadBytes, _ := json.Marshal(msg.Payload)
		json.Unmarshal(payloadBytes, &payload)

		start := time.Now()
		stdout, stderr, exitCode, _ := s.workspace.Run(payload.Command)
		duration := time.Since(start)

		result := NewMessage(MsgRunResult, "system", KindHousemaster, RunResultPayload{
			Command:  payload.Command,
			ExitCode: exitCode,
			Stdout:   stdout,
			Stderr:   stderr,
			Duration: FormatDuration(duration),
		})

		// Send result to all participants
		s.broadcast <- result

	case MsgListAgents:
		return s.sendAgentList(sender)

	case MsgWorkspaceTree:
		files := s.workspace.Tree()
		return sender.Send(NewMessage(MsgWorkspaceTree, "system", KindHousemaster, WorkspaceTreePayload{
			Files: files,
		}))
	}

	return nil
}

func (s *Server) broadcastToAll(msg Message) {
	s.agentsMu.RLock()
	defer s.agentsMu.RUnlock()

	for _, agent := range s.agents {
		agent.Send(msg)
	}
}

func (s *Server) sendAgentList(target *Agent) error {
	s.agentsMu.RLock()
	agents := make([]AgentInfo, 0, len(s.agents))
	for _, a := range s.agents {
		agents = append(agents, AgentInfo{
			Name:        a.Name,
			Kind:        a.Kind,
			AgentID:     a.AgentID,
			ConnectedAt: a.ConnectedAt,
			LastActive:  a.LastActive,
		})
	}
	s.agentsMu.RUnlock()

	return target.Send(NewMessage(MsgAgentList, "system", KindHousemaster, AgentListPayload{
		Agents:        agents,
		HouseName:     s.name,
		WorkspacePath: s.workspace.Path,
	}))
}

func (s *Server) logEvent(msg Message) {
	s.eventLogMu.Lock()
	defer s.eventLogMu.Unlock()
	s.eventLog = append(s.eventLog, msg)
	if len(s.eventLog) > s.maxLogSize {
		s.eventLog = s.eventLog[len(s.eventLog)-s.maxLogSize:]
	}
}

func (s *Server) GetEventLog() []Message {
	s.eventLogMu.RLock()
	defer s.eventLogMu.RUnlock()
	result := make([]Message, len(s.eventLog))
	copy(result, s.eventLog)
	return result
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	s.agentsMu.RLock()
	count := len(s.agents)
	s.agentsMu.RUnlock()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "ok",
		"house":  s.name,
		"agents": count,
		"uptime": time.Now().Format(time.RFC3339),
	})
}

func sendError(conn *websocket.Conn, code int, message string) {
	msg := NewMessage(MsgError, "system", KindHousemaster, ErrorPayload{
		Code:    code,
		Message: message,
	})
	conn.WriteJSON(msg)
}
