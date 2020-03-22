package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"sync"

	"github.com/rusq/conio"
)

func init() {
	log.SetFlags(log.Lshortfile)
}

func main() {
	data := []byte{16, 9, 19, 91}

	var (
		r  io.Reader
		sz int64
	)
	if len(os.Args) > 1 {
		fi, err := os.Stat(os.Args[1])
		if err != nil || !fi.Mode().IsRegular() {
			log.Fatal("must be a regular file")
		}
		f, err := os.Open(os.Args[1])
		if err != nil {
			log.Fatal(err)
		}
		defer f.Close()
		r = f
		sz = fi.Size()
	} else {
		r = bytes.NewReader(data)
		sz = int64(len(data))
	}

	// emulating network connection
	client, server := net.Pipe()

	var wg sync.WaitGroup
	wg.Add(1)
	var serr error
	// sending the data async
	go func() {
		serr = send(server, r, sz)
		if serr != nil {
			panic(serr)
		}
		wg.Done()
	}()

	// receiving into buffer.
	var buf bytes.Buffer
	err := receive(&buf, client)
	if err != nil {
		log.Fatal(err)
	}
	wg.Wait()
	if serr != nil {
		log.Fatal("Send error: ", serr)
	}
}

type txinfo struct {
	Size int64
}

type txresult struct {
	OK   bool
	Data []byte
}

func send(conn net.Conn, r io.Reader, sz int64) error {
	dumper := io.TeeReader(conn, os.Stderr)
	enc, dec := json.NewEncoder(conn), json.NewDecoder(dumper)
	// send size
	if err := enc.Encode(txinfo{sz}); err != nil {
		return err
	}

	// copy data
	cw := conio.NewWriter(conn)
	if _, err := io.Copy(cw, r); err != nil {
		return err
	}
	if err := cw.Close(); err != nil {
		panic(err)
	}
	res := txresult{}
	// receive confirmation
	if err := dec.Decode(&res); err != nil {
		return err
	}
	// send something else
	if err := enc.Encode(txresult{Data: []byte{19, 91, 9, 16}}); err != nil {
		return err
	}
	// receive something else
	if err := dec.Decode(&res); err != nil {
		return err
	}
	// spew.Dump(res)
	log.Printf("server responded: %v", res)
	return nil
}

func receive(w io.Writer, conn net.Conn) error {
	dumper := io.TeeReader(conn, os.Stderr)
	enc, dec := json.NewEncoder(conn), json.NewDecoder(dumper)
	// receive info
	ti := txinfo{}
	if err := dec.Decode(&ti); err != nil {
		return err
	}
	cr := conio.NewReader(conn)
	// receive data
	if n, err := io.Copy(w, cr); err != nil || n != ti.Size {
		if err != nil {
			return err
		}
		return fmt.Errorf("error:  received: %d,  expected:  %d", n, ti.Size)
	}
	// send result
	if err := enc.Encode(txresult{OK: true}); err != nil {
		return err
	}
	res := txresult{}
	// receive something else
	if err := dec.Decode(&res); err != nil {
		return err
	}
	// send something else
	if err := enc.Encode(txresult{Data: []byte("004")}); err != nil {
		return err
	}
	// spew.Dump(res)
	log.Println("received ok")
	return nil
}
