package csv

import (
	"encoding/csv"
	"io"

	"github.com/curtisnewbie/miso/errs"
	"github.com/curtisnewbie/miso/util/osutil"
	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

// Write csv.
func Write(fpath string, records [][]string) error {
	f, err := osutil.OpenRWFile(fpath, true)
	if err != nil {
		return err
	}
	defer f.Close()
	w := csv.NewWriter(f)
	if err := w.WriteAll(records); err != nil {
		return errs.Wrap(err)
	}
	return nil
}

// Wrap csv data reader.
func Reader(reader io.Reader) *csv.Reader {
	var transformer = unicode.BOMOverride(encoding.Nop.NewDecoder())
	return csv.NewReader(transform.NewReader(reader, transformer))
}

// Read all csv content.
func ReadAll(reader io.Reader) ([][]string, error) {
	records, err := Reader(reader).ReadAll()
	if err != nil {
		return nil, err
	}
	return records, nil
}

// Read all csv content and ignore empty row.
func ReadAllIgnoreEmpty(reader io.Reader) ([][]string, error) {
	records := [][]string{}
	r := Reader(reader)
	validRow := func(row []string) bool {
		for _, c := range row {
			if c != "" {
				return true
			}
		}
		return false
	}
	for {
		record, err := r.Read()
		if err == io.EOF {
			return records, nil
		}
		if err != nil {
			return nil, err
		}
		if validRow(record) {
			records = append(records, record)
		}
	}
}
