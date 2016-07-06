package main

import (
	"archive/zip"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/htmlindex"
)

var (
	extract     = flag.Bool("x", false, "extract")
	keepPartial = flag.Bool("k", false, "keep partial result when unzip fails")
	charset     = flag.String("c", "gb18030", "charset used in the zip file")
)

var conv *encoding.Decoder

func main() {
	flag.Parse()

	cs, err := htmlindex.Get(*charset)
	if err != nil {
		log.Fatalf("get encoding of charset=%s err=%v", *charset, err)
	}
	conv = cs.NewDecoder()

	for _, zf := range flag.Args() {
		unzip(zf)
	}
}

func unzip(zf string) {
	r, err := zip.OpenReader(zf)
	if err != nil {
		log.Panicf("open zip reader of file=%s err=%v", zf, err)

	}
	defer r.Close()

	var tempdir string
	if *extract {
		tempdir, err = ioutil.TempDir(".", zf)
		if err != nil {
			log.Panicf("create tempdir err=%v", err)
		}
		os.Chdir(tempdir)

		defer func() {
			os.Chdir("..")

			if *keepPartial {
				return
			}

			err = os.RemoveAll(tempdir)
			if err != nil {
				log.Panicf("remove tempdir=%s err=%v", tempdir, err)
			}
		}()
	}

	for _, f := range r.File {
		unzipOne(f)
	}

	if *extract {
		d, err := os.Open(".")
		if err != nil {
			log.Panicf("open cwd err=%v", err)
		}
		names, err := d.Readdirnames(0)
		if err != nil {
			log.Panicf("readdirnames err=%v", err)
		}
		for _, name := range names {
			newname := "../" + name
			if _, err = os.Lstat(newname); err == nil {
				log.Panicf("would overwrite file=%s", name)
			}
		}
		for _, name := range names {
			newname := "../" + name
			if err = os.Rename(name, newname); err != nil {
				log.Panicf("rename %s to %s err=%v", name, newname, err)
			}
		}
	}
}

func unzipOne(zf *zip.File) {
	fn, err := conv.String(zf.Name)
	if err != nil {
		fn = zf.Name
	}

	if !*extract {
		log.Println(fn)
		return
	}

	if d, f := filepath.Split(fn); d != "" {
		log.Printf("mkdir entry=%s", d)
		if err := os.MkdirAll(d, zf.Mode()|0770); err != nil {
			log.Panicf("mkdirall d=%s err=%v", d, err)
		}
		if f == "" {
			return
		}
	}

	log.Printf("extracting file=%s", fn)
	in, err := zf.Open()
	if err != nil {
		log.Panicf("open zip file=%s err=%v", fn, err)
	}
	out, err := os.Create(fn)
	if err != nil {
		log.Panicf("create zip file=%s err=%v", fn, err)
	}
	if _, err = io.Copy(out, in); err != nil {
		log.Panicf("extract zip file=%s err=%v", fn, err)

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
