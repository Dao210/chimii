package build

import (
	"context"
	"math"
)

// linearFeasibility solves A*x <= b, x >= 0 by a bounded two-phase tableau.
// It is deliberately private and limited to the small reference load model.
func linearFeasibility(ctx context.Context, a [][]float64, b []float64, maxPivots int) ([]float64, string) {
	const epsilon = 1e-9
	m := len(b)
	if m == 0 || len(a) != m {
		return nil, "invalid linear system"
	}
	n := len(a[0])
	if n == 0 || m > 192 || n > 256 {
		return nil, "linear system outside model bounds"
	}
	d := make([][]float64, m+2)
	basis := make([]int, m)
	nonbasis := make([]int, n+1)
	for i := range d {
		d[i] = make([]float64, n+2)
	}
	for i := 0; i < m; i++ {
		if len(a[i]) != n || math.IsNaN(b[i]) || math.IsInf(b[i], 0) {
			return nil, "invalid linear system"
		}
		for _, v := range a[i] {
			if math.IsNaN(v) || math.IsInf(v, 0) {
				return nil, "invalid linear system"
			}
		}
		copy(d[i], a[i])
		basis[i] = n + i
		d[i][n] = -1
		d[i][n+1] = b[i]
	}
	for j := 0; j < n; j++ {
		nonbasis[j] = j
	}
	nonbasis[n] = -1
	d[m+1][n] = 1
	pivots := 0
	pivot := func(r, s int) {
		inverse := 1 / d[r][s]
		for i := 0; i < m+2; i++ {
			if i == r {
				continue
			}
			factor := d[i][s] * inverse
			for j := 0; j < n+2; j++ {
				if j != s {
					d[i][j] -= d[r][j] * factor
				}
			}
		}
		for j := 0; j < n+2; j++ {
			if j != s {
				d[r][j] *= inverse
			}
		}
		for i := 0; i < m+2; i++ {
			if i != r {
				d[i][s] *= -inverse
			}
		}
		d[r][s] = inverse
		basis[r], nonbasis[s] = nonbasis[s], basis[r]
		pivots++
	}
	simplex := func(phase int) string {
		row := m
		if phase == 1 {
			row = m + 1
		}
		for {
			if err := ctx.Err(); err != nil {
				return err.Error()
			}
			if pivots >= maxPivots {
				return "pivot budget exceeded"
			}
			s := -1
			for j := 0; j <= n; j++ {
				if phase == 2 && nonbasis[j] == -1 {
					continue
				}
				if s < 0 || d[row][j] < d[row][s]-epsilon || math.Abs(d[row][j]-d[row][s]) <= epsilon && nonbasis[j] < nonbasis[s] {
					s = j
				}
			}
			if s < 0 || d[row][s] >= -epsilon {
				return "optimal"
			}
			r := -1
			for i := 0; i < m; i++ {
				if d[i][s] <= epsilon {
					continue
				}
				if r < 0 || d[i][n+1]/d[i][s] < d[r][n+1]/d[r][s]-epsilon || math.Abs(d[i][n+1]/d[i][s]-d[r][n+1]/d[r][s]) <= epsilon && basis[i] < basis[r] {
					r = i
				}
			}
			if r < 0 {
				return "unbounded"
			}
			pivot(r, s)
		}
	}
	r := 0
	for i := 1; i < m; i++ {
		if d[i][n+1] < d[r][n+1] {
			r = i
		}
	}
	if d[r][n+1] < -epsilon {
		pivot(r, n)
		status := simplex(1)
		if status != "optimal" {
			return nil, status
		}
		if math.Abs(d[m+1][n+1]) > epsilon {
			return nil, "infeasible"
		}
		for i := 0; i < m; i++ {
			if basis[i] != -1 {
				continue
			}
			s := -1
			for j := 0; j <= n; j++ {
				if math.Abs(d[i][j]) > epsilon && (s < 0 || nonbasis[j] < nonbasis[s]) {
					s = j
				}
			}
			if s >= 0 {
				pivot(i, s)
			}
		}
	}
	if status := simplex(2); status != "optimal" {
		return nil, status
	}
	x := make([]float64, n)
	for i := 0; i < m; i++ {
		if basis[i] >= 0 && basis[i] < n {
			x[basis[i]] = d[i][n+1]
		}
	}
	for j, v := range x {
		if math.IsNaN(v) || math.IsInf(v, 0) || v < -1e-7 {
			return nil, "invalid numerical solution"
		}
		if v < 0 {
			x[j] = 0
		}
	}
	return x, "feasible"
}
