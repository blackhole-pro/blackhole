// Package mesh provides the mesh networking domain for the Blackhole Framework.
// This domain handles communication, discovery, and coordination between nodes.
package mesh

import (
	"context"
	"net"
	"time"
)

// MeshNetwork is the main interface for mesh networking.
type MeshNetwork interface {
	// Node management
	RegisterNode(nodeInfo NodeInfo) error
	UnregisterNode(nodeID NodeID) error
	UpdateNode(nodeInfo NodeInfo) error
	
	// Node discovery
	DiscoverNodes(criteria DiscoveryCriteria) ([]NodeInfo, error)
	GetNode(nodeID NodeID) (NodeInfo, error)
	ListNodes() ([]NodeInfo, error)
	
	// Communication
	SendMessage(target NodeID, message Message) error
	BroadcastMessage(message Message) error
	SendRequest(target NodeID, request Request) (Response, error)
	
	// Routing
	SetupRoute(from NodeID, to NodeID) (Route, error)
	GetOptimalRoute(target NodeID) (Route, error)
	
	// Network topology
	GetTopology() (NetworkTopology, error)
	UpdateTopology() error
}

// NodeID represents a unique node identifier.
type NodeID string

// NodeInfo contains information about a network node.
type NodeInfo struct {
	ID           NodeID            `json:"id"`
	Name         string            `json:"name"`
	Type         NodeType          `json:"type"`
	Status       NodeStatus        `json:"status"`
	Endpoints    []Endpoint        `json:"endpoints"`
	Capabilities []NodeCapability  `json:"capabilities"`
	Resources    NodeResources     `json:"resources"`
	Metadata     map[string]string `json:"metadata"`
	LastSeen     time.Time         `json:"last_seen"`
	Version      string            `json:"version"`
}

// NodeType defines the type of node.
type NodeType string

const (
	NodeTypeCore        NodeType = "core"        // Core framework node
	NodeTypePlugin      NodeType = "plugin"      // Plugin execution node
	NodeTypeStorage     NodeType = "storage"     // Storage provider node
	NodeTypeCompute     NodeType = "compute"     // Compute provider node
	NodeTypeRelay       NodeType = "relay"       // Relay/proxy node
	NodeTypeBootstrap   NodeType = "bootstrap"   // Bootstrap node
)

// NodeStatus represents the current status of a node.
type NodeStatus string

const (
	NodeStatusUnknown     NodeStatus = "unknown"
	NodeStatusConnecting  NodeStatus = "connecting"
	NodeStatusOnline      NodeStatus = "online"
	NodeStatusDegraded    NodeStatus = "degraded"
	NodeStatusOffline     NodeStatus = "offline"
	NodeStatusMaintenance NodeStatus = "maintenance"
)

// Endpoint represents a network endpoint for communication.
type Endpoint struct {
	Protocol  string `json:"protocol"` // tcp, udp, quic, websocket
	Address   string `json:"address"`  // IP:port or hostname:port
	Public    bool   `json:"public"`   // Whether endpoint is publicly accessible
	Secure    bool   `json:"secure"`   // Whether endpoint uses encryption
	Priority  int    `json:"priority"` // Priority for connection attempts
}

// NodeCapability defines what a node can do.
type NodeCapability string

const (
	CapabilityPluginExecution NodeCapability = "plugin_execution"
	CapabilityStorage         NodeCapability = "storage"
	CapabilityCompute         NodeCapability = "compute"
	CapabilityRelay           NodeCapability = "relay"
	CapabilityDiscovery       NodeCapability = "discovery"
	CapabilityCoordination    NodeCapability = "coordination"
)

// NodeResources represents available resources on a node.
type NodeResources struct {
	CPU struct {
		Cores     int     `json:"cores"`
		Available float64 `json:"available_percent"`
	} `json:"cpu"`
	
	Memory struct {
		Total     uint64 `json:"total_bytes"`
		Available uint64 `json:"available_bytes"`
	} `json:"memory"`
	
	Storage struct {
		Total     uint64 `json:"total_bytes"`
		Available uint64 `json:"available_bytes"`
	} `json:"storage"`
	
	Network struct {
		Bandwidth uint64 `json:"bandwidth_bps"`
		Latency   time.Duration `json:"latency"`
	} `json:"network"`
}

// DiscoveryCriteria defines criteria for node discovery.
type DiscoveryCriteria struct {
	NodeType     NodeType         `json:"node_type,omitempty"`
	Capabilities []NodeCapability `json:"capabilities,omitempty"`
	MinResources NodeResources    `json:"min_resources,omitempty"`
	MaxLatency   time.Duration    `json:"max_latency,omitempty"`
	Region       string           `json:"region,omitempty"`
	Tags         map[string]string `json:"tags,omitempty"`
}

// Message represents a message sent between nodes.
type Message struct {
	ID          string            `json:"id"`
	Type        MessageType       `json:"type"`
	Source      NodeID            `json:"source"`
	Destination NodeID            `json:"destination"`
	Payload     []byte            `json:"payload"`
	Headers     map[string]string `json:"headers"`
	Timestamp   time.Time         `json:"timestamp"`
	TTL         time.Duration     `json:"ttl"`
	Priority    MessagePriority   `json:"priority"`
}

// MessageType defines the type of message.
type MessageType string

const (
	MessageTypeData        MessageType = "data"
	MessageTypeControl     MessageType = "control"
	MessageTypeHeartbeat   MessageType = "heartbeat"
	MessageTypeDiscovery   MessageType = "discovery"
	MessageTypeBroadcast   MessageType = "broadcast"
)

// MessagePriority defines the priority of a message.
type MessagePriority int

const (
	PriorityLow    MessagePriority = iota
	PriorityNormal
	PriorityHigh
	PriorityCritical
)

// Request represents a request sent to a node.
type Request struct {
	ID       string                 `json:"id"`
	Method   string                 `json:"method"`
	Params   map[string]interface{} `json:"params"`
	Data     []byte                 `json:"data,omitempty"`
	Timeout  time.Duration          `json:"timeout"`
	Headers  map[string]string      `json:"headers,omitempty"`
}

// Response represents a response from a node.
type Response struct {
	ID      string                 `json:"id"`
	Success bool                   `json:"success"`
	Result  map[string]interface{} `json:"result,omitempty"`
	Data    []byte                 `json:"data,omitempty"`
	Error   string                 `json:"error,omitempty"`
	Headers map[string]string      `json:"headers,omitempty"`
}

// Route represents a communication route between nodes.
type Route struct {
	Source      NodeID        `json:"source"`
	Destination NodeID        `json:"destination"`
	Hops        []NodeID      `json:"hops"`
	Protocol    string        `json:"protocol"`
	Latency     time.Duration `json:"latency"`
	Bandwidth   uint64        `json:"bandwidth"`
	Cost        float64       `json:"cost"`
	Reliability float64       `json:"reliability"`
}

// NetworkTopology represents the network topology.
type NetworkTopology struct {
	Nodes       []NodeInfo      `json:"nodes"`
	Connections []Connection    `json:"connections"`
	Routes      []Route         `json:"routes"`
	Regions     []Region        `json:"regions"`
	UpdateTime  time.Time       `json:"update_time"`
}

// Connection represents a connection between two nodes.
type Connection struct {
	From        NodeID        `json:"from"`
	To          NodeID        `json:"to"`
	Protocol    string        `json:"protocol"`
	Status      ConnectionStatus `json:"status"`
	Latency     time.Duration `json:"latency"`
	Bandwidth   uint64        `json:"bandwidth"`
	LastActive  time.Time     `json:"last_active"`
}

// ConnectionStatus represents the status of a connection.
type ConnectionStatus string

const (
	ConnectionStatusUnknown     ConnectionStatus = "unknown"
	ConnectionStatusConnecting  ConnectionStatus = "connecting"
	ConnectionStatusConnected   ConnectionStatus = "connected"
	ConnectionStatusDisconnected ConnectionStatus = "disconnected"
	ConnectionStatusFailed      ConnectionStatus = "failed"
)

// Region represents a network region.
type Region struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Location    Location  `json:"location"`
	Nodes       []NodeID  `json:"nodes"`
	Latency     time.Duration `json:"avg_latency"`
	Reliability float64   `json:"reliability"`
}

// Location represents a geographical location.
type Location struct {
	Country   string  `json:"country"`
	Region    string  `json:"region"`
	City      string  `json:"city"`
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// NodeDiscovery handles node discovery and registration.
type NodeDiscovery interface {
	// Node registration
	RegisterNode(nodeInfo NodeInfo) error
	UnregisterNode(nodeID NodeID) error
	
	// Node discovery
	DiscoverNodes(criteria DiscoveryCriteria) ([]NodeInfo, error)
	FindNearestNodes(location Location, count int) ([]NodeInfo, error)
	
	// Node health monitoring
	StartHealthMonitoring() error
	StopHealthMonitoring() error
	CheckNodeHealth(nodeID NodeID) (NodeHealth, error)
}

// NodeHealth represents the health status of a node.
type NodeHealth struct {
	NodeID       NodeID        `json:"node_id"`
	Status       NodeStatus    `json:"status"`
	LastCheck    time.Time     `json:"last_check"`
	ResponseTime time.Duration `json:"response_time"`
	ErrorCount   int           `json:"error_count"`
	Uptime       time.Duration `json:"uptime"`
}

// MessageRouter handles message routing and delivery.
type MessageRouter interface {
	// Message routing
	RouteMessage(message Message) error
	BroadcastMessage(message Message) error
	
	// Route management
	FindRoute(from, to NodeID) (Route, error)
	UpdateRoutes() error
	GetRoutingTable() (RoutingTable, error)
	
	// Load balancing
	SelectNode(criteria DiscoveryCriteria) (NodeID, error)
	DistributeLoad(nodes []NodeID, message Message) error
}

// RoutingTable represents the routing table.
type RoutingTable map[NodeID]Route

// Transport handles low-level communication protocols.
type Transport interface {
	// Connection management
	Connect(endpoint Endpoint) (Connection, error)
	Disconnect(connection Connection) error
	
	// Message transport
	Send(connection Connection, data []byte) error
	Receive(connection Connection) ([]byte, error)
	
	// Protocol support
	GetSupportedProtocols() []string
	CreateListener(endpoint Endpoint) (Listener, error)
}

// Listener represents a network listener.
type Listener interface {
	Accept() (Connection, error)
	Close() error
	Addr() net.Addr
}

// TopologyManager handles network topology management.
type TopologyManager interface {
	// Topology discovery
	DiscoverTopology() (NetworkTopology, error)
	UpdateTopology() error
	
	// Topology optimization
	OptimizeTopology() error
	SuggestConnections() ([]Connection, error)
	
	// Topology analysis
	AnalyzeLatency() (LatencyMap, error)
	DetectPartitions() ([]Partition, error)
	CalculateReliability() (float64, error)
}

// LatencyMap represents latency between nodes.
type LatencyMap map[NodeID]map[NodeID]time.Duration

// Partition represents a network partition.
type Partition struct {
	ID    string   `json:"id"`
	Nodes []NodeID `json:"nodes"`
	Size  int      `json:"size"`
}

// SecurityManager handles mesh security.
type SecurityManager interface {
	// Authentication
	AuthenticateNode(nodeID NodeID, credentials Credentials) error
	IssueCredentials(nodeID NodeID) (Credentials, error)
	RevokeCredentials(nodeID NodeID) error
	
	// Authorization
	AuthorizeAction(nodeID NodeID, action Action) error
	GetPermissions(nodeID NodeID) ([]Permission, error)
	
	// Encryption
	EncryptMessage(message Message, target NodeID) ([]byte, error)
	DecryptMessage(data []byte, source NodeID) (Message, error)
	
	// Trust management
	EstablishTrust(nodeID NodeID) error
	GetTrustLevel(nodeID NodeID) (TrustLevel, error)
	UpdateTrustLevel(nodeID NodeID, level TrustLevel) error
}

// Credentials represent node credentials.
type Credentials struct {
	NodeID      NodeID    `json:"node_id"`
	PublicKey   []byte    `json:"public_key"`
	Certificate []byte    `json:"certificate"`
	Signature   []byte    `json:"signature"`
	ExpiresAt   time.Time `json:"expires_at"`
}

// Action represents an action that can be authorized.
type Action string

const (
	ActionSendMessage    Action = "send_message"
	ActionReceiveMessage Action = "receive_message"
	ActionJoinNetwork    Action = "join_network"
	ActionLeaveNetwork   Action = "leave_network"
	ActionDiscoverNodes  Action = "discover_nodes"
	ActionExecutePlugin  Action = "execute_plugin"
)

// Permission represents a permission granted to a node.
type Permission struct {
	Action    Action    `json:"action"`
	Resource  string    `json:"resource"`
	ExpiresAt time.Time `json:"expires_at"`
}

// TrustLevel represents the trust level of a node.
type TrustLevel int

const (
	TrustLevelUntrusted TrustLevel = iota
	TrustLevelLow
	TrustLevelMedium
	TrustLevelHigh
	TrustLevelTrusted
)