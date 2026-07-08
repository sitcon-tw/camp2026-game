package main

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"github.com/sitcon-tw/camp2026-game/tgbot/internal/telegram"
)

type fakeProfileGetter struct {
	users []telegram.User
	errs  []error
	calls int
}

func (f *fakeProfileGetter) GetMe(context.Context) (telegram.User, error) {
	call := f.calls
	f.calls++
	if call < len(f.errs) && f.errs[call] != nil {
		return telegram.User{}, f.errs[call]
	}
	if call < len(f.users) {
		return f.users[call], nil
	}
	return telegram.User{}, nil
}

func TestGetTelegramBotProfileRetriesTransientError(t *testing.T) {
	getter := &fakeProfileGetter{
		users: []telegram.User{
			{},
			{Username: "camp_bot"},
		},
		errs: []error{
			errors.New("telegram getMe failed: Bad Gateway"),
			nil,
		},
	}

	got, err := getTelegramBotProfile(context.Background(), getter, testLogger())
	if err != nil {
		t.Fatalf("expected retry to succeed, got %v", err)
	}
	if got.Username != "camp_bot" {
		t.Fatalf("unexpected bot profile: %#v", got)
	}
	if getter.calls != 2 {
		t.Fatalf("expected 2 getMe calls, got %d", getter.calls)
	}
}

func TestGetTelegramBotProfileRejectsPermanentError(t *testing.T) {
	getter := &fakeProfileGetter{
		errs: []error{errors.New("telegram getMe failed: Unauthorized")},
	}

	_, err := getTelegramBotProfile(context.Background(), getter, testLogger())
	if err == nil {
		t.Fatal("expected permanent error")
	}
	if getter.calls != 1 {
		t.Fatalf("expected 1 getMe call, got %d", getter.calls)
	}
}

func testLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
