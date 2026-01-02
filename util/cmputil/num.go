package cmputil

func MinInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}

func MaxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}

func MinFloat(a float64, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

func MaxFloat(a float64, b float64) float64 {
	if a > b {
		return a
	}
	return b
}
