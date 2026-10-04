// Package shake 是震屏合成：按种类覆写默认参数、多来源叠加并钳制、按剩余时间线性衰减、
// 每秒强度预算与请求幂等。
//
// 已生效的来源分两段存放：pending 按起始时间排序、live 按结束时间排序。
// Active 只把到期的来源在两段之间搬运或移除，并维护线性衰减的三个聚合量，
// 合成强度 O(1) 得出，因此每条来源至多被扫描两次（激活、到期各一次），
// 扫描代价不随历史请求数或查询次数放大。
package shake

import (
	"container/heap"
	"errors"
)

// ErrBadParams 震屏参数不合法。
var ErrBadParams = errors.New("震屏参数不合法")

// Params 是一组震屏参数。
type Params struct {
	Amplitude  float64
	DurationMs int
}

// source 是一个已经生效的震屏来源，amplitude 是预算截断后实际生效的强度。
type source struct {
	kind      string
	amplitude float64
	startAt   int
	duration  int
}

// Shaker 是震屏合成器。
type Shaker struct {
	base      Params
	overrides map[string]Params
	maxTotal  float64
	budget    float64
	spent     map[int]float64
	seen      map[string]bool

	pending startHeap
	live    endHeap

	// 存活来源的线性衰减聚合量：
	// sumAmp   = Σ amplitude
	// sumSlope = Σ amplitude / duration
	// sumShift = Σ amplitude * startAt / duration
	// 时刻 now 的原始合成强度 = sumAmp + sumShift - now*sumSlope
	sumAmp   float64
	sumSlope float64
	sumShift float64

	scanned int
}

// New 建震屏合成器：base 是默认参数，maxTotal 是合成立场上限，budgetPerSec 是每秒强度预算。
func New(base Params, maxTotal, budgetPerSec float64) (*Shaker, error) {
	if base.Amplitude <= 0 || base.DurationMs <= 0 || maxTotal <= 0 || budgetPerSec <= 0 {
		return nil, ErrBadParams
	}
	return &Shaker{
		base:      base,
		overrides: map[string]Params{},
		maxTotal:  maxTotal,
		budget:    budgetPerSec,
		spent:     map[int]float64{},
		seen:      map[string]bool{},
	}, nil
}

// SetOverride 给某个种类配上覆写参数。
func (s *Shaker) SetOverride(kind string, params Params) {
	s.overrides[kind] = params
}

// Push 推入一次震屏请求，返回实际生效的强度（0 表示这次请求没有生效）。
func (s *Shaker) Push(reqID, kind string, priority int, scale float64, atMs int) float64 {
	if scale <= 0 {
		return 0
	}
	if s.seen[reqID] {
		return 0
	}
	s.seen[reqID] = true

	params, ok := s.overrides[kind]
	if !ok {
		params = s.base
	}
	if params.Amplitude <= 0 || params.DurationMs <= 0 {
		return 0
	}

	want := params.Amplitude * scale
	bucket := atMs / 1000
	room := s.budget - s.spent[bucket]
	if room <= 0 {
		return 0
	}
	granted := want
	if granted > room {
		granted = room
	}
	s.spent[bucket] += granted

	heap.Push(&s.pending, source{
		kind:      kind,
		amplitude: granted,
		startAt:   atMs,
		duration:  params.DurationMs,
	})
	return granted
}

// Active 返回 nowMs 时刻的合成震屏强度。
func (s *Shaker) Active(nowMs int) float64 {
	for s.pending.Len() > 0 && s.pending[0].startAt <= nowMs {
		item := heap.Pop(&s.pending).(source)
		s.scanned++
		s.addAggregate(item)
		heap.Push(&s.live, item)
	}
	for s.live.Len() > 0 && s.live[0].startAt+s.live[0].duration <= nowMs {
		item := heap.Pop(&s.live).(source)
		s.scanned++
		s.removeAggregate(item)
	}
	total := s.sumAmp + s.sumShift - float64(nowMs)*s.sumSlope
	if total < 0 {
		total = 0
	}
	if total > s.maxTotal {
		total = s.maxTotal
	}
	return total
}

// Scanned 是累计扫描的来源数（规模观测）。
func (s *Shaker) Scanned() int { return s.scanned }

func (s *Shaker) addAggregate(item source) {
	d := float64(item.duration)
	s.sumAmp += item.amplitude
	s.sumSlope += item.amplitude / d
	s.sumShift += item.amplitude * float64(item.startAt) / d
}

func (s *Shaker) removeAggregate(item source) {
	d := float64(item.duration)
	s.sumAmp -= item.amplitude
	s.sumSlope -= item.amplitude / d
	s.sumShift -= item.amplitude * float64(item.startAt) / d
}

// startHeap 按起始时间排序，存放尚未到生效时刻的来源。
type startHeap []source

func (h startHeap) Len() int { return len(h) }
func (h startHeap) Less(i, j int) bool {
	return h[i].startAt < h[j].startAt
}
func (h startHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *startHeap) Push(x any)   { *h = append(*h, x.(source)) }
func (h *startHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}

// endHeap 按结束时间排序，存放当前存活的来源。
type endHeap []source

func (h endHeap) Len() int { return len(h) }
func (h endHeap) Less(i, j int) bool {
	return h[i].startAt+h[i].duration < h[j].startAt+h[j].duration
}
func (h endHeap) Swap(i, j int) { h[i], h[j] = h[j], h[i] }
func (h *endHeap) Push(x any)   { *h = append(*h, x.(source)) }
func (h *endHeap) Pop() any {
	old := *h
	n := len(old)
	item := old[n-1]
	*h = old[:n-1]
	return item
}
