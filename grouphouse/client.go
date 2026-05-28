package grouphouse

import (
	"encoding/json"
	"fmt"
	"net/url"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Client is a WebSocket client that connects to a Group House server.
// Used by agents (AI tools) and guests to join a housemaster's session.
type Client struct {
	serverURL string
	name      string
	kind      ParticipantKind
	agentID   string

	conn *websocket.Conn
	mu   sync.Mutex
	done chan struct{}

	// Callbacks
	OnMessage      func(Message)
	OnAgentJoined  func(AgentInfo)
	OnAgentLeft    func(string)
	OnFileChanged  func(FileChangedPayload)
	OnRunResult    func(RunResultPayload)
	OnError        func(ErrorPayload)
	OnConnected    func()
	OnDisconnected func(error)
}

func NewClient(serverURL, name string, kind ParticipantKind, agentID string) *Client {
	return &Client{
		serverURL: serverURL,
		name:      name,
		kind:      kind,
		agentID:   agentID,
		done:      make(chan struct{}),
	}
}

// Connect establishes the WebSocket connection and sends a join message.
func (c *Client) Connect() error {
	u, err := url.Parse(c.serverURL)
	if err != nil {
		return fmt.Errorf("invalid server URL: %w", err)
	}
	u.Path = "/ws"

	conn, _, err := websocket.DefaultDialer.Dial(u.String(), nil)
	if err != nil {
		return fmt.Errorf("websocket dial: %w", err)
	}
	c.conn = conn

	// Send join message
	join := NewMessage(MsgJoin, c.name, c.kind, JoinPayload{
		Name:    c.name,
		Kind:    c.kind,
		AgentID: c.agentID,
	})

	if err := conn.WriteJSON(join); err != nil {
		_ = conn.Close()
		return fmt.Errorf("sending join: %w", err)
	}

	if c.OnConnected != nil {
		c.OnConnected()
	}

	go c.readLoop()

	// Start ping ticker
	go c.pingLoop()

	return nil
}

func (c *Client) readLoop() {
	defer func() {
		_ = c.conn.Close()
		if c.OnDisconnected != nil {
			c.OnDisconnected(nil)
		}
	}()

	for {
		var msg Message
		if err := c.conn.ReadJSON(&msg); err != nil {
			// Suppress stdout/stderr logging to keep BubbleTea TUI intact.
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseNormalClosure) {
				return
			}

			return
		}

		c.handleMessage(msg)
	}
}

func (c *Client) pingLoop() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-c.done:
			return
		case <-ticker.C:
			_ = c.Send(NewMessage(MsgPing, c.name, c.kind, nil))
		}
	}
}

func (c *Client) handleMessage(msg Message) {
	switch msg.Type {
	case MsgPong:
		// Heartbeat response — nothing to do

	case MsgAgentList:
		if c.OnMessage != nil {
			c.OnMessage(msg)
		}

	case MsgAgentJoined:
		var info AgentInfo
		if data, err := json.Marshal(msg.Payload); err == nil {
			if err := json.Unmarshal(data, &info); err != nil {
				return
			}
		}
		info.Name = msg.Sender
		if c.OnAgentJoined != nil {
			c.OnAgentJoined(info)
		}

	case MsgAgentLeft:
		if c.OnAgentLeft != nil {
			c.OnAgentLeft(msg.Sender)
		}

	case MsgFileChanged:
		var payload FileChangedPayload
		if data, err := json.Marshal(msg.Payload); err == nil {
			if err := json.Unmarshal(data, &payload); err != nil {
				return
			}
		}
		if c.OnFileChanged != nil {
			c.OnFileChanged(payload)
		}

	case MsgRunResult:
		var payload RunResultPayload
		if data, err := json.Marshal(msg.Payload); err == nil {
			if err := json.Unmarshal(data, &payload); err != nil {
				return
			}
		}
		if c.OnRunResult != nil {
			c.OnRunResult(payload)
		}

	case MsgBroadcast:
		if c.OnMessage != nil {
			c.OnMessage(msg)
		}

	case MsgDirectMessage:
		if c.OnMessage != nil {
			c.OnMessage(msg)
		}

	case MsgError:
		var payload ErrorPayload
		if data, err := json.Marshal(msg.Payload); err == nil {
			if err := json.Unmarshal(data, &payload); err != nil {
				return
			}
		}
		if c.OnError != nil {
			c.OnError(payload)
		}

	case MsgWorkspaceTree:
		if c.OnMessage != nil {
			c.OnMessage(msg)
		}

	default:
		if c.OnMessage != nil {
			c.OnMessage(msg)
		}
	}
}

// Send sends a message to the server.
func (c *Client) Send(msg Message) error {
	msg.Sender = c.name
	msg.SenderKind = c.kind
	msg.Timestamp = time.Now()

	c.mu.Lock()
	defer c.mu.Unlock()

	return c.conn.WriteJSON(msg)
}

// WriteFile sends a file_write command to the server.
func (c *Client) WriteFile(path, content string) error {
	return c.Send(NewMessage(MsgFileWrite, c.name, c.kind, FileWritePayload{
		Path:    path,
		Content: content,
	}))
}

// RunCommand sends a run command to the server.
func (c *Client) RunCommand(command string, timeout int) error {
	return c.Send(NewMessage(MsgRun, c.name, c.kind, RunPayload{
		Command: command,
		Timeout: timeout,
	}))
}

// Broadcast sends a text message to all participants.
func (c *Client) Broadcast(text string) error {
	return c.Send(NewMessage(MsgBroadcast, c.name, c.kind, BroadcastPayload{
		Text: text,
	}))
}

// DirectMessage sends a private message to a specific participant.
func (c *Client) DirectMessage(target, text string) error {
	return c.Send(NewMessage(MsgDirectMessage, c.name, c.kind, DirectMessagePayload{
		Target: target,
		Text:   text,
	}))
}

// ListAgents requests the current agent list.
func (c *Client) ListAgents() error {
	return c.Send(NewMessage(MsgListAgents, c.name, c.kind, nil))
}

// RequestWorkspaceTree requests the current workspace file tree.
func (c *Client) RequestWorkspaceTree() error {
	return c.Send(NewMessage(MsgWorkspaceTree, c.name, c.kind, nil))
}

// Close disconnects from the server.
func (c *Client) Close() error {
	close(c.done)
	if c.conn != nil {
		return c.conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
	}
	return nil
}
