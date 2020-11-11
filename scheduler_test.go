package socket

import (
	"context"
	"github.com/stretchr/testify/assert"
	"net"
	"reflect"
	"sync"
	"testing"
)

func TestNewScheduler(t *testing.T) {
	ctx := context.WithValue(context.Background(), "test1", "value1")
	con1, _ := net.Pipe()
	cc1 := NewConnection(con1)
	ch := make(chan []byte, 100)
	sc := NewScheduler(ctx, cc1, ch)
	assert.EqualValues(t, sc, &Scheduler{
		Conn:      cc1,
		Ctx:       ctx,
		WriteChan: ch,
		values:    make(map[interface{}]interface{}),
	})
}

func TestScheduler_Decrement(t *testing.T) {
	type fields struct {
		Conn      *Connection
		Ctx       context.Context
		WriteChan chan<- []byte
		rw        sync.RWMutex
		values    map[interface{}]interface{}
	}
	type args struct {
		key interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{"test decrement 1", fields{
			values: make(map[interface{}]interface{}),
		}, args{"key1"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &Scheduler{
				Conn:      tt.fields.Conn,
				Ctx:       tt.fields.Ctx,
				WriteChan: tt.fields.WriteChan,
				rw:        tt.fields.rw,
				values:    tt.fields.values,
			}
			if got := sc.Decrement(tt.args.key); got != tt.want {
				t.Errorf("Decrement() = %v, want %v", got, tt.want)
			}
			if got := sc.Decrement(tt.args.key); got != tt.want-1 {
				t.Errorf("Decrement() = %v, want %v", got, tt.want-1)
			}
		})
	}
}

func TestScheduler_Delete(t *testing.T) {
	type fields struct {
		Conn      *Connection
		Ctx       context.Context
		WriteChan chan<- []byte
		rw        sync.RWMutex
		values    map[interface{}]interface{}
	}
	type args struct {
		key interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{"test delete", fields{
			values: make(map[interface{}]interface{}),
		}, args{
			key: "test1",
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &Scheduler{
				Conn:      tt.fields.Conn,
				Ctx:       tt.fields.Ctx,
				WriteChan: tt.fields.WriteChan,
				values:    tt.fields.values,
			}
			sc.Store(tt.args.key, "123")
			_, ok := sc.Load(tt.args.key)
			assert.True(t, ok)
			sc.Delete(tt.args.key)
			_, ok = sc.Load(tt.args.key)
			assert.True(t, !ok)
		})
	}
}

func TestScheduler_Increment(t *testing.T) {
	type fields struct {
		Conn      *Connection
		Ctx       context.Context
		WriteChan chan<- []byte
		rw        sync.RWMutex
		values    map[interface{}]interface{}
	}
	type args struct {
		key interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{"test decrement 1", fields{
			values: make(map[interface{}]interface{}),
		}, args{"key1"}, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &Scheduler{
				Conn:      tt.fields.Conn,
				Ctx:       tt.fields.Ctx,
				WriteChan: tt.fields.WriteChan,
				rw:        tt.fields.rw,
				values:    tt.fields.values,
			}
			if got := sc.Increment(tt.args.key); got != tt.want {
				t.Errorf("Increment() = %v, want %v", got, tt.want)
			}
			if got := sc.Increment(tt.args.key); got != tt.want+1 {
				t.Errorf("Increment() = %v, want %v", got, tt.want+1)
			}
		})
	}
}

func TestScheduler_Load(t *testing.T) {
	type fields struct {
		Conn      *Connection
		Ctx       context.Context
		WriteChan chan<- []byte
		rw        sync.RWMutex
		values    map[interface{}]interface{}
	}
	type args struct {
		key interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   interface{}
		want1  bool
	}{
		{"test load", fields{
			values: map[interface{}]interface{}{
				"123": "456",
			},
		}, args{"123"}, "456", true},
		{"test load", fields{
			values: make(map[interface{}]interface{}),
		}, args{"123"}, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &Scheduler{
				Conn:      tt.fields.Conn,
				Ctx:       tt.fields.Ctx,
				WriteChan: tt.fields.WriteChan,
				rw:        tt.fields.rw,
				values:    tt.fields.values,
			}
			got, got1 := sc.Load(tt.args.key)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("Load() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Load() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestScheduler_LoadAndDelete(t *testing.T) {
	type fields struct {
		Conn      *Connection
		Ctx       context.Context
		WriteChan chan<- []byte
		rw        sync.RWMutex
		values    map[interface{}]interface{}
	}
	type args struct {
		key interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   interface{}
		want1  bool
	}{
		{"test load", fields{
			values: map[interface{}]interface{}{
				"123": "456",
			},
		}, args{"123"}, "456", true},
		{"test load", fields{
			values: make(map[interface{}]interface{}),
		}, args{"123"}, nil, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &Scheduler{
				Conn:      tt.fields.Conn,
				Ctx:       tt.fields.Ctx,
				WriteChan: tt.fields.WriteChan,
				rw:        tt.fields.rw,
				values:    tt.fields.values,
			}
			got, got1 := sc.LoadAndDelete(tt.args.key)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("LoadAndDelete() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("LoadAndDelete() got1 = %v, want %v", got1, tt.want1)
			}
			_, got3 := sc.LoadAndDelete(tt.args.key)
			if got3 != false {
				t.Errorf("LoadAndDelete() got1 = %v, want %v", got1, false)
			}
		})
	}
}

func TestScheduler_Send(t *testing.T) {
	ch := make(chan []byte, 10)
	sc := NewScheduler(context.Background(), nil, ch)
	sc.Send([]byte{0x11, 0x22, 0xff})
	by := <-ch
	assert.EqualValues(t, []byte{0x11, 0x22, 0xff}, by)
}

func TestScheduler_Store(t *testing.T) {
	type fields struct {
		Conn      *Connection
		Ctx       context.Context
		WriteChan chan<- []byte
		rw        sync.RWMutex
		values    map[interface{}]interface{}
	}
	type args struct {
		key   interface{}
		value interface{}
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		{"test store", fields{
			values: make(map[interface{}]interface{}),
		}, args{"test1", "test111"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			sc := &Scheduler{
				Conn:      tt.fields.Conn,
				Ctx:       tt.fields.Ctx,
				WriteChan: tt.fields.WriteChan,
				rw:        tt.fields.rw,
				values:    tt.fields.values,
			}
			sc.Store(tt.args.key, tt.args.value)
			value, ok := sc.Load(tt.args.key)
			assert.True(t, ok)
			assert.EqualValues(t, value, tt.args.value)
		})
	}
}
