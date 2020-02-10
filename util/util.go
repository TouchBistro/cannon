package util

type OffsetWriter interface {
	WriteAt(b []byte, off int64) (n int, err error)
}
