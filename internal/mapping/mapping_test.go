package mapping

import "testing"

func TestBinarySearch(t *testing.T) {
	cases := []struct {
		name      string
		values    []uint32
		search    uint32
		wantLow   int
		wantHigh  int
		wantMatch *int
	}{
		{
			name:     "value between elements",
			values:   []uint32{1, 3, 5, 7, 9},
			search:   4,
			wantLow:  1,
			wantHigh: 2,
		},
		{
			name:     "value less than first element",
			values:   []uint32{1, 3, 5, 7, 9},
			search:   0,
			wantLow:  0,
			wantHigh: 0,
		},
		{
			name:     "value greater than last element",
			values:   []uint32{1, 3, 5, 7, 9},
			search:   10,
			wantLow:  4,
			wantHigh: 4,
		},
		{
			name:     "empty array",
			values:   []uint32{},
			search:   1,
			wantLow:  0,
			wantHigh: -1,
		},
		{
			name:      "value at start of array",
			values:    []uint32{1, 3, 5, 7, 9},
			search:    1,
			wantLow:   0,
			wantHigh:  0,
			wantMatch: intPtr(0),
		},
		{
			name:      "value at end of array",
			values:    []uint32{1, 3, 5, 7, 9},
			search:    9,
			wantLow:   4,
			wantHigh:  4,
			wantMatch: intPtr(4),
		},
		{
			name:      "single element array, value matches",
			values:    []uint32{1},
			search:    1,
			wantLow:   0,
			wantHigh:  0,
			wantMatch: intPtr(0),
		},
		{
			name:     "single element array, value does not match",
			values:   []uint32{1},
			search:   2,
			wantLow:  0,
			wantHigh: 0,
		},
		{
			name:      "two elements array, value matches first",
			values:    []uint32{1, 2},
			search:    1,
			wantLow:   0,
			wantHigh:  0,
			wantMatch: intPtr(0),
		},
		{
			name:      "two elements array, value matches second",
			values:    []uint32{1, 2},
			search:    2,
			wantLow:   1,
			wantHigh:  1,
			wantMatch: intPtr(1),
		},
		{
			name:     "two elements array, value does not match",
			values:   []uint32{1, 2},
			search:   3,
			wantLow:  1,
			wantHigh: 1,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertBinarySearch(t, tc.values, tc.search, tc.wantLow, tc.wantHigh, tc.wantMatch)
		})
	}
}

func TestTranslateOffset(t *testing.T) {
	cases := []struct {
		name        string
		start       uint32
		fromOffset  uint32
		toOffset    uint32
		fromLength  uint32
		toLength    uint32
		want        uint32
		wantOk      bool
	}{
		{
			name:        "start within fromRange, offset within toRange",
			start:       5,
			fromOffset: 1,
			toOffset:   11,
			fromLength: 9,
			toLength: 9,
			want:        15,
			wantOk:      true,
		},
		{
			name:        "start outside fromRange",
			start:       0,
			fromOffset: 1,
			toOffset:   11,
			fromLength: 9,
			toLength: 9,
			want:        0,
			wantOk:      false,
		},
		{
			name:        "start at end of fromRange with shorter toLength",
			start:       10,
			fromOffset: 1,
			toOffset:   11,
			fromLength: 9,
			toLength:   7,
			want:        18,
			wantOk:      true,
		},
		{
			name:        "uses fromLengths when toLengths is empty",
			start:       3,
			fromOffset: 1,
			toOffset:   11,
			fromLength: 4,
			toLength: 4,
			want:        13,
			wantOk:      true,
		},
		{
			name:        "start equals fromOffset",
			start:       10,
			fromOffset: 10,
			toOffset:   50,
			fromLength: 2,
			toLength: 2,
			want:        50,
			wantOk:      true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			assertTranslateOffset(t, tc.start, tc.fromOffset, tc.toOffset, tc.fromLength, tc.toLength, tc.want, tc.wantOk)
		})
	}
}

func TestSourceMapLocations(t *testing.T) {
	mapping1 := Mapping{
		SourceOffset:  0,
		ServiceOffset: 100,
		SourceLength:  5,
	}
	mapping2 := Mapping{
		SourceOffset:  10,
		ServiceOffset: 110,
		SourceLength:  5,
	}
	overlapMapping := Mapping{
		SourceOffset:  0,
		ServiceOffset: 100,
		SourceLength:  10,
	}

	cases := []struct {
		name      string
		mappings  []Mapping
		toSource  bool
		offset    uint32
		want      []uint32
		wantEmpty bool
	}{
		{
			name:      "empty mappings",
			mappings:  nil,
			toSource:  true,
			offset:    10,
			wantEmpty: true,
		},
		{
			name:     "to source location",
			mappings: []Mapping{mapping1, mapping2},
			toSource: true,
			offset:   102,
			want:     []uint32{2},
		},
		{
			name:     "to service location",
			mappings: []Mapping{mapping1, mapping2},
			toSource: false,
			offset:   12,
			want:     []uint32{112},
		},
		{
			name:     "dedupes mapping across memo buckets",
			mappings: []Mapping{overlapMapping},
			toSource: true,
			offset:   105,
			want:     []uint32{5},
		},
		{
			name:     "no matching location",
			mappings: []Mapping{mapping1, mapping2},
			toSource: false,
			offset:   99,
			want:     nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sourceMap := NewSourceMap(tc.mappings)
			var locations []MappedLocation
			if tc.toSource {
				locations = sourceMap.ToSourceLocation(tc.offset)
			} else {
				locations = sourceMap.ToServiceLocation(tc.offset)
			}

			if tc.wantEmpty {
				if len(locations) != 0 {
					t.Fatalf("expected no locations, got %d", len(locations))
				}
				return
			}
			assertLocationsSet(t, locations, tc.want)
		})
	}
}

func TestSourceMapRanges(t *testing.T) {
	basicMapping := Mapping{
		SourceOffset:  0,
		ServiceOffset: 100,
		SourceLength:  5,
	}
	multiSegment1 := Mapping{
		SourceOffset:  0,
		ServiceOffset: 100,
		SourceLength:  5,
	}
	multiSegment2 := Mapping{
		SourceOffset:  20,
		ServiceOffset: 200,
		SourceLength:  5,
	}
	serviceLengthMapping := Mapping{
		SourceOffset:  0,
		ServiceOffset: 100,
		SourceLength:  10,
		ServiceLength: 5,
	}
	mappingA := Mapping{
		SourceOffset:  0,
		ServiceOffset: 100,
		SourceLength:  5,
	}
	mappingB := Mapping{
		SourceOffset:  10,
		ServiceOffset: 200,
		SourceLength:  5,
	}
	reversedStart := Mapping{
		SourceOffset:  0,
		ServiceOffset: 200,
		SourceLength:  5,
	}
	reversedEnd := Mapping{
		SourceOffset:  10,
		ServiceOffset: 100,
		SourceLength:  5,
	}

	cases := []struct {
		name     string
		mappings []Mapping
		toSource bool
		start    uint32
		end      uint32
		fallback bool
		want     [][2]uint32
	}{
		{
			name:     "direct mapping to source",
			mappings: []Mapping{basicMapping},
			toSource: true,
			start:    100,
			end:      105,
			fallback: false,
			want:     [][2]uint32{{0, 5}},
		},
		{
			name:     "direct mapping to service",
			mappings: []Mapping{basicMapping},
			toSource: false,
			start:    0,
			end:      5,
			fallback: false,
			want:     [][2]uint32{{100, 105}},
		},
		{
			name:     "start in segment, end outside without fallback",
			mappings: []Mapping{basicMapping},
			toSource: false,
			start:    2,
			end:      12,
			fallback: false,
			want:     nil,
		},
		{
			name:     "fallback maps across mappings",
			mappings: []Mapping{mappingA, mappingB},
			toSource: false,
			start:    2,
			end:      12,
			fallback: true,
			want:     [][2]uint32{{102, 202}},
		},
		{
			name:     "fallback skips reversed range",
			mappings: []Mapping{reversedStart, reversedEnd},
			toSource: false,
			start:    2,
			end:      12,
			fallback: true,
			want:     nil,
		},
		{
			name:     "service lengths constrain mapped end",
			mappings: []Mapping{serviceLengthMapping},
			toSource: false,
			start:    2,
			end:      8,
			fallback: false,
			want:     [][2]uint32{{102, 105}},
		},
		{
			name:     "multiple segments in one mapping",
			mappings: []Mapping{multiSegment1, multiSegment2},
			toSource: false,
			start:    22,
			end:      24,
			fallback: false,
			want:     [][2]uint32{{202, 204}},
		},
		{
			name:     "no mappings",
			mappings: nil,
			toSource: true,
			start:    0,
			end:      1,
			fallback: true,
			want:     nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sourceMap := NewSourceMap(tc.mappings)
			var ranges []MappedRange
			if tc.toSource {
				ranges = sourceMap.ToSourceRange(tc.start, tc.end, tc.fallback)
			} else {
				ranges = sourceMap.ToServiceRange(tc.start, tc.end, tc.fallback)
			}
			assertRangesSet(t, ranges, tc.want)
		})
	}
}

func assertBinarySearch(t *testing.T, values []uint32, search uint32, wantLow int, wantHigh int, wantMatch *int) {
	t.Helper()

	low, high, match := BinarySearch(values, search)
	if low != wantLow || high != wantHigh {
		t.Fatalf("expected low=%d high=%d, got low=%d high=%d", wantLow, wantHigh, low, high)
	}

	if wantMatch == nil {
		return
	}

	if match != *wantMatch {
		t.Fatalf("expected match index %d, got match=%d", *wantMatch, match)
	}
}

func assertTranslateOffset(
	t *testing.T,
	start uint32,
	fromOffset uint32,
	toOffset uint32,
	fromLength uint32,
	toLength uint32,
	want uint32,
	wantOk bool,
) {
	t.Helper()

	got, ok := TranslateOffset(start, fromOffset, toOffset, fromLength, toLength)
	if ok != wantOk || (ok && got != want) {
		t.Fatalf("expected ok=%v offset=%d, got ok=%v offset=%d", wantOk, want, ok, got)
	}
}

func assertLocationsSet(t *testing.T, got []MappedLocation, want []uint32) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %d locations, got %d", len(want), len(got))
	}
	seen := make([]bool, len(want))
	for _, item := range got {
		matched := false
		for i, w := range want {
			if seen[i] {
				continue
			}
			if item.Offset == w {
				seen[i] = true
				matched = true
				break
			}
		}
		if !matched {
			t.Fatalf("unexpected location offset %d", item.Offset)
		}
	}
}

func assertRangesSet(t *testing.T, got []MappedRange, want [][2]uint32) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("expected %d ranges, got %d", len(want), len(got))
	}
	seen := make([]bool, len(want))
	for _, item := range got {
		matched := false
		for i, w := range want {
			if seen[i] {
				continue
			}
			if item.MappedStart == w[0] && item.MappedEnd == w[1] {
				seen[i] = true
				matched = true
				break
			}
		}
		if !matched {
			t.Fatalf("unexpected range [%d, %d]", item.MappedStart, item.MappedEnd)
		}
	}
}

func intPtr(value int) *int {
	return &value
}
