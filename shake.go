// Package shake 是震屏合成：按种类覆写默认参数、多来源矢量叠加并钳制、按剩余时间衰减、
// 每秒强度预算与请求幂等。
// 缺陷：种类覆写没生效、合并不钳制、衰减没做、每秒预算没扣、请求幂等没做、
// 每次查询扫全部历史来源。
package shake

import "errors"

// ErrBadParams 震屏参数不合法。
var ErrBadParams = errors.New("震屏参数不合法")

// Params 是一组震屏参数。
type Params struct {
	Amplitude  float64
	DurationMs int
}

// source 是一个已经生效的震屏来源。
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
	active    []source
	seen      map[string]bool
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
	amplitude := s.base.Amplitude * scale
	s.active = append(s.active, source{
		kind:      kind,
		amplitude: amplitude,
		startAt:   atMs,
		duration:  s.base.DurationMs,
	})
	s.spent[atMs/1000] += amplitude
	return amplitude
}

// Active 返回 nowMs 时刻的合成震屏强度。
func (s *Shaker) Active(nowMs int) float64 {
	total := 0.0
	for _, item := range s.active {
		s.scanned++
		if nowMs < item.startAt || nowMs >= item.startAt+item.duration {
			continue
		}
		total += item.amplitude
	}
	return total
}

// Scanned 是累计扫描的来源数（规模观测）。
func (s *Shaker) Scanned() int { return s.scanned }
