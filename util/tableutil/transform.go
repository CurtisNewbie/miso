package tableutil

import (
	"fmt"
	"io"
	"strings"

	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util/slutil"
)

// Table Transform
//
// Transform rows and columns while reading.
type TableTransform struct {
	// Skip row range.
	//
	// You don't need to skip rows in HeaderRowRangeExpr.
	SkipRowRangeSpec *Range
	// Include column range.
	InclColRangeSpec *Range
	// Header row range.
	HeaderRowRangeSpec *Range
	// Row Seperator, default to `'\n'`
	RowSeperator string
	// Col Seperator, default to `'  '` (two spaces)
	ColSeperator string
}

func (p TableTransform) HeaderRowRange() Range {
	if p.HeaderRowRangeSpec == nil {
		return ZeroRange()
	}
	return *p.HeaderRowRangeSpec
}

func (p TableTransform) SkipRowRange() Range {
	if p.SkipRowRangeSpec == nil {
		return ZeroRange()
	}
	return *p.SkipRowRangeSpec
}

func (p TableTransform) InclColRange() Range {
	if p.InclColRangeSpec == nil {
		return ZeroRange()
	}
	return *p.InclColRangeSpec
}

func (p TableTransform) WriteRow(buf *strings.Builder, header []string, inclColRange Range, r []string) {
	if buf.Len() > 0 {
		buf.WriteRune('\n')
		buf.WriteString(p.RowSeperator)
	}

	sp := p.ColSeperator
	if sp == "" {
		sp = strings.Repeat(" ", 2)
	}

	fst := true
	for i, v := range r {
		if !inclColRange.Nil() && !inclColRange.In(i) {
			continue
		}
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		if !fst {
			_, _ = buf.WriteString(sp)
		}
		if i < len(header) && header[i] != "" {
			_, _ = buf.WriteString(fmt.Sprintf("%v: ", header[i]))
		}
		_, _ = buf.WriteString(v)
		fst = false
	}
}

type loadedFileReader struct {
	i    int
	rows [][]string
}

func (l *loadedFileReader) Read() ([]string, error) {
	if l.i >= len(l.rows) {
		return nil, io.EOF
	}
	r := l.rows[l.i]
	l.i++
	return r, nil
}

func (p TableTransform) ParseFileLoaded(rail miso.Rail, rows [][]string) (string, error) {
	return p.ParseFile(rail, &loadedFileReader{rows: rows})
}

func (p TableTransform) ParseFile(rail miso.Rail, reader interface{ Read() ([]string, error) }) (string, error) {
	skipRowRange := p.SkipRowRange()
	inclColRange := p.InclColRange()
	headerRowRange := p.HeaderRowRange()
	rail.Infof("SkipRowRange: %#v, InclColRange: %#v, HeaderRowRange: %#v", skipRowRange, inclColRange, headerRowRange)

	var header []string
	b := &strings.Builder{}
	validRow := func(i int, row []string) bool {
		if skipRowRange.In(i) {
			return false
		}

		if headerRowRange.In(i) {
			header = slutil.Copy(row)
			return false
		}

		for j, c := range row {
			if !inclColRange.Nil() && !inclColRange.In(j) {
				continue
			}
			if c != "" {
				return true
			}
		}
		return false
	}
	i := 0
	for {
		record, err := reader.Read()
		if err == io.EOF {
			return b.String(), nil
		}
		if err != nil {
			return "", err
		}
		if validRow(i, record) {
			p.WriteRow(b, header, inclColRange, record)
		}
		i++
	}
}

func ZeroRange() Range {
	return Range{Min: -1, Max: -1}
}

type Range struct {
	Min int
	Max int
}

func (c Range) In(i int) bool {
	return i >= c.Min && i <= c.Max
}

func (c Range) Nil() bool {
	return c.Min < 0 && c.Max < 0
}
