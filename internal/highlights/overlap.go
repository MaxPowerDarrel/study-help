package highlights

// rangesOverlap reports whether two character ranges over (verse, offset)
// share any position. Both ranges are half-open: [start, end). This
// matches the Selection API's Range convention and means highlights
// touching at a single boundary (e.g. one ending at offset 10 and
// another starting at offset 10 in the same verse) do not conflict.
func rangesOverlap(aStartV, aStartO, aEndV, aEndO, bStartV, bStartO, bEndV, bEndO int) bool {
	return less(aStartV, aStartO, bEndV, bEndO) && less(bStartV, bStartO, aEndV, aEndO)
}

func less(v1, o1, v2, o2 int) bool {
	if v1 != v2 {
		return v1 < v2
	}
	return o1 < o2
}
