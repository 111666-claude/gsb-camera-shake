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

func TestPushUsesOverrideParams(t *testing.T) {
	s, err := New(Params{Amplitude: 10, DurationMs: 200}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	s.SetOverride("explosion", Params{Amplitude: 30, DurationMs: 400})
	if got := s.Push("r1", "explosion", 0, 1, 0); got != 30 {
		t.Fatalf("覆写种类的强度应该是 30，得到 %v", got)
	}
	if got := s.Active(200); got != 15 {
		t.Fatalf("覆写种类的时长应该是 400， halfway 强度应该是 15，得到 %v", got)
	}
	if got := s.Push("r2", "hit", 0, 1, 0); got != 10 {
		t.Fatalf("没覆写的种类应该用默认振幅 10，得到 %v", got)
	}
}

func TestPushScalesAmplitude(t *testing.T) {
	s, err := New(Params{Amplitude: 10, DurationMs: 200}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Push("r1", "hit", 0, 2, 0); got != 20 {
		t.Fatalf("强度应该是振幅乘 scale 等于 20，得到 %v", got)
	}
}

func TestActiveClampsToMaxTotal(t *testing.T) {
	s, err := New(Params{Amplitude: 10, DurationMs: 200}, 12, 100)
	if err != nil {
		t.Fatal(err)
	}
	s.Push("r1", "hit", 0, 1, 0)
	s.Push("r2", "hit", 0, 1, 0)
	if got := s.Active(0); got != 12 {
		t.Fatalf("合成强度应该钳到 12，得到 %v", got)
	}
}

func TestActiveDecaysLinearly(t *testing.T) {
	s, err := New(Params{Amplitude: 10, DurationMs: 200}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	s.Push("r1", "hit", 0, 1, 0)
	if got := s.Active(100); got != 5 {
		t.Fatalf("过半时长强度应该衰减到 5，得到 %v", got)
	}
}

func TestPushTruncatesByBudget(t *testing.T) {
	s, err := New(Params{Amplitude: 10, DurationMs: 200}, 100, 15)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Push("r1", "hit", 0, 1, 0); got != 10 {
		t.Fatalf("第一次应该全额生效 10，得到 %v", got)
	}
	if got := s.Push("r2", "hit", 0, 1, 0); got != 5 {
		t.Fatalf("超出预算应该截断到 5，得到 %v", got)
	}
	if got := s.Push("r3", "hit", 0, 1, 0); got != 0 {
		t.Fatalf("额度用完应该返回 0，得到 %v", got)
	}
}

func TestBudgetResetsNextSecond(t *testing.T) {
	s, err := New(Params{Amplitude: 10, DurationMs: 200}, 100, 15)
	if err != nil {
		t.Fatal(err)
	}
	s.Push("r1", "hit", 0, 1, 0)
	s.Push("r2", "hit", 0, 1, 0)
	if got := s.Push("r3", "hit", 0, 1, 1000); got != 10 {
		t.Fatalf("跨秒应该重新计算预算，得到 %v", got)
	}
}

func TestPushSameReqIDOnlyOnce(t *testing.T) {
	s, err := New(Params{Amplitude: 10, DurationMs: 200}, 100, 100)
	if err != nil {
		t.Fatal(err)
	}
	if got := s.Push("same", "hit", 0, 1, 0); got != 10 {
		t.Fatalf("第一次应该生效 10，得到 %v", got)
	}
	if got := s.Push("same", "hit", 0, 1, 0); got != 0 {
		t.Fatalf("重复 reqID 应该返回 0，得到 %v", got)
	}
}

func TestScannedDoesNotGrowWithQueries(t *testing.T) {
	s, err := New(Params{Amplitude: 10, DurationMs: 200000}, 100000000, 100000000)
	if err != nil {
		t.Fatal(err)
	}
	for index := 0; index < 3000; index++ {
		s.Push(string(rune(index)), "hit", 0, 1, 0)
	}
	for index := 0; index < 3000; index++ {
		s.Active(0)
	}
	if got := s.Scanned(); got > 6000 {
		t.Fatalf("Scanned 不应该随查询次数放大，得到 %d", got)
	}
}

func TestReplaySameSequenceGivesSameResults(t *testing.T) {
	run := func() []float64 {
		s, err := New(Params{Amplitude: 10, DurationMs: 200}, 12, 15)
		if err != nil {
			t.Fatal(err)
		}
		s.SetOverride("explosion", Params{Amplitude: 30, DurationMs: 400})
		out := []float64{
			s.Push("a", "explosion", 0, 1, 0),
			s.Push("b", "hit", 0, 1, 0),
			s.Push("a", "hit", 0, 1, 0),
			s.Active(0),
			s.Active(100),
			s.Push("c", "hit", 0, 1, 1000),
			s.Active(1000),
		}
		return out
	}
	first, second := run(), run()
	for index := range first {
		if first[index] != second[index] {
			t.Fatalf("第 %d 步重放结果不一致：%v 对 %v", index, first, second)
		}
	}
}
