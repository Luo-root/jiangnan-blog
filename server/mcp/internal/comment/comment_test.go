package comment

import (
	"strings"
	"testing"
)

func TestNewRejectsBlankBody(t *testing.T) {
	if _, err := New("human", "jiangnan", Input{Body: "   "}); err == nil {
		t.Fatal("blank body should fail")
	}
}

func TestNewIDAndSlice(t *testing.T) {
	c, err := New("agent", "bot", Input{Body: "looks stale", ReplyTo: "cmt_old"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(c.ID, "cmt_") {
		t.Fatalf("id = %s", c.ID)
	}
	if c.AuthorType != "agent" || c.Author != "bot" || c.ReplyTo != "cmt_old" {
		t.Fatalf("%+v", c)
	}
	if got := Slice(nil); got == nil || len(got) != 0 {
		t.Fatalf("nil slice = %#v", got)
	}
}
