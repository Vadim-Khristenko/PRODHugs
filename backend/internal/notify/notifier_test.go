package notify

import (
	"context"
	"testing"

	"go-service-template/internal/models"

	"github.com/google/uuid"
)

type fakeUsers struct{ u *models.User }

func (f *fakeUsers) GetByID(context.Context, uuid.UUID) (*models.User, error) { return f.u, nil }

func strptr(s string) *string { return &s }

func TestNotifyHugSuggestionBuildsButtons(t *testing.T) {
	tg := &fakeProvider{name: "telegram", enabled: true}
	deps := &fakeDeps{addr: Address{TelegramID: tgID(1)}, refs: map[string][]SentRef{}}
	r := NewRouter([]Provider{tg}, deps, deps, deps, nil)
	users := &fakeUsers{u: &models.User{Username: "anna", DisplayName: strptr("Аня")}}
	n := NewNotifier(r, users, nil)

	receiver, giver, hid := uuid.New(), uuid.New(), uuid.New()
	n.NotifyHugSuggestion(context.Background(), receiver, hid, giver, "warm", nil)

	if len(tg.sent) != 1 {
		t.Fatalf("want 1 send, got %d", len(tg.sent))
	}
	btns := tg.sent[0].Buttons
	if len(btns) != 1 || len(btns[0]) != 2 {
		t.Fatalf("want one row of two buttons, got %+v", btns)
	}
	if btns[0][0].Action != "hug.accept:"+hid.String() {
		t.Fatalf("accept action wrong: %q", btns[0][0].Action)
	}
	if btns[0][1].Action != "hug.decline:"+hid.String() {
		t.Fatalf("decline action wrong: %q", btns[0][1].Action)
	}
}

func TestNotifyHugCompletedSendsBoth(t *testing.T) {
	tg := &fakeProvider{name: "telegram", enabled: true}
	deps := &fakeDeps{addr: Address{TelegramID: tgID(1)}, refs: map[string][]SentRef{}}
	r := NewRouter([]Provider{tg}, deps, deps, deps, nil)
	users := &fakeUsers{u: &models.User{Username: "bob"}}
	n := NewNotifier(r, users, nil)
	n.NotifyHugCompleted(context.Background(), uuid.New(), uuid.New(), uuid.New(), "bear", 2, nil)
	if len(tg.sent) != 2 {
		t.Fatalf("completed should message both participants, got %d", len(tg.sent))
	}
}
