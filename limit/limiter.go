package limit

import (
	"sync/atomic"
	"time"
)

var now = time.Now

type window struct {
	s int64
	n int64
}

func (w *window) start() int64 {
	return atomic.LoadInt64(&w.s)
}

func (w *window) num() int64 {
	return atomic.LoadInt64(&w.n)
}

func (w *window) incr(n int64) int64 {
	return atomic.AddInt64(&w.n, n)
}

func (w *window) set(start, num int64) {
	atomic.StoreInt64(&w.s, start)
	atomic.StoreInt64(&w.n, num)
}

type limiter struct {
	rate  int64
	limit int64
	curr  *window
	prev  *window
}

func NewLimiter(rate time.Duration, limit int) Limiter {
	return &limiter{
		rate:  rate.Nanoseconds(), // window size
		limit: int64(limit),       // limit per slide windows
		curr:  &window{},          // current window data
		prev:  &window{},          // previous window data
	}
}

func (l *limiter) renew(now time.Time) {

	// prepare the start of new active window
	currNS := now.Truncate(time.Duration(l.rate)).UnixNano()

	// calculate difference between active window and current window
	diff := (currNS - l.curr.start()) / l.rate

	// if active window collides with current window we need to update previous
	if diff >= 1 {
		old := int64(0)

		// after big downtime diff will be a big number
		// but if we had direct transition we need to store current count to the prev window on update
		// otherwise just set zero
		if diff == 1 {
			old = l.curr.num()
		}

		// reset windows, set old one as 1x rate ago and new one right now with new counts
		l.prev.set(currNS-l.rate, old)
		l.curr.set(currNS, 0)
	}
}

func (l *limiter) count() int64 {
	now := now()

	// try to renew windows by dynamically move it forward and swap values
	l.renew(now)

	// formula of count approx. for the slided window:
	// count_in_prev_window * (window_size-window_offset)/window_size + count_in_curr_window

	offset := now.UnixNano() - l.curr.start()

	weight := float64(l.rate-offset) / float64(l.rate)

	return int64(weight*float64(l.prev.num())) + l.curr.num()
}

func (l *limiter) Allow() bool {
	if l.count() >= l.limit {
		return false
	}

	l.curr.incr(1)

	return true
}
