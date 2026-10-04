// Package shake 是震屏合成：按种类覆写默认参数、多来源叠加并钳制、按剩余时间线性衰减、
// 每秒强度预算与请求幂等，只用 Go 标准库。
//
// Active 用两个堆（按开始时间、按结束时间）加运行聚合值维护当前活跃来源，
// 单次查询均摊 O(log n)，不随历史请求数放大。
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

// source 是一个已经生效的震屏来源。
type source struct {
	amplitude float64
	startAt   int
	duration  int
}

// end 是来源的结束时刻（不含）。
func (item source) end() int { return item.startAt + item.duration }

// rate 是来源的线性衰减速率：amplitude/duration。
func (item source) rate() float64 { return item.amplitude / float64(item.duration) }

// sourceHeap 是按 less 排序的来源最小堆。
type sourceHeap struct {
	items []source
	less  func(a, b source) bool
}

func (h sourceHeap) Len() int           { return len(h.items) }
func (h sourceHeap) Less(i, j int) bool { return h.less(h.items[i], h.items[j]) }
func (h sourceHeap) Swap(i, j int)      { h.items[i], h.items[j] = h.items[j], h.items[i] }
func (h *sourceHeap) Push(x any)        { h.items = append(h.items, x.(source)) }
func (h *sourceHeap) Pop() any {
	old := h.items
	n := len(old)
	item := old[n-1]
	h.items = old[:n-1]
	return item
}

// Shaker 是震屏合成器。
type Shaker struct {
	base      Params
	overrides map[string]Params
	maxTotal  float64
	budget    float64
	spent     map[int]float64
	seen      map[string]bool
	pending   *sourceHeap // 未到开始时刻的来源，按 startAt 排序
	active    *sourceHeap // 已生效未结束的来源，按 end 排序
	sumRate   float64     // 活跃来源的 Σ amplitude/duration
	sumBase   float64     // 活跃来源的 Σ amplitude*end/duration
	scanned   int
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
		pending:   &sourceHeap{less: func(a, b source) bool { return a.startAt < b.startAt }},
		active:    &sourceHeap{less: func(a, b source) bool { return a.end() < b.end() }},
	}, nil
}

// SetOverride 给某个种类配上覆写参数；参数不合法（振幅或时长不大于 0）时忽略。
func (s *Shaker) SetOverride(kind string, params Params) {
	if params.Amplitude <= 0 || params.DurationMs <= 0 {
		return
	}
	s.overrides[kind] = params
}

// Push 推入一次震屏请求，返回实际生效的强度（0 表示这次请求没有生效）。
func (s *Shaker) Push(reqID, kind string, priority int, scale float64, atMs int) float64 {
	_ = priority
	if scale <= 0 {
		return 0
	}
	if s.seen[reqID] {
		return 0
	}
	s.seen[reqID] = true
	params := s.base
	if override, ok := s.overrides[kind]; ok {
		params = override
	}
	amplitude := params.Amplitude * scale
	second := atMs / 1000
	effective := amplitude
	if remaining := s.budget - s.spent[second]; effective > remaining {
		effective = remaining
	}
	if effective <= 0 {
		return 0
	}
	s.spent[second] += effective
	heap.Push(s.pending, source{amplitude: effective, startAt: atMs, duration: params.DurationMs})
	return effective
}

// Active 返回 nowMs 时刻的合成震屏强度。
func (s *Shaker) Active(nowMs int) float64 {
	for len(s.pending.items) > 0 && s.pending.items[0].startAt <= nowMs {
		item := heap.Pop(s.pending).(source)
		s.scanned++
		s.sumRate += item.rate()
		s.sumBase += item.rate() * float64(item.end())
		heap.Push(s.active, item)
	}
	for len(s.active.items) > 0 && s.active.items[0].end() <= nowMs {
		item := heap.Pop(s.active).(source)
		s.scanned++
		s.sumRate -= item.rate()
		s.sumBase -= item.rate() * float64(item.end())
	}
	total := s.sumBase - float64(nowMs)*s.sumRate
	if total < 0 {
		return 0
	}
	if total > s.maxTotal {
		return s.maxTotal
	}
	return total
}

// Scanned 是累计扫描的来源数（规模观测）。
func (s *Shaker) Scanned() int { return s.scanned }
