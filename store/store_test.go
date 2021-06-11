package store

import (
	"sync"
	"testing"

	"github.com/nsip/otf-weighting/util"
)

func TestOption_FileSyncToMap(t *testing.T) {
	type fields struct {
		Dir            string
		Ext            string
		OnFileConflict func(existing, coming string) (bool, string)
		SM             *sync.Map
		OnSMapConflict func(existing, coming string) (bool, string)
		M              map[interface{}]interface{}
		OnMapConflict  func(existing, coming string) (bool, string)
		WG             *sync.WaitGroup
		Mtx            *sync.Mutex
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		// TODO: Add test cases.
		{
			name: "OK",
			fields: fields{
				Dir: "../in",
				Ext: "json",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := &Option{
				Dir:            tt.fields.Dir,
				Ext:            tt.fields.Ext,
				OnFileConflict: tt.fields.OnFileConflict,
				SM:             tt.fields.SM,
				OnSMapConflict: tt.fields.OnSMapConflict,
				M:              tt.fields.M,
				OnMapConflict:  tt.fields.OnMapConflict,
				WG:             tt.fields.WG,
				Mtx:            tt.fields.Mtx,
			}
			opt.FileSyncToMap()
		})
	}
}

func TestOption_AppendJSONFromFile(t *testing.T) {
	type fields struct {
		WG             *sync.WaitGroup
		Mtx            *sync.Mutex
		Dir            string
		Ext            string
		OnFileConflict func(existing, coming string) (bool, string)
		M              map[interface{}]interface{}
		OnMapConflict  func(existing, coming string) (bool, string)
		SM             *sync.Map
		OnSMapConflict func(existing, coming string) (bool, string)
	}
	type args struct {
		dir string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		// TODO: Add test cases.
		{
			name: "OK",
			fields: fields{
				Dir:            "../in",
				Ext:            "json",
				OnFileConflict: util.FactoryAppendJA(),
				M:              map[interface{}]interface{}{},
				OnMapConflict:  util.FactoryAppendJA(),
				SM:             &sync.Map{},
				OnSMapConflict: util.FactoryAppendJA(),
			},
			args: args{
				dir: "../temp",
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := &Option{
				WG:             tt.fields.WG,
				Mtx:            tt.fields.Mtx,
				Dir:            tt.fields.Dir,
				Ext:            tt.fields.Ext,
				OnFileConflict: tt.fields.OnFileConflict,
				M:              tt.fields.M,
				OnMapConflict:  tt.fields.OnMapConflict,
				SM:             tt.fields.SM,
				OnSMapConflict: tt.fields.OnSMapConflict,
			}
			opt.AppendJSONFromFile(tt.args.dir)
		})
	}
}
