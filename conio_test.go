// Package conio provides controlled I/O reader and writer.  It may be useful
// when you need to have a compressed reader/writer over the net.Conn and then
// resume your normal reads writes on it.
package conio

import (
	"bytes"
	"io"
	"testing"
)

func TestConWriter_Write(t *testing.T) {
	type args struct {
		p []byte
	}
	tests := []struct {
		name       string
		args       args
		want       int
		wantWriter []byte
		wantErr    bool
	}{
		{"sample write",
			args{[]byte{19, 91, 9, 16}},
			4,
			[]byte{04, 00, 00, 00, 19, 91, 9, 16},
			false,
		},
		{"empty write",
			args{[]byte{}},
			0,
			[]byte{},
			false,
		},
		{"overflow",
			args{make([]byte, maxSz+1)},
			0,
			[]byte{},
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			w := &ConWriter{
				w: buf,
			}
			got, err := w.Write(tt.args.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConWriter.Write() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ConWriter.Write() = %v, want %v", got, tt.want)
			}

			if gotWriter := buf.Bytes(); !bytes.Equal(gotWriter, tt.wantWriter) {
				t.Errorf("ConWriter.Write() = % x, want % x", gotWriter, tt.wantWriter)
			}
		})
	}
}

func TestConWriter_Close(t *testing.T) {
	tests := []struct {
		name       string
		wantWriter []byte
		wantErr    bool
	}{
		{"ok", []byte{0, 0, 0, 0x80}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := &bytes.Buffer{}
			w := &ConWriter{
				w: buf,
			}
			if err := w.Close(); (err != nil) != tt.wantErr {
				t.Errorf("ConWriter.Close() error = %v, wantErr %v", err, tt.wantErr)
			}
			if gotWriter := buf.Bytes(); !bytes.Equal(gotWriter, tt.wantWriter) {
				t.Errorf("ConWriter.Write() = % x, want % x", gotWriter, tt.wantWriter)
			}
		})
	}
}

func TestConReader_Read(t *testing.T) {
	type fields struct {
		r      io.Reader
		unread int
	}
	type args struct {
		p []byte
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int
		wantP   []byte
		wantErr bool
	}{
		{"closed",
			fields{
				r: bytes.NewReader([]byte{0, 0, 0, 0x80}), // this should result in EOF
			},
			args{make([]byte, 100)},
			0,
			[]byte{},
			true,
		},
		{"four bytes",
			fields{
				r: bytes.NewReader([]byte{04, 00, 00, 00, 19, 91, 9, 16}), // this should result in EOF
			},
			args{make([]byte, 100)},
			4,
			[]byte{19, 91, 9, 16},
			false,
		},
		{"four bytes unread",
			fields{
				r:      bytes.NewReader([]byte{19, 91, 9, 16}), // this should result in EOF
				unread: 4,
			},
			args{make([]byte, 100)},
			4,
			[]byte{19, 91, 9, 16},
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := &ConReader{
				r:      tt.fields.r,
				unread: tt.fields.unread,
			}
			got, err := r.Read(tt.args.p)
			if (err != nil) != tt.wantErr {
				t.Errorf("ConReader.Read() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("ConReader.Read() = %v, want %v", got, tt.want)
			}
			if !bytes.Equal(tt.args.p[:got], tt.wantP) {
				t.Errorf("ConWriter.Read() = % x, want % x", tt.args.p[:got], tt.wantP)
			}
		})
	}
}
