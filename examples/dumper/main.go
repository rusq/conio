package main

import (
	"bytes"
	"compress/gzip"
	"errors"
	"flag"
	"io"
	"log"
	"os"

	"github.com/rusq/conio"
)

var (
	dumpfile = flag.String("d", "dump.bin", "dump `filename`")
	read     = flag.Bool("r", false, "read the dump file and write the reconstructed data")
	output   = flag.String("o", "", "reconstructed file name")
	compress = flag.Bool("z", false, "do compression")
)

func main() {
	flag.Parse()

	var err error
	if *read {
		err = reconstruct(*output, *dumpfile, *compress)
	} else {
		var in string
		if flag.NArg() > 0 {
			in = flag.Arg(0)
		}
		err = dumpwrite(*dumpfile, in, *compress)
	}
	if err != nil {
		log.Fatal(err)
	}
}

func reconstruct(dst, src string, compress bool) error {
	if src == "" {
		return errors.New("source file not given")
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	var out io.Writer = os.Stdout
	if dst != "" {
		f, err := os.Create(dst)
		if err != nil {
			return err
		}
		defer f.Close()
		out = f
	}

	r := conio.NewReader(in)
	cr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	n, err := io.Copy(out, cr)
	log.Printf("%d bytes read", n)
	return err
}

func dumpwrite(dst, src string, compress bool) error {

	data := []byte{16, 9, 19, 91}

	var (
		r io.Reader
	)
	if flag.NArg() > 0 {
		filename := flag.Arg(0)
		fi, err := os.Stat(filename)
		if err != nil || !fi.Mode().IsRegular() {
			return errors.New("must be a regular file")
		}
		f, err := os.Open(filename)
		if err != nil {
			return err
		}
		defer f.Close()
		r = f
		// sz = fi.Size()
	} else {
		r = bytes.NewReader(data)
	}

	output, err := os.Create(*dumpfile)
	if err != nil {
		return err
	}
	defer output.Close()

	nw := conio.NewWriter(output)
	if err != nil {
		return err
	}
	czw := gzip.NewWriter(nw)
	if err != nil {
		return err
	}
	n, err := io.Copy(czw, r)
	if err != nil {
		return err
	}
	if err := czw.Close(); err != nil {
		return err
	}
	if err := nw.Close(); err != nil {
		return err
	}
	log.Printf("%d bytes copied", n)
	return nil
}
