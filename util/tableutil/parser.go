package tableutil

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/curtisnewbie/miso/middleware/expr"
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util/slutil"
	"github.com/spf13/cast"
)

var (
	rowRangeExprPool = expr.NewPooledExpr[struct{}](100)
)

// Table Transform
//
// Transform rows and columns while reading.
type TableTransform struct {
	Name               string // Parser name
	SkipRowRangeExpr   string // Skip row range expression. [0] - start row index, [1] - end row index. E.g., `[0, 3]` or `[]`.
	InclColRangeExpr   string // Include column range expression. [0] - start col index, [1] - end col index. E.g., `[0, 3]` or `[]`.
	HeaderRowRangeExpr string // Header row range expression. [0] - start row index, [1] - end row index. E.g., `[0, 3]` or `[]`
	RowSeperator       string // Row Seperator, default to `'\n'`
	ColSeperator       string // Col Seperator, default to `'  '` (two spaces)
}

func (p TableTransform) HeaderRowRange() (Range, error) {
	return p.parseRowRange(p.HeaderRowRangeExpr)
}

func (p TableTransform) SkipRowRange() (Range, error) {
	return p.parseRowRange(p.SkipRowRangeExpr)
}

func (p TableTransform) InclColRange() (Range, error) {
	return p.parseRowRange(p.InclColRangeExpr)
}

func (p TableTransform) parseRowRange(s string) (Range, error) {
	if s == "" {
		return Range{Min: -1, Max: -1}, nil
	}
	v, err := rowRangeExprPool.Eval(s, struct{}{})
	if err != nil {
		return Range{}, err
	}
	is := cast.ToIntSlice(v)
	min := -1
	max := -1
	slices.Sort(is)
	if len(is) > 0 {
		min = is[0]
		max = is[len(is)-1]
	}
	return Range{Min: min, Max: max}, nil
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

func (p TableTransform) ParseFileLoaded(rail miso.Rail, rows [][]string) (*strings.Builder, error) {
	return p.ParseFile(rail, &loadedFileReader{rows: rows})
}

func (p TableTransform) ParseFile(rail miso.Rail, reader interface{ Read() ([]string, error) }) (*strings.Builder, error) {
	skipRowRange, err := p.SkipRowRange()
	if err != nil {
		return nil, err
	}
	inclColRange, err := p.InclColRange()
	if err != nil {
		return nil, err
	}
	headerRowRange, err := p.HeaderRowRange()
	if err != nil {
		return nil, err
	}
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
			return b, nil
		}
		if err != nil {
			return nil, err
		}
		if validRow(i, record) {
			p.WriteRow(b, header, inclColRange, record)
		}
		i++
	}
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
