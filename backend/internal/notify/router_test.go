package notify

import (
	"context"
	"testing"

	"go-service-template/internal/notify/richtext"

	"github.com/google/uuid"
)

type fakeProvider struct {
	name     string
	enabled  bool
	sent     []Message
	chatRefs []string
	edited   []SentRef
	failWith error
}

func (f *fakeProvider) Name() string  { return f.name }
func (f *fakeProvider) Enabled() bool { return f.enabled }
func (f *fakeProvider) Send(_ context.Context, chatRef string, msg Message) (SentRef, error) {
	if f.failWith != nil {
		return SentRef{}, f.failWith
	}
	f.sent = append(f.sent, msg)
	f.chatRefs = append(f.chatRefs, chatRef)
	return SentRef{Provider: f.name, ChatRef: chatRef, MessageRef: "m1"}, nil
}
func (f *fakeProvider) Edit(_ context.Context, ref SentRef, _ Message) error {
	f.edited = append(f.edited, ref)
	return nil
}

type fakeDeps struct {
	addr    Address
	refs    map[string][]SentRef
	blocked []uuid.UUID
}

func (d *fakeDeps) ResolveAddress(context.Context, uuid.UUID) (Address, error) { return d.addr, nil }
func (d *fakeDeps) SaveRef(_ context.Context, kind string, id uuid.UUID, r SentRef) error {
	d.refs[kind+id.String()] = append(d.refs[kind+id.String()], r)
	return nil
}
func (d *fakeDeps) GetRefs(_ context.Context, kind string, id uuid.UUID) ([]SentRef, error) {
	return d.refs[kind+id.String()], nil
}
func (d *fakeDeps) GetRefByMessage(_ context.Context, _, _ string) (string, uuid.UUID, bool, error) {
	return "", uuid.Nil, false, nil
}
func (d *fakeDeps) MarkTelegramBlocked(_ context.Context, u uuid.UUID) error {
	d.blocked = append(d.blocked, u)
	return nil
}

func tgID(v int64) *int64 { return &v }

func TestDispatchFansOutToEnabledOnly(t *testing.T) {
	tg := &fakeProvider{name: "telegram", enabled: true}
	mx := &fakeProvider{name: "matrix", enabled: false}
	deps := &fakeDeps{addr: Address{TelegramID: tgID(42)}, refs: map[string][]SentRef{}}
	r := NewRouter([]Provider{tg, mx}, deps, deps, deps, nil)

	uid, hid := uuid.New(), uuid.New()
	msg := Message{Body: New().Text("hi").Build()}
	r.Dispatch(context.Background(), uid, "hug_suggestion", hid, msg)

	if len(tg.sent) != 1 || tg.chatRefs[0] != "42" {
		t.Fatalf("telegram not sent to chat 42: %+v", tg.chatRefs)
	}
	if len(mx.sent) != 0 {
		t.Fatal("disabled matrix should not send")
	}
	if got := deps.refs["hug_suggestion"+hid.String()]; len(got) != 1 {
		t.Fatalf("ref not recorded: %+v", got)
	}
}

func TestDispatchSkipsProviderWithoutAddress(t *testing.T) {
	tg := &fakeProvider{name: "telegram", enabled: true}
	deps := &fakeDeps{addr: Address{}, refs: map[string][]SentRef{}}
	r := NewRouter([]Provider{tg}, deps, deps, deps, nil)
	r.Dispatch(context.Background(), uuid.New(), "k", uuid.New(), Message{Body: New().Text("x").Build()})
	if len(tg.sent) != 0 {
		t.Fatal("should skip user with no telegram id")
	}
}

func TestDispatchMarksBlocked(t *testing.T) {
	tg := &fakeProvider{name: "telegram", enabled: true, failWith: Blocked(context.DeadlineExceeded)}
	deps := &fakeDeps{addr: Address{TelegramID: tgID(7)}, refs: map[string][]SentRef{}}
	r := NewRouter([]Provider{tg}, deps, deps, deps, nil)
	uid := uuid.New()
	r.Dispatch(context.Background(), uid, "k", uuid.New(), Message{Body: New().Text("x").Build()})
	if len(deps.blocked) != 1 || deps.blocked[0] != uid {
		t.Fatalf("blocked user not marked: %+v", deps.blocked)
	}
}

func TestEditByEventEditsRecordedRefs(t *testing.T) {
	tg := &fakeProvider{name: "telegram", enabled: true}
	hid := uuid.New()
	deps := &fakeDeps{
		addr: Address{TelegramID: tgID(1)},
		refs: map[string][]SentRef{"hug_suggestion" + hid.String(): {{Provider: "telegram", ChatRef: "1", MessageRef: "m1"}}},
	}
	r := NewRouter([]Provider{tg}, deps, deps, deps, nil)
	r.EditByEvent(context.Background(), "hug_suggestion", hid, Message{Body: New().Text("done").Build()})
	if len(tg.edited) != 1 || tg.edited[0].MessageRef != "m1" {
		t.Fatalf("edit not applied: %+v", tg.edited)
	}
}

var _ = richtext.Doc{}
