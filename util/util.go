package util

type OffsetWriter interface {
	Truncate(size int64) error
	WriteAt(b []byte, off int64) (n int, err error)
}
