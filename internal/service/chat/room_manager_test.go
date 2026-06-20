package chat_test

import (
	"errors"
	"sync"
	"testing"

	"github.com/google/uuid"

	"github.com/Kash4299/todo-chat-app/internal/service/chat"
)

type fakeConn struct {
	id uuid.UUID
}

func (f *fakeConn) ID() uuid.UUID { return f.id }
func (f *fakeConn) Close() error  { return nil }

// TestRoomManager_ConcurrentJoinDifferentChannelsIsRaceSafe is the RED test
// for BE-15. It exercises the property that motivates the sharded design:
// many goroutines joining many different channels must not data-race or
// corrupt membership state.
//
// Run with: go test -race ./internal/service/chat/...
//
// What this test proves:
//   - No data races under -race when Join is called from N goroutines
//     across M channels.
//   - Membership integrity: after all Joins return, Count(channel) equals
//     the number of Joins issued to that channel.
//
// What this test does NOT prove:
//   - That the implementation is sharded. A single-mutex implementation
//     also passes this test. Sharding is verified by code review, not here.
//     (Do not "prove" sharding with a timing benchmark — flaky and dishonest.)
func TestRoomManager_ConcurrentJoinDifferentChannelsIsRaceSafe(t *testing.T) {
	t.Parallel()

	const (
		channels       = 64
		connsPerChannel = 16
	)

	mgr := chat.NewRoomManager()

	channelIDs := make([]uuid.UUID, channels)
	for i := range channelIDs {
		channelIDs[i] = uuid.New()
	}

	var wg sync.WaitGroup
	for i := range channels {
		chID := channelIDs[i]
		for range connsPerChannel {
			wg.Go(func() {
				c := &fakeConn{id: uuid.New()}
				if err := mgr.Join(chID, c); err != nil {
					t.Errorf("Join(%s) returned error: %v", chID, err)
				}
			})
		}
	}
	wg.Wait()

	for _, chID := range channelIDs {
		if got := mgr.Count(chID); got != connsPerChannel {
			t.Errorf("Count(%s) = %d, want %d", chID, got, connsPerChannel)
		}
	}
}

// TestRoomManager_JoinNilConnReturnsError pins the contract that nil is not
// a legal Conn. The lifecycle layer must construct a non-nil wrapper before
// calling Join; passing nil is a programming error and must surface as
// ErrNilConn, not a runtime panic inside the manager.
func TestRoomManager_JoinNilConnReturnsError(t *testing.T) {
	t.Parallel()

	mgr := chat.NewRoomManager()
	err := mgr.Join(uuid.New(), nil)

	if !errors.Is(err, chat.ErrNilConn) {
		t.Fatalf("Join(_, nil) returned %v, want %v", err, chat.ErrNilConn)
	}
}

// TestRoomManager_JoinThenListReturnsConn locks the basic happy path: every
// Conn that Joins a channel must be reported by List for that channel. List
// returns a snapshot, so caller order does not matter; we compare by ID set.
func TestRoomManager_JoinThenListReturnsConn(t *testing.T) {
	t.Parallel()

	mgr := chat.NewRoomManager()
	channelID := uuid.New()

	conns := []*fakeConn{
		{id: uuid.New()},
		{id: uuid.New()},
		{id: uuid.New()},
	}
	want := make(map[uuid.UUID]bool, len(conns))
	for _, c := range conns {
		if err := mgr.Join(channelID, c); err != nil {
			t.Fatalf("Join: %v", err)
		}
		want[c.ID()] = true
	}

	got := mgr.List(channelID)
	if len(got) != len(conns) {
		t.Fatalf("List length = %d, want %d", len(got), len(conns))
	}
	for _, c := range got {
		if !want[c.ID()] {
			t.Errorf("List returned unexpected conn %s", c.ID())
		}
		delete(want, c.ID())
	}
	if len(want) > 0 {
		t.Errorf("List missing conns: %v", want)
	}
}

// TestRoomManager_LeaveRemovesConn locks the lifecycle: after Leave, the conn
// must be gone from both List and Count. The channel itself becomes empty
// (an implementation detail — we don't assert internal cleanup of the channel
// entry, only that the public surface reports zero).
func TestRoomManager_LeaveRemovesConn(t *testing.T) {
	t.Parallel()

	mgr := chat.NewRoomManager()
	channelID := uuid.New()
	c := &fakeConn{id: uuid.New()}

	if err := mgr.Join(channelID, c); err != nil {
		t.Fatalf("Join: %v", err)
	}
	if got := mgr.Count(channelID); got != 1 {
		t.Fatalf("Count after Join = %d, want 1", got)
	}

	mgr.Leave(channelID, c.ID())

	if got := mgr.Count(channelID); got != 0 {
		t.Errorf("Count after Leave = %d, want 0", got)
	}
	if got := mgr.List(channelID); len(got) != 0 {
		t.Errorf("List after Leave = %v, want empty", got)
	}
}

// TestRoomManager_LeaveIsIdempotent locks the contract that Leave must never
// panic regardless of input. The lifecycle layer can call Leave from multiple
// cleanup paths (read error, write error, explicit close, double-close races),
// so a missing or already-removed entry must be a no-op.
func TestRoomManager_LeaveIsIdempotent(t *testing.T) {
	t.Parallel()

	mgr := chat.NewRoomManager()

	// Leave on a channel that never existed.
	mgr.Leave(uuid.New(), uuid.New())

	// Leave on an existing channel but a conn that was never joined.
	channelID := uuid.New()
	c := &fakeConn{id: uuid.New()}
	if err := mgr.Join(channelID, c); err != nil {
		t.Fatalf("Join: %v", err)
	}
	mgr.Leave(channelID, uuid.New())
	if got := mgr.Count(channelID); got != 1 {
		t.Errorf("after Leave(unknown conn): Count = %d, want 1", got)
	}

	// Leave the same conn twice.
	mgr.Leave(channelID, c.ID())
	mgr.Leave(channelID, c.ID())
	if got := mgr.Count(channelID); got != 0 {
		t.Errorf("after double Leave: Count = %d, want 0", got)
	}
}
