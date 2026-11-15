package domain

import (
	"testing"

	"github.com/google/uuid"

	"sunshine/internal/screening/contract"
	"sunshine/internal/screening/scorer"
)

func TestIsVietnamese(t *testing.T) {
	cases := map[string]bool{
		"Hồ bơi cuối tuần mở mấy giờ vậy?":                true,
		"ho boi cuoi tuan mo may gio vay moi nguoi oi":    true, // không dấu vẫn là tiếng Việt
		"This is what you have to do with the car.":       false,
		"cho minh hoi thang may tang 8 co sua chua khong": true,
		"OK": true, // quá ngắn để kết luận
	}
	for text, want := range cases {
		if got := IsVietnamese(text); got != want {
			t.Errorf("IsVietnamese(%q) = %v, want %v", text, got, want)
		}
	}
}

func TestBlocklistMatchesWithoutDiacritics(t *testing.T) {
	snap := Snapshot{Block: []BlockEntry{{Term: "vay nóng không thế chấp", Rule: 6}}}
	b := snap.MatchBlock("Ở đây có VAY NONG khong the chap!")
	if b == nil || b.Rule != 6 {
		t.Fatalf("MatchBlock = %+v, want rule 6", b)
	}
	if snap.MatchBlock("vay tiền mua nhà") != nil {
		t.Error("partial phrase must not match")
	}
}

func TestDecide(t *testing.T) {
	s := Settings{HoldThreshold: 0.5, AutoDeleteThreshold: 0.95, OutagePolicy: PolicyFailClosed}
	res := func(off, hate float64) scorer.Result {
		return scorer.Result{ModelVersion: "v", Labels: map[string]float64{"offensive": off, "hate": hate}}
	}
	cases := []struct {
		name       string
		r          scorer.Result
		autoDelete bool
		outcome    string
		label      string
		rule       *int
	}{
		{"clean", res(0.1, 0.2), false, contract.OutcomePublished, "clean", nil},
		{"hate held, no rule", res(0.1, 0.8), false, contract.OutcomePending, "hate", nil},
		{"offensive held as rule 4", res(0.7, 0.1), false, contract.OutcomePending, "offensive", ptr(4)},
		{"hate wins over offensive", res(0.9, 0.6), false, contract.OutcomePending, "hate", nil},
		{"auto-delete off by default", res(0.97, 0.1), false, contract.OutcomePending, "offensive", ptr(4)},
		{"auto-delete when enabled", res(0.97, 0.1), true, contract.OutcomeAutoDeleted, "offensive", ptr(4)},
		{"hate never auto-deleted", res(0.1, 0.99), true, contract.OutcomePending, "hate", nil},
	}
	for _, c := range cases {
		s.AutoDeleteEnabled = c.autoDelete
		got := Decide(c.r, s, s.HoldThreshold, s.AutoDeleteThreshold)
		if got.Outcome != c.outcome || got.Label != c.label || (got.Rule == nil) != (c.rule == nil) ||
			(got.Rule != nil && *got.Rule != *c.rule) {
			t.Errorf("%s: got %s/%s/%v", c.name, got.Outcome, got.Label, got.Rule)
		}
	}
}

func TestChannelThresholdOverrides(t *testing.T) {
	ch := uuid.New()
	snap := Snapshot{Settings: Settings{HoldThreshold: 0.5}, Channels: map[uuid.UUID]ChannelThreshold{ch: {Hold: 0.3}}}
	if snap.HoldFor(ch) != 0.3 || snap.HoldFor(uuid.New()) != 0.5 {
		t.Error("channel threshold should override the global one")
	}
}

func TestSettingsValidate(t *testing.T) {
	ok := Settings{HoldThreshold: 0.5, AutoDeleteThreshold: 0.9, OutagePolicy: PolicyFailOpen}
	if err := ok.Validate(); err != nil {
		t.Fatal(err)
	}
	bad := ok
	bad.AutoDeleteThreshold = 0.4
	if bad.Validate() == nil {
		t.Error("auto-delete below hold must fail")
	}
}
