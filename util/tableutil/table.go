package tableutil

import "github.com/curtisnewbie/miso/util/slutil"

// Collect table col at idx.
func TableColAt(table [][]string, idx int) []string {
	if idx < 0 || len(table) < 1 || len(table[0]) <= idx {
		return []string{}
	}
	return slutil.MapTo(table, func(r []string) string { return r[idx] })
}

func PadTable(table [][]string, colCnt int) {
	for i, r := range table {
		if len(r) < colCnt {
			table[i] = append(r, make([]string, colCnt-len(r))...)
		}
	}
}
