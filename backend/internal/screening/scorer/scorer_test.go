package scorer

import (
	"context"
	"testing"
)

func TestStandInScores(t *testing.T) {
	cases := []struct {
		text string
		want string // nhãn vượt 0.5, "" nếu sạch
	}{
		{"Hồ bơi cuối tuần mở mấy giờ vậy mọi người?", ""},
		{"Ai đỗ xe chắn lối là đồ khốn ngu.", "hate"},
		{"Đm cái thang máy lại hỏng.", "offensive"},
		{"Mọi người ơi. Thằng đó đúng là súc vật!", "hate"},
	}
	for _, c := range cases {
		res, err := StandIn{}.Score(context.Background(), c.text)
		if err != nil {
			t.Fatal(err)
		}
		got := ""
		for _, l := range []string{"hate", "offensive"} {
			if res.Labels[l] >= 0.5 && got == "" {
				got = l
			}
		}
		if got != c.want {
			t.Errorf("%q: got %q (%v), want %q", c.text, got, res.Labels, c.want)
		}
	}
}

func TestStandInSentenceOffsetsAreRunes(t *testing.T) {
	res, _ := StandIn{}.Score(context.Background(), "Chào cả nhà. Đồ khốn!")
	if len(res.Sentences) != 1 || res.Sentences[0].Start != 12 {
		t.Fatalf("sentences = %+v, want one starting at rune 12", res.Sentences)
	}
}

func TestStandInOffline(t *testing.T) {
	s := StandIn{Offline: func(context.Context) bool { return true }}
	if _, err := s.Score(context.Background(), "chào"); err != ErrOffline {
		t.Fatalf("err = %v, want ErrOffline", err)
	}
}
