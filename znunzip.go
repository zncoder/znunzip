package main

import (
	"archive/zip"
	"bytes"
	"flag"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/zncoder/zniconv"
)

var (
	extract = flag.Bool("x", false, "extract")
	charset = flag.String("c", "gb18030", "charset used in the zip file")
)

func main() {
	flag.Parse()

	for _, zf := range flag.Args() {
		unzip(zf)
	}
}

func unzip(zf string) {
	r, err := zip.OpenReader(zf)
	if err != nil {
		log.Fatalf("open zip reader of file=%s err=%v", zf, err)

	}
	defer r.Close()

	conv, err := zniconv.NewReader(zniconv.Options{From: *charset}, nil)
	defer conv.Close()

	for _, f := range r.File {
		unzipOne(f, conv)
	}
}

func convertName(fn string, conv *zniconv.Reader) string {
	conv.Reset(bytes.NewBufferString(fn))
	b, err := conv.ReadAll()
	if err != nil {
		log.Fatalf("conv fn=%s err=%v", fn, err)
	}
	return string(b)
}

func unzipOne(zf *zip.File, conv *zniconv.Reader) {
	fn := convertName(zf.Name, conv)

	if !*extract {
		log.Println(fn)
		return
	}

	if d, f := filepath.Split(fn); d != "" {
		log.Printf("mkdir entry=%s", d)
		if err := os.MkdirAll(d, zf.Mode()|0770); err != nil {
			log.Fatalf("mkdirall d=%s err=%v", d, err)
		}
		if f == "" {
			return
		}
	}

	log.Printf("extracting file=%s", fn)
	in, err := zf.Open()
	if err != nil {
		log.Fatalf("open zip file=%s err=%v", fn, err)
	}
	out, err := os.Create(fn)
	if err != nil {
		log.Fatalf("create zip file=%s err=%v", fn, err)
	}
	if _, err = io.Copy(out, in); err != nil {
		log.Fatalf("extract zip file=%s err=%v", fn, err)

	}

	out.Close()
	in.Close()

	if err = os.Chmod(fn, zf.Mode()); err != nil {
		log.Printf("set file=%s to mode=%v err=%v", fn, zf.Mode(), err)
	}

	if err = os.Chtimes(fn, zf.ModTime(), zf.ModTime()); err != nil {
		log.Printf("set file=%s modtime=%v err=%v", fn, zf.ModTime(), err)
	}
}
