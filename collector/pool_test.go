package collector

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"reflect"
	"testing"
	"time"
)

func Test_collector_Collect(t *testing.T) {
	ts := httptest.NewServer(nil)
	defer ts.Close()

	type fields struct {
		fixed    int
		overflow int
		spawned  int
	}
	type args struct {
		ctx   context.Context
		urls  []string
		limit int
	}
	tests := []struct {
		name         string
		fields       fields
		args         args
		handler      http.HandlerFunc
		want         []res
		wantErr      bool
		cancelBefore func(cancelFunc context.CancelFunc)
	}{
		{
			name: "simple few requests",
			fields: fields{
				fixed: 1,
			},
			args: args{
				ctx:   context.Background(),
				urls:  makeUrls(ts, 5),
				limit: 1,
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, "some_text")
			},
			want: makeRes(ts, "some_text", 5),
		},
		{
			name: "too long response",
			fields: fields{
				fixed: 1,
			},
			args: args{
				ctx:   context.Background(),
				urls:  makeUrls(ts, 5),
				limit: 1,
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(time.Second)
				fmt.Fprint(w, "some_text")
			},
			wantErr: true,
		},
		{
			name: "multiple workers",
			fields: fields{
				fixed: 2,
			},
			args: args{
				ctx:   context.Background(),
				urls:  makeUrls(ts, 8),
				limit: 2,
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, "some_text")
			},
			want: makeRes(ts, "some_text", 8),
		},
		{
			name: "spawn overflow workers",
			fields: fields{
				fixed:    1,
				overflow: 4,
			},
			args: args{
				ctx:   context.Background(),
				urls:  makeUrls(ts, 10),
				limit: 5,
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, "some_text")
			},
			want: makeRes(ts, "some_text", 10),
		},
		{
			name: "pool size is lower that limit",
			fields: fields{
				fixed:    1,
				overflow: 3,
			},
			args: args{
				ctx:   context.Background(),
				urls:  makeUrls(ts, 10),
				limit: 5,
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, "some_text")
			},
			wantErr: true,
		},
		{
			name: "can't spawn more overflow workers",
			fields: fields{
				fixed:    1,
				overflow: 4,
				spawned:  4,
			},
			args: args{
				ctx:   context.Background(),
				urls:  makeUrls(ts, 10),
				limit: 5,
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				fmt.Fprint(w, "some_text")
			},
			wantErr: true,
		},
		{
			name: "ending of collect context",
			fields: fields{
				fixed: 2,
			},
			args: args{
				ctx:   withTimeout(context.Background(), time.Millisecond*100),
				urls:  makeUrls(ts, 8),
				limit: 2,
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(time.Second / 10)
				fmt.Fprint(w, "some_text")
			},
			wantErr: true,
		},
		{
			name: "ending of outer context",
			fields: fields{
				fixed: 2,
			},
			args: args{
				ctx:   context.Background(),
				urls:  makeUrls(ts, 10),
				limit: 2,
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(time.Second / 3)
				fmt.Fprint(w, "some_text")
			},
			want: makeRes(ts, "some_text", 10),
			cancelBefore: func(cancelFunc context.CancelFunc) {
				time.Sleep(time.Second)
				cancelFunc()
			},
		},
		{
			name: "immediately ending of outer context, try to elastic grow",
			fields: fields{
				fixed:    4,
				overflow: 4,
			},
			args: args{
				ctx:   context.Background(),
				urls:  makeUrls(ts, 10),
				limit: 4,
			},
			handler: func(w http.ResponseWriter, r *http.Request) {
				time.Sleep(time.Second / 3)
				fmt.Fprint(w, "some_text")
			},
			wantErr: true,
			cancelBefore: func(cancelFunc context.CancelFunc) {
				cancelFunc()
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts.Config.Handler = tt.handler

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			if tt.cancelBefore != nil {
				go tt.cancelBefore(cancel)
			}

			c := NewCollector(tt.fields.fixed, tt.fields.overflow, time.Second)
			c.(*collector).spawned = tt.fields.spawned

			c.Start(ctx)

			time.Sleep(time.Second / 2)

			got, err := c.Collect(tt.args.ctx, tt.args.urls, tt.args.limit)
			if (err != nil) != tt.wantErr {
				t.Errorf("Collect() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Collect() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func withTimeout(ctx context.Context, dur time.Duration) (ret context.Context) {
	ret, _ = context.WithTimeout(ctx, dur)
	return
}

func makeRes(ts *httptest.Server, msg string, count int) []res {
	var ress = make([]res, count)

	for i := 0; i < count; i++ {
		ress[i] = res{Url: ts.URL, Body: msg}
	}

	return ress
}

func makeUrls(ts *httptest.Server, count int) []string {
	var urls = make([]string, count)

	for i := 0; i < count; i++ {
		urls[i] = ts.URL
	}

	return urls
}
