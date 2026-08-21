package inbox

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Luo-root/jiangnan-blog/mcp/internal/comment"
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
	if err := s.Update(doneID, StatusDone, "", "", nil, nil); err != nil {
		t.Fatal(err)
	}
	if err := s.Update(abandonedID, StatusAbandoned, "", "", nil, nil); err != nil {
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

func TestCommentAppendDoesNotChangeStatus(t *testing.T) {
	s, err := New(t.TempDir(), 7)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)

	id, err := s.Append("bot", "do this", "todo", []string{"ux"})
	if err != nil {
		t.Fatal(err)
	}
	c, err := comment.New("human", "jiangnan", comment.Input{Body: "looks stale"})
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Update(id, "", "", "", nil, &c); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get(id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Status != StatusPending {
		t.Fatalf("status = %s", got.Status)
	}
	if len(got.Comments) != 1 || got.Comments[0].Body != "looks stale" {
		t.Fatalf("comments = %+v", got.Comments)
	}
	list, err := s.List()
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].CommentCount != 1 {
		t.Fatalf("list = %+v", list)
	}
}

func TestStatusMachineIrreversible(t *testing.T) {
	s, err := New(t.TempDir(), 7)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(s.Close)
	id, err := s.Append("bot", "x", "x", nil)
	if err != nil {
		t.Fatal(err)
	}
	if err := s.Update(id, StatusReviewing, "", "", nil, nil); err != nil {
		t.Fatal(err)
	}
	err = s.Update(id, StatusPending, "", "", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "invalid status transition") {
		t.Fatalf("reviewing → pending should fail, got %v", err)
	}
	if err := s.Update(id, StatusDone, "", "", nil, nil); err != nil {
		t.Fatal(err)
	}
	err = s.Update(id, StatusReviewing, "", "", nil, nil)
	if err == nil || !strings.Contains(err.Error(), "invalid status transition") {
		t.Fatalf("done → reviewing should fail, got %v", err)
	}
	if err := s.Update(id, "", "fixed", "new title", []string{"a"}, nil); err != nil {
		t.Fatal(err)
	}
	got, _ := s.Get(id)
	if got.Status != StatusDone || got.Content != "fixed" || got.Title != "new title" {
		t.Fatalf("%+v", got)
	}
}
