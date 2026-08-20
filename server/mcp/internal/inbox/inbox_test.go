package inbox

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRetentionDeletesDoneAndAbandoned(t *testing.T) {
	dir := t.TempDir()
	s, err := New(dir, 1)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)

	keepID, err := s.Append("bot", "keep me", "keep", nil)
	if err != nil {
		t.Fatal(err)
	}
	doneID, err := s.Append("bot", "old done", "done", nil)
	if err != nil {
		t.Fatal(err)
	}
	abandonedID, err := s.Append("bot", "old abandoned", "abandoned", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Update(doneID, StatusDone, "", "", nil); err != nil {
		t.Fatal(err)
	}
	if err := s.Update(abandonedID, StatusAbandoned, "", "", nil); err != nil {
		t.Fatal(err)
	}

	old := time.Now().Add(-48 * time.Hour)
	for _, id := range []string{doneID, abandonedID} {
		item, err := s.Get(id)
		if err != nil {
			t.Fatal(err)
		}
		item.UpdatedAt = old
		if err := writeItem(filepath.Join(dir, id+".md"), *item); err != nil {
			t.Fatal(err)
		}
	}

	s.cleanup()
	if _, err := os.Stat(filepath.Join(dir, keepID+".md")); err != nil {
		t.Fatalf("pending should remain: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, doneID+".md")); !os.IsNotExist(err) {
		t.Fatalf("done should be deleted, err=%v", err)
	}
	if _, err := os.Stat(filepath.Join(dir, abandonedID+".md")); !os.IsNotExist(err) {
		t.Fatalf("abandoned should be deleted, err=%v", err)
	}
}
