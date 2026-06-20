package chat

import (
	"encoding/binary"
	"errors"
	"sync"

	"github.com/google/uuid"
)

// Conn is the minimum surface the room manager needs from a transport.
// Handlers hold the real *websocket.Conn (or net.Conn); the manager only
// needs identity and lifecycle. Narrowing the dependency keeps the manager
// testable without spinning up a real socket.
type Conn interface {
	ID() uuid.UUID
	Close() error
}

// IRoomManager is a local, per-node membership index: which connections on
// this process are subscribed to which channel. It owns local state only.
// Cross-node delivery (Kafka publish / consume) is a separate concern and
// must NOT be added to this interface (see T09 design doc § cross-node fan-out).
type IRoomManager interface {
	Join(channelID uuid.UUID, conn Conn) error
	Leave(channelID uuid.UUID, connID uuid.UUID)
	List(channelID uuid.UUID) []Conn
	Count(channelID uuid.UUID) int
}

// ErrNilConn is returned when Join is called with a nil Conn. Callers must
// not pass nil — it indicates a programming error in the lifecycle layer.
var ErrNilConn = errors.New("chat: nil conn passed to room manager")

// shardCount is the number of independent lock domains. Must be a power of two
// so we can index with a bitmask (`& shardMask`) instead of integer modulo.
// Sized for the 1M-user target / ~30 WS nodes: ~33K conns/node, ~32-64
// concurrent writer goroutines during reconnect storms => 0.125–0.25
// writers/shard expected. Revisit only if mutex-profile shows shard contention.
const (
	shardCount = 256
	shardMask  = shardCount - 1
)

// NewRoomManager returns the sharded implementation. Each shard owns an
// independent rooms map and RWMutex; calls on different shards proceed in
// parallel. Calls on the same shard still serialize — that's expected and
// is what the design constrains.
func NewRoomManager() IRoomManager {
	m := &roomManager{}
	for i := range shardCount {
		m.shards[i].rooms = make(map[uuid.UUID]map[uuid.UUID]Conn)
	}
	return m
}

type roomManager struct {
	shards [shardCount]roomShard
}

// TODO(perf): pad roomShard to a 64-byte cache line if mutex profiles
// during BE-44 (pprof baseline) show false-sharing contention between
// adjacent shards. Premature padding without evidence is fake rigor.
type roomShard struct {
	mu    sync.RWMutex
	rooms map[uuid.UUID]map[uuid.UUID]Conn
}

// shardFor selects the shard owning a channelID by XOR-folding the UUID's
// two halves and masking with shardMask. Folding (not slicing a single byte)
// defends against UUIDv7 — its first 48 bits are a timestamp and would
// cluster on a few shards if used directly.
func (m *roomManager) shardFor(channelID uuid.UUID) *roomShard {
	hi := binary.BigEndian.Uint64(channelID[0:8])
	lo := binary.BigEndian.Uint64(channelID[8:16])
	return &m.shards[(hi^lo)&shardMask]
}

func (m *roomManager) Join(channelID uuid.UUID, conn Conn) error {
	if conn == nil {
		return ErrNilConn
	}
	s := m.shardFor(channelID)
	s.mu.Lock()
	defer s.mu.Unlock()
	room, ok := s.rooms[channelID]
	if !ok {
		room = make(map[uuid.UUID]Conn)
		s.rooms[channelID] = room
	}
	room[conn.ID()] = conn
	return nil
}

func (m *roomManager) Leave(channelID uuid.UUID, connID uuid.UUID) {
	s := m.shardFor(channelID)
	s.mu.Lock()
	defer s.mu.Unlock()
	room, ok := s.rooms[channelID]
	if !ok {
		return
	}
	delete(room, connID)
	if len(room) == 0 {
		delete(s.rooms, channelID)
	}
}

func (m *roomManager) List(channelID uuid.UUID) []Conn {
	s := m.shardFor(channelID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	room, ok := s.rooms[channelID]
	if !ok {
		return nil
	}
	out := make([]Conn, 0, len(room))
	for _, c := range room {
		out = append(out, c)
	}
	return out
}

func (m *roomManager) Count(channelID uuid.UUID) int {
	s := m.shardFor(channelID)
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.rooms[channelID])
}
