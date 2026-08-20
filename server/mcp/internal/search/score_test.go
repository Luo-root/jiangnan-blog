package search

import (
	"math"
	"testing"
	"time"
)

func TestTokenize(t *testing.T) {
	got := Tokenize("Hello，kubernetes 调度")
	if len(got) != 3 {
		t.Fatalf("%v", got)
	}
}

func TestAccessNeverVisited(t *testing.T) {
	if Access(0, time.Time{}, time.Now(), 7, 1) != 0 {
		t.Fatal("unvisited must be 0")
	}
}

func TestAccessDecay(t *testing.T) {
	now := time.Now()
	fresh := Access(10, now, now, 7, 1)
	old := Access(10, now.Add(-7*24*time.Hour), now, 7, 1)
	want := 10 / math.E // SCHEMA §25: count * exp(-days / half_life)
	if math.Abs(old-want) > 0.01 {
		t.Fatalf("half-life score = %v want %v", old, want)
	}
	if fresh <= old {
		t.Fatalf("fresh=%v old=%v", fresh, old)
	}
}
