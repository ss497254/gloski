package actions

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/ss497254/gloski/internal/database"
)

func TestStoreRecordRecentAndUndo(t *testing.T) {
	ctx := context.Background()
	db, err := database.Open(filepath.Join(t.TempDir(), "master.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	store := NewStore(db)

	first := Action{DeviceID: "dev", Kind: KindAgentStart, Target: "agent", From: "stopped", To: "running", Result: ResultOK, Undoable: true}
	if err := store.Record(ctx, first); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)
	second := Action{DeviceID: "dev", Kind: KindTrustHost, Target: "agent", Result: ResultOK}
	if err := store.Record(ctx, second); err != nil {
		t.Fatal(err)
	}
	time.Sleep(2 * time.Millisecond)
	other := Action{DeviceID: "other", Kind: KindAgentStop, Result: ResultOK, Undoable: true}
	if err := store.Record(ctx, other); err != nil {
		t.Fatal(err)
	}

	recent, err := store.Recent(ctx, "dev", 10)
	if err != nil || len(recent) != 2 {
		t.Fatalf("Recent() = %+v, %v; want two actions", recent, err)
	}
	if recent[0].Kind != KindTrustHost || recent[0].Undoable {
		t.Fatalf("newest action = %+v, want non-undoable trust_host", recent[0])
	}
	if recent[1].Kind != KindAgentStart || !recent[1].Undoable || recent[1].From != "stopped" {
		t.Fatalf("older action = %+v", recent[1])
	}
	if recent[0].ID == "" || recent[0].CreatedAt.IsZero() {
		t.Fatalf("Record() did not fill ID and timestamp: %+v", recent[0])
	}

	saved, err := store.Get(ctx, recent[1].ID)
	if err != nil || saved.DeviceID != "dev" {
		t.Fatalf("Get() = %+v, %v", saved, err)
	}
	claimed, err := store.ClaimUndo(ctx, saved.ID)
	if err != nil || !claimed {
		t.Fatalf("ClaimUndo() = %v, %v; want true", claimed, err)
	}
	claimed, err = store.ClaimUndo(ctx, saved.ID)
	if err != nil || claimed {
		t.Fatalf("second ClaimUndo() = %v, %v; want false", claimed, err)
	}
	saved, err = store.Get(ctx, saved.ID)
	if err != nil || saved.UndoneAt == nil {
		t.Fatalf("ClaimUndo() left %+v, %v", saved, err)
	}
	if err := store.ReleaseUndo(ctx, saved.ID); err != nil {
		t.Fatal(err)
	}
	claimed, err = store.ClaimUndo(ctx, saved.ID)
	if err != nil || !claimed {
		t.Fatalf("ClaimUndo() after release = %v, %v; want true", claimed, err)
	}
}
