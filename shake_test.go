package shake

import "testing"

func TestNewRejectsBadParams(t *testing.T) {
	if _, err := New(Params{Amplitude: 0, DurationMs: 200}, 12, 15); err != ErrBadParams {
		t.Fatal("振幅不合法应该报 ErrBadParams")
	}
}

func TestPushRejectsNonPositiveScale(t *testing.T) {
	s, err := New(Params{Amplitude: 10, DurationMs: 200}, 12, 15)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Push("r1", "hit", 0, 0, 0); got != 0 {
		t.Fatalf("scale 不大于 0 的请求应该返回 0，得到 %v", got)
	}
}

func TestActiveBeforeAnyPush(t *testing.T) {
	s, err := New(Params{Amplitude: 10, DurationMs: 200}, 12, 15)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Active(0); got != 0 {
		t.Fatalf("没有来源时合成强度应该是 0，得到 %v", got)
	}
}

func TestActiveAfterDurationIsZero(t *testing.T) {
	s, err := New(Params{Amplitude: 10, DurationMs: 200}, 12, 15)
	if err != nil {
		t.Fatal(err)
	}
	s.Push("r1", "hit", 0, 1, 0)
	if got := s.Active(500); got != 0 {
		t.Fatalf("超时之后合成强度应该是 0，得到 %v", got)
	}
}

func TestScannedStartsAtZero(t *testing.T) {
	s, err := New(Params{Amplitude: 10, DurationMs: 200}, 12, 15)
	if err != nil {
		t.Fatal(err)
	}
	if s.Scanned() != 0 {
		t.Fatal("Scanned 初始应该是 0")
	}
}
