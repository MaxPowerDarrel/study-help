package highlights

import "testing"

func TestRangesOverlap(t *testing.T) {
	cases := []struct {
		name string
		a, b [4]int // {startVerse, startOffset, endVerse, endOffset}
		want bool
	}{
		{"identical", [4]int{1, 0, 1, 5}, [4]int{1, 0, 1, 5}, true},
		{"a contains b", [4]int{1, 0, 1, 10}, [4]int{1, 2, 1, 5}, true},
		{"b contains a", [4]int{1, 2, 1, 5}, [4]int{1, 0, 1, 10}, true},
		{"partial overlap same verse", [4]int{1, 0, 1, 5}, [4]int{1, 3, 1, 8}, true},
		{"adjacent same verse (boundary touch)", [4]int{1, 0, 1, 5}, [4]int{1, 5, 1, 10}, false},
		{"non-overlapping same verse", [4]int{1, 0, 1, 3}, [4]int{1, 5, 1, 8}, false},
		{"different verses no overlap", [4]int{1, 0, 1, 10}, [4]int{2, 0, 2, 10}, false},
		{"cross-verse overlap", [4]int{1, 5, 2, 5}, [4]int{2, 0, 3, 5}, true},
		{"cross-verse adjacent boundary", [4]int{1, 0, 2, 0}, [4]int{2, 0, 3, 0}, false},
		{"a fully inside b's middle verse", [4]int{2, 5, 2, 10}, [4]int{1, 0, 3, 0}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := rangesOverlap(tc.a[0], tc.a[1], tc.a[2], tc.a[3], tc.b[0], tc.b[1], tc.b[2], tc.b[3])
			if got != tc.want {
				t.Errorf("rangesOverlap(%v, %v) = %v; want %v", tc.a, tc.b, got, tc.want)
			}
			// Symmetry: swapping arguments must yield the same result.
			gotSwap := rangesOverlap(tc.b[0], tc.b[1], tc.b[2], tc.b[3], tc.a[0], tc.a[1], tc.a[2], tc.a[3])
			if gotSwap != tc.want {
				t.Errorf("rangesOverlap not symmetric: swapped = %v; want %v", gotSwap, tc.want)
			}
		})
	}
}
