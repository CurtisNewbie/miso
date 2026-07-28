package plotutil

import (
	"bytes"
	"fmt"
	"testing"

	"gonum.org/v1/plot/plotter"
)

func intsEqual(a, b []int) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestValueLabelIndices(t *testing.T) {
	cases := []struct {
		name string
		dat  plotter.XYs
		want []int
	}{
		{
			name: "normal: min at 1, max at 3, last 3 are 3,4,5",
			dat: plotter.XYs{
				{Y: 5},  // 0
				{Y: 1},  // 1 min
				{Y: 4},  // 2
				{Y: 10}, // 3 max
				{Y: 6},  // 4
				{Y: 7},  // 5
			},
			// min(1), last3: 3=max skip, 4, 5, max(3)
			want: []int{1, 4, 5, 3},
		},
		{
			name: "last point is max",
			dat: plotter.XYs{
				{Y: 3},  // 0
				{Y: 1},  // 1 min
				{Y: 5},  // 2
				{Y: 4},  // 3
				{Y: 10}, // 4 max
			},
			// min(1), last3: 2, 3, 4=max skip, max(4)
			want: []int{1, 2, 3, 4},
		},
		{
			name: "last point is min",
			dat: plotter.XYs{
				{Y: 5},  // 0
				{Y: 10}, // 1 max
				{Y: 4},  // 2
				{Y: 3},  // 3
				{Y: 1},  // 4 min
			},
			// min(4), last3: 2, 3, 4=seen skip, max(1)
			want: []int{4, 2, 3, 1},
		},
		{
			name: "last 3 includes min in middle",
			dat: plotter.XYs{
				{Y: 5},  // 0
				{Y: 10}, // 1 max
				{Y: 4},  // 2
				{Y: 1},  // 3 min
				{Y: 6},  // 4
			},
			// min(3), last3: 2, 3=seen skip, 4, max(1)
			want: []int{3, 2, 4, 1},
		},
		{
			name: "two points",
			dat: plotter.XYs{
				{Y: 2}, // 0 min
				{Y: 8}, // 1 max
			},
			// min(0), last3: -1 skip, 0=seen skip, 1=max skip, max(1)
			want: []int{0, 1},
		},
		{
			name: "single point: min==max",
			dat: plotter.XYs{
				{Y: 5}, // 0
			},
			// min==max no min label, last3: 0=max skip, max(0)
			want: []int{0},
		},
		{
			name: "all same values",
			dat: plotter.XYs{
				{Y: 5}, // 0
				{Y: 5}, // 1
				{Y: 5}, // 2
				{Y: 5}, // 3
				{Y: 5}, // 4
			},
			// min==max no min label, last3: 2,3,4 all =maxi(0)? no, maxi=4 (>= keeps updating)
			// Actually >= means maxi keeps updating: maxi=4. last3: 2,3,4=max skip, max(4)
			want: []int{2, 3, 4},
		},
		{
			name: "empty",
			dat:  plotter.XYs{},
			want: nil,
		},
		{
			name: "exactly 3 points",
			dat: plotter.XYs{
				{Y: 3}, // 0
				{Y: 1}, // 1 min
				{Y: 5}, // 2 max
			},
			// min(1), last3: 0, 1=seen skip, 2=max skip, max(2)
			want: []int{1, 0, 2},
		},
	}

	for _, tc := range cases {
		got := valueLabelIndices(tc.dat, 3)
		if !intsEqual(got, tc.want) {
			t.Fatalf("%s: got %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestPlotLine(t *testing.T) {
	dat := plotter.XYs{
		{X: 0, Y: 3},
		{X: 1, Y: 1},
		{X: 2, Y: 4},
		{X: 3, Y: 10},
		{X: 4, Y: 6},
		{X: 5, Y: 7},
	}

	var buf bytes.Buffer
	if err := PlotLine("Test Plot", dat, &buf,
		WithXLabel("X Axis"),
		WithYLabel("Y Axis"),
		WithLineLabel("series1"),
		WithFormat("png"),
	); err != nil {
		t.Fatalf("PlotLine failed: %v", err)
	}
	if buf.Len() == 0 {
		t.Fatal("PlotLine produced empty output")
	}

	// with X tick names
	var buf2 bytes.Buffer
	if err := PlotLine("Test Nominal", dat, &buf2,
		WithXNames([]string{"A", "B", "C", "D", "E", "F"}),
	); err != nil {
		t.Fatalf("PlotLine with XNames failed: %v", err)
	}
	if buf2.Len() == 0 {
		t.Fatal("PlotLine with XNames produced empty output")
	}

	// empty data
	var buf3 bytes.Buffer
	if err := PlotLine("Empty", plotter.XYs{}, &buf3); err != nil {
		t.Fatalf("PlotLine with empty data failed: %v", err)
	}
}

func TestValueLabelIndicesValues(t *testing.T) {
	// verify the actual Y values at the returned indices match expectations
	dat := plotter.XYs{
		{Y: 5},  // 0
		{Y: 1},  // 1 min
		{Y: 4},  // 2
		{Y: 10}, // 3 max
		{Y: 6},  // 4
		{Y: 7},  // 5
	}
	indices := valueLabelIndices(dat, 3)
	wantVals := []float64{1, 6, 7, 10} // min, last3(4,5), max
	if len(indices) != len(wantVals) {
		t.Fatalf("got %d indices, want %d", len(indices), len(wantVals))
	}
	for i, idx := range indices {
		if dat[idx].Y != wantVals[i] {
			t.Fatalf("index %d: got Y=%v, want Y=%v (indices=%v)", i, dat[idx].Y, wantVals[i], fmt.Sprint(indices))
		}
	}
}

func TestValueLabelIndicesLastN(t *testing.T) {
	dat := plotter.XYs{
		{Y: 5},  // 0
		{Y: 1},  // 1 min
		{Y: 4},  // 2
		{Y: 10}, // 3 max
		{Y: 6},  // 4
		{Y: 7},  // 5
		{Y: 8},  // 6
	}

	// lastN=1: only last point (6), skip if max
	indices := valueLabelIndices(dat, 1)
	want := []int{1, 6, 3} // min(1), last1(6), max(3)
	if !intsEqual(indices, want) {
		t.Fatalf("lastN=1: got %v, want %v", indices, want)
	}

	// lastN=5: last 5 points (2,3,4,5,6), skip max(3)
	indices = valueLabelIndices(dat, 5)
	want = []int{1, 2, 4, 5, 6, 3} // min(1), last5(2,4,5,6), max(3)
	if !intsEqual(indices, want) {
		t.Fatalf("lastN=5: got %v, want %v", indices, want)
	}

	// lastN=0: no last points
	indices = valueLabelIndices(dat, 0)
	want = []int{1, 3} // min(1), max(3)
	if !intsEqual(indices, want) {
		t.Fatalf("lastN=0: got %v, want %v", indices, want)
	}
}
