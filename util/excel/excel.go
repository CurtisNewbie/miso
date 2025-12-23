package excel

import (
	"github.com/curtisnewbie/miso/miso"
	"github.com/curtisnewbie/miso/util/errs"
	"github.com/curtisnewbie/miso/util/slutil"
	"github.com/spf13/cast"
	"github.com/xuri/excelize/v2"
)

type ExcelSheet struct {
	Name        string
	Records     [][]string
	MergedCells []MergeCell
}

func (e *ExcelSheet) Append(r []string) {
	e.Records = append(e.Records, r)
}

// Present a merge cell in excel sheet.
//
// E.g., "C2:D4".
type MergeCell struct {
	StartAxis string // axis of top left cell. E.g., "C2".
	EndAxis   string // axis of bottom right cell. E.g., "D4".
	Val       string

	StartX int  // top left cell row no, e.g., "2" in "C2".
	StartY byte // top left cell col index, e.g., "C" in "C2".

	EndX int  // bottom right cell row no, e.g., "4" in "D4".
	EndY byte // bottom right cell col index, e.g., "D" in "D4"
}

func BuildMergeCell(t excelize.MergeCell) MergeCell {
	st := t.GetStartAxis()
	ed := t.GetEndAxis()
	return MergeCell{
		StartAxis: st,
		EndAxis:   ed,
		Val:       t.GetCellValue(),
		StartY:    st[0],
		StartX:    cast.ToInt(st[1:]),
		EndY:      ed[0],
		EndX:      cast.ToInt(ed[1:]),
	}
}

type readExcelOp struct {
	OverwriteMergeCell       bool
	WithHyperlink            bool
	WithHyperlinkPostProcess func(string) string
}

// Merge cell content is written to each row to maintain the overall structure.
//
// Without this option, merge cells' content only appear in the first row (just like how xlsx is normally converted to csv).
func OverwriteMergeCell() func(o *readExcelOp) {
	return func(o *readExcelOp) {
		o.OverwriteMergeCell = true
	}
}

// Include hyperlink if any.
func WithHyperlink(postprocess ...func(link string) string) func(o *readExcelOp) {
	f, ok := slutil.First(postprocess)
	return func(o *readExcelOp) {
		o.WithHyperlink = true
		if ok {
			o.WithHyperlinkPostProcess = f
		}
	}
}

// Read Excel File.
//
// Notice the whole excel file is loaded into memory.
//
// See [OverwriteMergeCell].
func ReadExcel(rail miso.Rail, fpath string, ops ...func(*readExcelOp)) ([]ExcelSheet, error) {
	op := &readExcelOp{}
	for _, f := range ops {
		f(op)
	}

	f, err := excelize.OpenFile(fpath)
	if err != nil {
		return nil, errs.Wrap(err)
	}
	defer func() {
		if err := f.Close(); err != nil {
			rail.Warnf("Failed to close excel file, %v", err)
		}
	}()

	sheetNames := f.GetSheetList()
	sheets := make([]ExcelSheet, 0, len(sheetNames))

	for _, s := range sheetNames {

		mcels, err := f.GetMergeCells(s)
		if err != nil {
			return nil, errs.Wrap(err)
		}

		rows, err := f.GetRows(s)
		if err != nil {
			return nil, errs.Wrap(err)
		}

		if op.WithHyperlink {
			for x, r := range rows {
				for y := range r {
					cell, err := excelize.CoordinatesToCellName(y+1, x+1)
					if err != nil {
						return nil, err
					}
					ok, hy, err := f.GetCellHyperLink(s, cell)
					if err != nil {
						return nil, err
					}
					if ok {
						if op.WithHyperlinkPostProcess != nil {
							hy = op.WithHyperlinkPostProcess(hy)
						}
						if hy != "" {
							rows[x][y] += " " + hy
						}
					}
				}
			}
		}

		st := &ExcelSheet{
			Name:    s,
			Records: make([][]string, 0, len(rows)),
			MergedCells: slutil.Transform(mcels, slutil.MapFunc(func(t excelize.MergeCell) MergeCell {
				return BuildMergeCell(t)
			}))}

		max := 0
		for i, row := range rows {
			if len(row) > max {
				max = len(row)
			}
			if op.OverwriteMergeCell {
				if len(row) < max { // pad empty cells
					row = append(row, make([]string, max-len(row))...)
				}
				for j := range row {
					if v, ok := InMergeCell(i, j, st.MergedCells); ok {
						row[j] = v
					}
				}
			}
			st.Append(row)
		}
		sheets = append(sheets, *st)
	}
	return sheets, nil
}

// Check if (x,y) in merge cell.
func InMergeCell(rowIdx int, colIdx int, mcels []MergeCell) (string, bool) {
	rx := rowIdx + 1
	ry := byte('A' + colIdx)
	for _, c := range mcels {
		sty := c.StartY
		stx := c.StartX
		edy := c.EndY
		edx := c.EndX
		if rx >= stx && rx <= edx && ry >= sty && ry <= edy {
			return c.Val, true
		}
	}
	return "", false
}

func Write(rail miso.Rail, fpath string, sheet string, records [][]string) error {
	f := excelize.NewFile()
	defer func() {
		if err := f.Close(); err != nil {
			rail.Warnf("Failed to close excel file, %v", err)
		}
	}()

	index, err := f.NewSheet(sheet)
	if err != nil {
		return errs.Wrap(err)
	}

	for i, r := range records {
		if err := f.SetSheetRow(sheet, "A"+cast.ToString(i+1), &r); err != nil {
			return errs.Wrap(err)
		}
	}

	f.SetActiveSheet(index)
	return errs.Wrap(f.SaveAs(fpath))
}
