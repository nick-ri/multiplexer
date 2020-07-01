package limit

import (
	"testing"
	"time"
)

func Test_limiter_count(t *testing.T) {
	type fields struct {
		rate int64
		curr *window
		prev *window
	}

	tests := []struct {
		name      string
		fields    fields
		wantCount int64 // formula looks count_in_prev_window * (window_size-window_offset)/window_size + count_in_curr_window
		now       func() time.Time
	}{
		{
			name: "empty windows",
			fields: fields{
				rate: time.Second.Nanoseconds(),
				curr: &window{},
				prev: &window{},
			},
			wantCount: 0, // 0 * (1000000000 - (0))/1000000000 + 0 = 0
			now: func() time.Time { // mock time function
				return time.Unix(0, time.Second.Nanoseconds())
			},
		},
		{
			name: "prev window has some values",
			fields: fields{
				rate: time.Second.Nanoseconds(),
				curr: &window{
					s: time.Second.Nanoseconds(),
				},
				prev: &window{
					n: 5,
				},
			},
			wantCount: 5, // 0 * (1000000000 - (0))/1000000000 + 5 = 5
			now: func() time.Time { // mock time function
				return time.Unix(0, time.Second.Nanoseconds())
			},
		},
		{
			name: "both windows has some values",
			fields: fields{
				rate: time.Second.Nanoseconds(),
				curr: &window{
					s: time.Second.Nanoseconds(),
					n: 4,
				},
				prev: &window{
					n: 2,
				},
			},
			wantCount: 6, // 2 * (1000000000 - (0))/1000000000 + 4 = 6
			now: func() time.Time { // mock time function
				return time.Unix(0, time.Second.Nanoseconds())
			},
		},
		{
			name: "overflowed windows",
			fields: fields{
				rate: time.Second.Nanoseconds(),
				curr: &window{
					s: time.Second.Nanoseconds(),
					n: 6,
				},
				prev: &window{
					n: 5,
				},
			},
			wantCount: 11, // 5 * (1000000000 - (0))/1000000000 + 6 = 11
			now: func() time.Time { // mock time function
				return time.Unix(0, time.Second.Nanoseconds())
			},
		},
		{
			name: "current window +100ms",
			fields: fields{
				rate: time.Second.Nanoseconds(),
				curr: &window{
					s: time.Second.Nanoseconds(),
					n: 3,
				},
				prev: &window{
					n: 7,
				},
			},
			wantCount: 9, // 7 * (1000000000 - (1100000000-1000000000))/1000000000 + 3 = 9
			now: func() time.Time { // mock time function
				return time.Unix(0, (time.Second + time.Millisecond*100).Nanoseconds())
			},
		},
		{
			name: "current window +300ms",
			fields: fields{
				rate: time.Second.Nanoseconds(),
				curr: &window{
					s: time.Second.Nanoseconds(),
					n: 2,
				},
				prev: &window{
					n: 10,
				},
			},
			wantCount: 9, // 10 * (1000000000 - (1300000000-1000000000))/1000000000 + 2 = 9
			now: func() time.Time { // mock time function
				return time.Unix(0, (time.Second + time.Millisecond*300).Nanoseconds())
			},
		},
		{
			name: "current window +900ms",
			fields: fields{
				rate: time.Second.Nanoseconds(),
				curr: &window{
					s: time.Second.Nanoseconds(),
					n: 9,
				},
				prev: &window{
					n: 10,
				},
			},
			wantCount: 10, // 10 * (1000000000 - (1900000000-1000000000))/1000000000 + 9 = 10
			now: func() time.Time { // mock time function
				return time.Unix(0, (time.Second + time.Millisecond*900).Nanoseconds())
			},
		},
		{
			name: "current window +1100ms, renewing windows",
			fields: fields{
				rate: time.Second.Nanoseconds(),
				curr: &window{
					s: time.Second.Nanoseconds(),
					n: 10,
				},
				prev: &window{
					n: 10,
				},
			},
			wantCount: 9, // 10 * (1000000000 - (2100000000-1000000000))/1000000000 + 10 = 9
			now: func() time.Time { // mock time function
				return time.Unix(0, (time.Second + time.Millisecond*1100).Nanoseconds())
			},
		},
		{
			name: "current window +1500ms, renewing windows",
			fields: fields{
				rate: time.Second.Nanoseconds(),
				curr: &window{
					s: time.Second.Nanoseconds(),
					n: 10,
				},
				prev: &window{
					n: 10,
				},
			},
			wantCount: 5, // 10 * (1000000000 - (2500000000-1000000000))/1000000000 + 10 = 5
			now: func() time.Time { // mock time function
				return time.Unix(0, (time.Second + time.Millisecond*1500).Nanoseconds())
			},
		},
		{
			name: "current window +1700ms, renewing windows",
			fields: fields{
				rate: time.Second.Nanoseconds(),
				curr: &window{
					s: time.Second.Nanoseconds(),
					n: 10,
				},
				prev: &window{
					n: 10,
				},
			},
			wantCount: 3, // 10 * (1000000000 - (2700000000-1000000000))/1000000000 + 10 = 3
			now: func() time.Time { // mock time function
				return time.Unix(0, (time.Second + time.Millisecond*1700).Nanoseconds())
			},
		},
		{
			name: "long downtime, renewing windows",
			fields: fields{
				rate: time.Second.Nanoseconds(),
				curr: &window{
					s: time.Second.Nanoseconds(),
					n: 10,
				},
				prev: &window{
					n: 10,
				},
			},
			wantCount: 0, // 0 * (1000000000 - (3000000000-3000000000))/1000000000 + 0 = 0
			now: func() time.Time { // mock time function
				return time.Unix(0, (time.Second * 3).Nanoseconds())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now = tt.now
			l := &limiter{
				rate: tt.fields.rate,
				curr: tt.fields.curr,
				prev: tt.fields.prev,
			}
			if got := l.count(); got != tt.wantCount {
				t.Errorf("count() = %v, want %v", got, tt.wantCount)
			}
		})
	}
}
