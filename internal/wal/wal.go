package wal

type Writer struct {
	dir     string
	maxSize int
}

func NewWriter(dir string, maxSize int) *Writer {
	return &Writer{dir: dir, maxSize: maxSize}
}

func (w *Writer) Append(data []byte) (offset int64, err error) {
	return 0, nil
}

func (w *Writer) Flush() error {
	return nil
}

func (w *Writer) Recover() ([]Record, error) {
	return nil, nil
}

func (w *Writer) Close() error {
	return nil
}

type Record struct {
	Offset int64
	Data   []byte
	Flag   byte
}
