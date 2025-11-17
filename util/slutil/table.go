package slutil

// Collect table col at idx.
func TableColAt(table [][]string, idx int) []string {
	if idx < 0 || len(table) < 1 || len(table[0]) <= idx {
		return []string{}
	}
	return MapTo(table, func(r []string) string { return r[idx] })
}
