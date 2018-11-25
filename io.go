package main

import (
	"errors"
	"io"
)

type F struct {
	buffer []byte
	ptr    int
}

func NewF(buffer []byte) *F {
	o := new(F)
	o.buffer = buffer
	return o
}

func (l *F) Read(o []byte) (n int, err error) {
	if l.ptr >= len(l.buffer) {
		return 0, io.EOF
	}
	n = copy(o, l.buffer[l.ptr:])
	l.ptr += n
	return n, nil

}

func (l *F) Write(o []byte) (n int, err error) {
	if l.ptr >= len(l.buffer) {
		return 0, io.EOF
	}
	n = copy(l.buffer[l.ptr:], o)
	l.ptr += n
	return n, nil

}

func (f *F) Seek(offset int64, whence int) (int64, error) {
	switch whence {
	case io.SeekStart:
		if offset < 0 {
			return int64(f.ptr), errors.New("Seeking before beginning of file")
		}
		f.ptr = int(offset)
		return int64(f.ptr), nil
	case io.SeekCurrent:
		if f.ptr+int(offset) < 0 {
			return int64(f.ptr), errors.New("Seeking before beginning of file")
		}
		f.ptr += int(offset)
		return int64(f.ptr), nil
	case io.SeekEnd:
		if len(f.buffer)+int(offset) < 0 {
			return int64(f.ptr), errors.New("Seeking before beginning of file")
		}
		f.ptr = len(f.buffer) + int(offset)
		return int64(f.ptr), nil
	default:
		return int64(f.ptr), errors.New("invalid whence value")

	}
	return 0, nil
}
