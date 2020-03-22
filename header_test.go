package conio

import (
	"bytes"
	"errors"
	"io"
	"reflect"
	"testing"
)

func Test_binheader_Bytes(t *testing.T) {
	type fields struct {
		size   int
		closed int
	}
	tests := []struct {
		name      string
		fields    fields
		want      []byte
		wantPanic bool
	}{
		{"zero open", fields{size: 0, closed: 0}, []byte{0, 0, 0, 0x00}, false},
		{"zero closed", fields{size: 0, closed: 1}, []byte{0, 0, 0, 0x80}, false},
		{"max size open", fields{size: 2147483647, closed: 0}, []byte{0xff, 0xff, 0xff, 0x7f}, false},
		{"max size closed", fields{size: 2147483647, closed: 1}, []byte{0xff, 0xff, 0xff, 0xff}, false},
		{"max size closed", fields{size: 1288490188, closed: 0}, []byte{0xcc, 0xcc, 0xcc, 0x4c}, false},
		{"overflow MAX", fields{size: 2147483648, closed: 0}, []byte{0xff, 0xff, 0xff, 0x7f}, true},
		{"< 0", fields{size: -1, closed: 0}, []byte{0xff, 0xff, 0xff, 0x7f}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if (r != nil) != tt.wantPanic {
					t.Fatalf("panicked, but panic was not expected: %s", r)
				}
			}()
			b := &binheader{
				size:   tt.fields.size,
				closed: tt.fields.closed,
			}
			if got := b.Bytes(); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("binheader.Bytes() = % x, want % x", got, tt.want)
			}
		})
	}
}

func Test_loadHeader(t *testing.T) {
	type args struct {
		p []byte
	}
	tests := []struct {
		name    string
		args    args
		want    *binheader
		wantErr bool
	}{
		{"zero open", args{[]byte{0, 0, 0, 0x00}}, &binheader{size: 0, closed: 0}, false},
		{"zero closed", args{[]byte{0, 0, 0, 0x80}}, &binheader{size: 0, closed: 1}, false},
		{"max size open", args{[]byte{0xff, 0xff, 0xff, 0x7f}}, &binheader{size: 2147483647, closed: 0}, false},
		{"max size closed", args{[]byte{0xff, 0xff, 0xff, 0xff}}, &binheader{size: 2147483647, closed: 1}, false},
		{"bytes empty", args{[]byte{}}, nil, true},
		{"bytes too small", args{[]byte{2, 3}}, nil, true},
		{"bytes bigger", args{[]byte{0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC}}, &binheader{size: 1288490188, closed: 1}, false},
		{"bytes bigger", args{[]byte{0xCC, 0xCC, 0xCC, 0x4C, 0xCC, 0xCC}}, &binheader{size: 1288490188, closed: 0}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := loadHeader(tt.args.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("loadHeader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("loadHeader() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_binheader_WriteTo(t *testing.T) {
	type fields struct {
		size   int
		closed int
	}
	tests := []struct {
		name    string
		fields  fields
		want    int64
		wantW   string
		wantErr bool
	}{
		{"simple", fields{size: 0x30313233, closed: 0}, 4, "3210", false},
		{"closed", fields{size: 0x30313233, closed: 1}, 4, "321\xb0", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &binheader{
				size:   tt.fields.size,
				closed: tt.fields.closed,
			}
			w := &bytes.Buffer{}
			got, err := b.WriteTo(w)
			if (err != nil) != tt.wantErr {
				t.Errorf("binheader.WriteTo() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("binheader.WriteTo() = %v, want %v", got, tt.want)
			}
			if gotW := w.String(); gotW != tt.wantW {
				t.Errorf("binheader.WriteTo() = %q, want %q", gotW, tt.wantW)
			}
		})
	}
}

func Test_binheader_sane(t *testing.T) {
	type fields struct {
		size   int
		closed int
	}
	tests := []struct {
		name      string
		fields    fields
		wantPanic bool
	}{
		// {"zero", fields{size: 0, closed: 0}, false},
		{"<0", fields{size: -1, closed: 0}, true},
		{"too big", fields{size: 1 << (hdrSz * 8), closed: 0}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if (r != nil) != tt.wantPanic {
					t.Fatalf("panicked, but panic was not expected: %s", r)
				}
			}()
			b := &binheader{
				size:   tt.fields.size,
				closed: tt.fields.closed,
			}
			b.sane()
		})
	}
}

func Test_newBinHeader(t *testing.T) {
	type args struct {
		size   int
		closed bool
	}
	tests := []struct {
		name    string
		args    args
		wantHdr *binheader
		wantErr bool
	}{
		{"zero", args{size: 0, closed: false}, &binheader{size: 0, closed: 0}, false},
		{"zero closed", args{size: 0, closed: true}, &binheader{size: 0, closed: 1}, false},
		{"nonzero", args{size: 1000, closed: true}, &binheader{size: 1000, closed: 1}, false},
		{"nonzero closed", args{size: 160991, closed: true}, &binheader{size: 160991, closed: 1}, false},
		{"overflow", args{size: 1 << 32, closed: true}, nil, true},
		{"negative", args{size: -3, closed: true}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotHdr, err := newBinHeader(tt.args.size, tt.args.closed)
			if (err != nil) != tt.wantErr {
				t.Errorf("newBinHeader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotHdr, tt.wantHdr) {
				t.Errorf("newBinHeader() = %v, want %v", gotHdr, tt.wantHdr)
			}
		})
	}
}

func Test_binheader_Size(t *testing.T) {
	type fields struct {
		size   int
		closed int
	}
	tests := []struct {
		name   string
		fields fields
		want   int
	}{
		{"10", fields{size: 10, closed: 0}, 10},
		{"100", fields{size: 0xC0C0C0, closed: 0}, 0xC0C0C0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &binheader{
				size:   tt.fields.size,
				closed: tt.fields.closed,
			}
			if got := b.Size(); got != tt.want {
				t.Errorf("binheader.Size() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_binheader_IsClosed(t *testing.T) {
	type fields struct {
		size   int
		closed int
	}
	tests := []struct {
		name   string
		fields fields
		want   bool
	}{
		{"not closed", fields{closed: 0}, false},
		{"closed", fields{closed: 1}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &binheader{
				size:   tt.fields.size,
				closed: tt.fields.closed,
			}
			if got := b.IsClosed(); got != tt.want {
				t.Errorf("binheader.IsClosed() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_readHeader(t *testing.T) {
	type args struct {
		r io.Reader
	}
	tests := []struct {
		name    string
		args    args
		want    *binheader
		wantErr bool
	}{
		{"zero open", args{bytes.NewReader([]byte{0, 0, 0, 0x00})}, &binheader{size: 0, closed: 0}, false},
		{"zero closed", args{bytes.NewReader([]byte{0, 0, 0, 0x80})}, &binheader{size: 0, closed: 1}, false},
		{"max size open", args{bytes.NewReader([]byte{0xff, 0xff, 0xff, 0x7f})}, &binheader{size: 2147483647, closed: 0}, false},
		{"max size closed", args{bytes.NewReader([]byte{0xff, 0xff, 0xff, 0xff})}, &binheader{size: 2147483647, closed: 1}, false},
		{"bytes empty", args{bytes.NewReader([]byte{})}, nil, true},
		{"bytes too small", args{bytes.NewReader([]byte{2, 3})}, nil, true},
		{"bytes bigger", args{bytes.NewReader([]byte{0xCC, 0xCC, 0xCC, 0xCC, 0xCC, 0xCC})}, &binheader{size: 1288490188, closed: 1}, false},
		{"bytes bigger", args{bytes.NewReader([]byte{0xCC, 0xCC, 0xCC, 0x4C, 0xCC, 0xCC})}, &binheader{size: 1288490188, closed: 0}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := readHeader(tt.args.r)
			if (err != nil) != tt.wantErr {
				t.Errorf("readHeader() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("readHeader() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_must(t *testing.T) {
	type args struct {
		hdr *binheader
		err error
	}
	tests := []struct {
		name      string
		args      args
		want      *binheader
		wantPanic bool
	}{
		{"no panic on nil error", args{&binheader{size: 100, closed: 1}, nil}, &binheader{size: 100, closed: 1}, false},
		{"panic on error", args{nil, errors.New("must panic")}, nil, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defer func() {
				r := recover()
				if (r != nil) != tt.wantPanic {
					t.Fatalf("panicked, but panic was not expected: %s", r)
				}
			}()
			if got := must(tt.args.hdr, tt.args.err); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("must() = %v, want %v", got, tt.want)
			}
		})
	}
}
