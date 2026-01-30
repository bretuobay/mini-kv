package index

// pathMatch implements basic glob matching for '*' and '?'.
func pathMatch(pattern, name string) (bool, error) {
	p := []rune(pattern)
	n := []rune(name)

	var pi, ni int
	var starIdx = -1
	var matchIdx = 0

	for ni < len(n) {
		if pi < len(p) && (p[pi] == '?' || p[pi] == n[ni]) {
			pi++
			ni++
			continue
		}
		if pi < len(p) && p[pi] == '*' {
			starIdx = pi
			matchIdx = ni
			pi++
			continue
		}
		if starIdx != -1 {
			pi = starIdx + 1
			matchIdx++
			ni = matchIdx
			continue
		}
		return false, nil
	}

	for pi < len(p) && p[pi] == '*' {
		pi++
	}
	return pi == len(p), nil
}
