package grouphouse

import "time"

// ── Message types ───────────────────────────────────────────────────────────

type MessageType string

const (
	MsgJoin          MessageType = "join"
	MsgLeave         MessageType = "leave"
	MsgFileWrite     MessageType = "file_write"
	MsgFileRead      MessageType = "file_read"
	MsgFileChanged   MessageType = "file_changed"
	MsgFileDeleted   MessageType = "file_deleted"
	MsgRun           MessageType = "run"
	MsgRunResult     MessageType = "run_result"
	MsgBroadcast     MessageType = "broadcast"
	MsgDirectMessage MessageType = "direct_message"
	MsgAgentJoined   MessageType = "agent_joined"
	MsgAgentLeft     MessageType = "agent_left"
	MsgError         MessageType = "error"
	MsgListAgents    MessageType = "list_agents"
	MsgAgentList     MessageType = "agent_list"
	MsgWorkspaceTree MessageType = "workspace_tree"
	MsgPing          MessageType = "ping"
	MsgPong          MessageType = "pong"
)

type ParticipantKind string

const (
	KindHousemaster ParticipantKind = "housemaster"
	KindAgent       ParticipantKind = "agent"
	KindGuest       ParticipantKind = "guest"
)

// ── Message envelope ────────────────────────────────────────────────────────

type Message struct {
	Type       MessageType     `json:"type"`
	Sender     string          `json:"sender,omitempty"`
	SenderKind ParticipantKind `json:"sender_kind,omitempty"`
	Payload    interface{}     `json:"payload,omitempty"`
	Timestamp  time.Time       `json:"timestamp"`
}

// ── Payload types ───────────────────────────────────────────────────────────

type JoinPayload struct {
	Name    string          `json:"name"`
	Kind    ParticipantKind `json:"kind"`
	AgentID string          `json:"agent_id,omitempty"` // unique identifier for agents
}

type LeavePayload struct {
	Reason string `json:"reason,omitempty"`
}

type FileWritePayload struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type FileReadPayload struct {
	Path string `json:"path"`
}

type FileChangedPayload struct {
	Path   string `json:"path"`
	Author string `json:"author"`
	Action string `json:"action"` // "write", "delete"
}

type RunPayload struct {
	Command string `json:"command"`
	Timeout int    `json:"timeout,omitempty"` // seconds, 0 = no timeout
}

type RunResultPayload struct {
	Command  string `json:"command"`
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	Duration string `json:"duration"` // human-readable
}

type BroadcastPayload struct {
	Text string `json:"text"`
}

type DirectMessagePayload struct {
	Target string `json:"target"` // agent name
	Text   string `json:"text"`
}

type AgentInfo struct {
	Name        string          `json:"name"`
	Kind        ParticipantKind `json:"kind"`
	AgentID     string          `json:"agent_id,omitempty"`
	ConnectedAt time.Time       `json:"connected_at"`
	LastActive  time.Time       `json:"last_active"`
}

type AgentListPayload struct {
	Agents        []AgentInfo `json:"agents"`
	HouseName     string      `json:"house_name"`
	WorkspacePath string      `json:"workspace_path"`
}

type WorkspaceTreePayload struct {
	Files []string `json:"files"`
}

type ErrorPayload struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// ── Message constructors ────────────────────────────────────────────────────

func NewMessage(msgType MessageType, sender string, kind ParticipantKind, payload interface{}) Message {
	return Message{
		Type:       msgType,
		Sender:     sender,
		SenderKind: kind,
		Payload:    payload,
		Timestamp:  time.Now(),
	}
}
