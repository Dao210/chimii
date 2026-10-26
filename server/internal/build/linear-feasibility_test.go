package build

import (
	"context"
	"math"
	"testing"
)

func TestReferenceLinearFeasibilityAnalyticSystems(t *testing.T) {
	for _, sample := range []struct {
		name   string
		a      [][]float64
		b      []float64
		status string
	}{
		{"compression", [][]float64{{1}, {-1}}, []float64{2, -2}, "feasible"},
		{"contradiction", [][]float64{{1}, {-1}}, []float64{1, -2}, "infeasible"},
		{"two_reactions", [][]float64{{1, 1}, {-1, -1}, {1, -1}, {-1, 1}}, []float64{4, -4, 0, 0}, "feasible"},
		{"redundant_equations", [][]float64{{1, 1}, {-1, -1}, {2, 2}, {-2, -2}}, []float64{4, -4, 8, -8}, "feasible"},
		{"unanchored_weight", [][]float64{{0}, {0}}, []float64{1, -1}, "infeasible"},
		{"invalid_coefficient", [][]float64{{math.NaN()}}, []float64{1}, "invalid linear system"},
	} {
		t.Run(sample.name, func(t *testing.T) {
			x, status := linearFeasibility(context.Background(), sample.a, sample.b, 200)
			if status != sample.status {
				t.Fatalf("status=%s want=%s", status, sample.status)
			}
			if status != "feasible" {
				return
			}
			for i, row := range sample.a {
				value := 0.0
				for j, a := range row {
					value += a * x[j]
				}
				if value > sample.b[i]+1e-8 {
					t.Fatalf("constraint %d violated: %g > %g", i, value, sample.b[i])
				}
			}
		})
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, status := linearFeasibility(ctx, [][]float64{{1}}, []float64{1}, 200); status == "feasible" {
		t.Fatal("cancelled solver returned success")
	}
	if _, status := linearFeasibility(context.Background(), [][]float64{{-1}}, []float64{-1}, 0); status != "pivot budget exceeded" {
		t.Fatalf("budget ignored: %s", status)
	}
}
