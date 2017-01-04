package main

func min(a, b int64) int64 {
	if a < b {
		return a
	}

	return b
}

func max(a, b int64) int64 {
	if a > b {
		return a
	}

	return b
}

func isBetween(a, b, c int64) bool {
	n, m := min(b, c), max(b, c)

	return a >= n && a <= m
}
