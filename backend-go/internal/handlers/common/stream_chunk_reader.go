package common

import (
	"bytes"
	"io"
	"time"
)

// ChunkChannelReadCloser exposes chunks read by a background goroutine as an io.ReadCloser.
// Preflight code can consume some chunks from the same channel first, then wrap the remaining
// channel with this reader so no bytes are lost between preflight and normal streaming.
type ChunkChannelReadCloser struct {
	chunks  <-chan []byte
	errs    <-chan error
	closer  io.Closer
	current []byte
}

func NewChunkChannelReadCloser(chunks <-chan []byte, errs <-chan error, closer io.Closer) *ChunkChannelReadCloser {
	return &ChunkChannelReadCloser{chunks: chunks, errs: errs, closer: closer}
}

func (r *ChunkChannelReadCloser) Read(p []byte) (int, error) {
	for len(r.current) == 0 {
		chunk, ok := <-r.chunks
		if !ok {
			select {
			case err := <-r.errs:
				if err != nil {
					return 0, err
				}
			default:
			}
			return 0, io.EOF
		}
		r.current = chunk
	}

	n := copy(p, r.current)
	r.current = r.current[n:]
	return n, nil
}

func (r *ChunkChannelReadCloser) ReadWithTimeout(p []byte, timeout time.Duration) (int, error, bool) {
	if timeout <= 0 {
		n, err := r.Read(p)
		return n, err, false
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for len(r.current) == 0 {
		select {
		case chunk, ok := <-r.chunks:
			if !ok {
				select {
				case err := <-r.errs:
					if err != nil {
						return 0, err, false
					}
				default:
				}
				return 0, io.EOF, false
			}
			r.current = chunk
		case <-timer.C:
			return 0, nil, true
		}
	}
	n := copy(p, r.current)
	r.current = r.current[n:]
	return n, nil, false
}

func (r *ChunkChannelReadCloser) Close() error {
	if r.closer != nil {
		return r.closer.Close()
	}
	return nil
}

type prefixedReadCloser struct {
	reader io.Reader
	closer io.Closer
}

func NewPrefixedReadCloser(prefix []byte, body io.ReadCloser) io.ReadCloser {
	return &prefixedReadCloser{
		reader: io.MultiReader(bytes.NewReader(prefix), body),
		closer: body,
	}
}

func (r *prefixedReadCloser) Read(p []byte) (int, error) {
	return r.reader.Read(p)
}

func (r *prefixedReadCloser) Close() error {
	if r.closer != nil {
		return r.closer.Close()
	}
	return nil
}

func StartBodyChunkReader(body io.ReadCloser, chunkSize int, bufferSize int) (<-chan []byte, <-chan error) {
	if chunkSize <= 0 {
		chunkSize = 32 * 1024
	}
	if bufferSize <= 0 {
		bufferSize = 16
	}

	chunkChan := make(chan []byte, bufferSize)
	errChan := make(chan error, 1)
	go func() {
		defer close(chunkChan)
		buf := make([]byte, chunkSize)
		for {
			n, err := body.Read(buf)
			if n > 0 {
				chunk := make([]byte, n)
				copy(chunk, buf[:n])
				chunkChan <- chunk
			}
			if err != nil {
				if err != io.EOF {
					errChan <- err
				}
				return
			}
		}
	}()
	return chunkChan, errChan
}
