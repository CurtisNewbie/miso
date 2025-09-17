package csv

import (
	"encoding/csv"
	"io"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

func Reader(reader io.Reader) *csv.Reader {
	var transformer = unicode.BOMOverride(encoding.Nop.NewDecoder())
	return csv.NewReader(transform.NewReader(reader, transformer))
}

func ReadAll(reader io.Reader) ([][]string, error) {
	records, err := Reader(reader).ReadAll()
	if err != nil {
		return nil, err
	}
	return records, nil
}

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
