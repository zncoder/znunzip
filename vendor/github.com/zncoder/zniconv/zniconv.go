// Package zniconv provides a Reader to convert the charset of data.
// It wraps an io.Reader, and converts the data read from the io.Reader to the target charset.
// The actual conversion is done by the glibc iconv.
//
// No Writer is provided, for two reasons,
// 1. conversion from one charset to another can be achieved with Reader
// 2. io.Writer requires that a short write return an error,
//    but short write does not play well with io.Copy.
//    Short write is legitimate when multibyte sequences are involved.
package zniconv

/*
#include <iconv.h>
#include <stdlib.h>

static void myiconv(iconv_t cd, char *inbuf, size_t *inbytesleft, char *outbuf, size_t *outbytesleft) {
  iconv(cd, &inbuf, inbytesleft, &outbuf, outbytesleft);
}

*/
import "C"

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"syscall"
	"unsafe"
)

//go:generate stringer -type=ErrCode
type ErrCode int

const (
	E2big ErrCode = 0 + iota
	Eilseq
	Einval
	Eio
	Eunknown
)

type Err struct {
	Code   ErrCode
	Reason string
}

func (er Err) Error() string {
	return fmt.Sprintf("%v: %s", er.Code, er.Reason)
}

var defaultBufSize = 16 * 1024

type Options struct {
	From    string // e.g. "gb18030"
	To      string // e.g. "utf8"
	BufSize int    // internal buffer size
}

func getCode(s string) *C.char {
	if s == "" {
		s = "utf8"
	}
	return C.CString(s)
}

func getBufSize(n int) int {
	if n == 0 {
		n = defaultBufSize
	}
	return n
}

type Reader struct {
	c       C.iconv_t
	r       io.Reader
	buf     []byte
	left    []byte
	err     error
	goff    int64
	bufSize int
}

// NewReader creates a Reader that converts the charset of data from r.
// The charset of data in r is opts.From, and the charset of the
// output of this Reader is opts.To.
func NewReader(opts Options, r io.Reader) (*Reader, error) {
	from := getCode(opts.From)
	defer C.free(unsafe.Pointer(from))
	to := getCode(opts.To)
	defer C.free(unsafe.Pointer(to))

	c, err := C.iconv_open(to, from)
	if err != nil {
		return nil, err
	}
	sz := getBufSize(opts.BufSize)
	return &Reader{
		c:       c,
		r:       r,
		buf:     make([]byte, sz),
		bufSize: sz,
	}, nil
}

// Read reads up to len(b) bytes to b. It returns the number of bytes read.
// Read only puts complete multibyte sequences to b, which means
// it may return less than len(b) bytes with no error.
func (r *Reader) Read(b []byte) (int, error) {
	off, n := 0, len(b)
	for off < n {
		r.refill()
		if len(r.left) == 0 || (r.err != nil && r.err != io.EOF) {
			break
		}

		i, o, err := iconv(r.c, r.left, b[off:])
		r.goff += int64(i)
		r.left = r.left[i:]
		off += o

		switch err {
		case nil:

		case syscall.E2BIG:
			// not enough space for the next multibyte sequence. this is not an error,
			// even if b is too small for a single multibyte sequence.
			return off, nil

		case syscall.EILSEQ:
			r.fail(Err{Code: Eilseq, Reason: fmt.Sprintf("invalid multibyte seq=%x at offset=%d", capped(r.left), r.goff)})

		case syscall.EINVAL:
			if r.err == io.EOF {
				// no bytes to refill, this is a real error.
				// otherwise keep refilling to handle EILSEQ
				r.fail(Err{Code: Einval, Reason: fmt.Sprintf("incomplete multibyte seq=%x at offset=%d", capped(r.left), r.goff)})
			}

		default:
			r.fail(Err{Code: Eunknown, Reason: fmt.Sprintf("unknown iconv err=%v at offset=%d", err, r.goff)})
		}
	}
	return off, r.err
}

func (r *Reader) Reset(rr io.Reader) {
	if _, err := C.iconv(r.c, nil, nil, nil, nil); err != nil {
		r.fail(Err{Code: Eunknown, Reason: fmt.Sprintf("unknown iconv err=%v at reset", err)})
		return
	}

	r.r = rr
	r.left = nil
	r.err = nil
	r.goff = 0
	if r.buf == nil {
		r.buf = make([]byte, r.bufSize)
	}
}

func (r *Reader) ReadAll() ([]byte, error) {
	out := new(bytes.Buffer)
	if _, err := io.Copy(out, r); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func capped(b []byte) []byte {
	if len(b) < 10 {
		return b
	}
	return b[:10]
}

func iconv(cd C.iconv_t, in, out []byte) (inOff, outOff int, err error) {
	cinleft := C.size_t(len(in))
	cin := (*C.char)(unsafe.Pointer(&in[0]))
	coutleft := C.size_t(len(out))
	cout := (*C.char)(unsafe.Pointer(&out[0]))
	_, err = C.myiconv(cd, cin, &cinleft, cout, &coutleft)
	inOff = len(in) - int(cinleft)
	outOff = len(out) - int(coutleft)
	return inOff, outOff, err
}

func (r *Reader) refill() {
	if r.err != nil {
		return
	}
	i := copy(r.buf, r.left)
	j, err := r.r.Read(r.buf[i:])
	r.left = r.buf[:i+j]
	switch err {
	case nil:
	case io.EOF:
		r.err = err
	default:
		r.err = Err{Code: Eio, Reason: fmt.Sprintf("read err=%v", err)}
	}
}

func (r *Reader) fail(err error) {
	if err == nil || err == io.EOF {
		log.Panicf("fail is called with err=%v", err)
	}
	if r.err != nil && r.err != io.EOF {
		// don't overwrite err
		return
	}
	r.err = err
	r.buf = nil
}

func (r *Reader) Close() error {
	_, err := C.iconv_close(r.c)
	return err
}

func Convert(from, to string, b []byte) ([]byte, error) {
	r, err := NewReader(Options{From: from, To: to}, bytes.NewBuffer(b))
	if err != nil {
		return nil, err
	}
	return r.ReadAll()
}
