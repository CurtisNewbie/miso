package util

import (
	"encoding/csv"
	"io"

	"golang.org/x/text/encoding"
	"golang.org/x/text/encoding/unicode"
	"golang.org/x/text/transform"
)

func CsvReader(reader io.Reader) *csv.Reader {
	var transformer = unicode.BOMOverride(encoding.Nop.NewDecoder())
	return csv.NewReader(transform.NewReader(reader, transformer))
}

func CsvReadAll(reader io.Reader) ([][]string, error) {
	records, err := CsvReader(reader).ReadAll()
	if err != nil {
		return nil, err
	}
	return records, nil
}

func CsvReadAllIgnoreEmpty(reader io.Reader) ([][]string, error) {
	records := [][]string{}
	r := CsvReader(reader)
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
