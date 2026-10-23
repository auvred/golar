package mapping

import (
	"slices"

	"github.com/microsoft/TypeScript/tsc/pkg/collections"
)

type CodeRangeKey int

const (
	SourceOffsets CodeRangeKey = iota
	ServiceOffsets
)

type Mapping struct {
	SourceOffset          uint32
	ServiceOffset         uint32
	SourceLength          uint32
	ServiceLength         uint32
	SuppressedDiagnostics []uint32
}

type MappingMemo struct {
	offsets  []uint32
	mappings [][]*Mapping
}

type SourceMap struct {
	Mappings               []Mapping
	sourceCodeOffsetsMemo  *MappingMemo
	serviceCodeOffsetsMemo *MappingMemo
}

type MappedLocation struct {
	Offset  uint32
	Mapping *Mapping
}

type MappedRange struct {
	MappedStart  uint32
	MappedEnd    uint32
	StartMapping *Mapping
	EndMapping   *Mapping
}

func NewSourceMap(mappings []Mapping) *SourceMap {
	return &SourceMap{Mappings: mappings}
}

func (m *SourceMap) ToSourceRange(
	serviceStart uint32,
	serviceEnd uint32,
	fallbackToAnyMatch bool,
) []MappedRange {
	return m.findMatchingStartEnd(serviceStart, serviceEnd, fallbackToAnyMatch, ServiceOffsets, nil)
}

func (m *SourceMap) ToSourceRangeFiltered(
	serviceStart uint32,
	serviceEnd uint32,
	fallbackToAnyMatch bool,
	filter func(*Mapping) bool,
) []MappedRange {
	return m.findMatchingStartEnd(serviceStart, serviceEnd, fallbackToAnyMatch, ServiceOffsets, filter)
}

func (m *SourceMap) AnySourceRangeMatch(
	serviceStart uint32,
	serviceEnd uint32,
	fallbackToAnyMatch bool,
	filter func(*Mapping) bool,
) bool {
	if filter == nil {
		return len(m.findMatchingStartEnd(serviceStart, serviceEnd, fallbackToAnyMatch, ServiceOffsets, nil)) > 0
	}

	mappedStarts := make([]MappedLocation, 0)
	endMatches := make([]MappedLocation, 0)

	for _, mappedStart := range m.findMatchingOffsets(serviceStart, ServiceOffsets) {
		if !filter(mappedStart.Mapping) {
			continue
		}
		mappedStarts = append(mappedStarts, mappedStart)
		mapping := mappedStart.Mapping
		if _, ok := TranslateOffset(
			serviceEnd,
			mapping.ServiceOffset,
			mapping.SourceOffset,
			getLength(mapping, ServiceOffsets),
			getLength(mapping, SourceOffsets),
		); ok {
			return true
		}
	}

	if !fallbackToAnyMatch {
		return false
	}

	endMatches = m.findMatchingOffsets(serviceEnd, ServiceOffsets)
	for _, mappedStart := range mappedStarts {
		for _, mappedEnd := range endMatches {
			if mappedEnd.Offset < mappedStart.Offset {
				continue
			}
			if filter(mappedEnd.Mapping) {
				return true
			}
		}
	}

	return false
}

func (m *SourceMap) ToServiceRange(
	sourceStart uint32,
	sourceEnd uint32,
	fallbackToAnyMatch bool,
) []MappedRange {
	return m.findMatchingStartEnd(sourceStart, sourceEnd, fallbackToAnyMatch, SourceOffsets, nil)
}

func (m *SourceMap) ToSourceLocation(serviceOffset uint32) []MappedLocation {
	return m.findMatchingOffsets(serviceOffset, ServiceOffsets)
}

func (m *SourceMap) ToServiceLocation(sourceOffset uint32) []MappedLocation {
	return m.findMatchingOffsets(sourceOffset, SourceOffsets)
}

// ToNearestSourceOffset maps a service offset to the source. An offset in
// generated code that no mapping covers maps to the source end of the closest
// mapping before it, the way a source map's greatest-lower-bound lookup does,
// or to the source start of the first mapping when none comes before it.
// It reports false only when there are no mappings at all.
func (m *SourceMap) ToNearestSourceOffset(serviceOffset uint32) (uint32, bool) {
	for _, location := range m.findMatchingOffsets(serviceOffset, ServiceOffsets) {
		return location.Offset, true
	}

	var before, after *Mapping
	for i := range m.Mappings {
		mapping := &m.Mappings[i]
		serviceEnd := mapping.ServiceOffset + getLength(mapping, ServiceOffsets)
		if serviceEnd <= serviceOffset {
			if before == nil || serviceEnd > before.ServiceOffset+getLength(before, ServiceOffsets) {
				before = mapping
			}
		} else if mapping.ServiceOffset > serviceOffset {
			if after == nil || mapping.ServiceOffset < after.ServiceOffset {
				after = mapping
			}
		}
	}

	if before != nil {
		return before.SourceOffset + before.SourceLength, true
	}
	if after != nil {
		return after.SourceOffset, true
	}
	return 0, false
}

func (m *SourceMap) findMatchingOffsets(
	offset uint32,
	fromRange CodeRangeKey,
) []MappedLocation {
	memo := m.getMemoBasedOnRange(fromRange)
	if len(memo.offsets) == 0 {
		return nil
	}

	start, end, _ := BinarySearch(memo.offsets, offset)
	skip := collections.NewSetWithSizeHint[*Mapping](len(memo.mappings))
	toRange := otherRangeKey(fromRange)
	results := make([]MappedLocation, 0)

	for i := start; i <= end; i++ {
		for _, mapping := range memo.mappings[i] {
			if !skip.AddIfAbsent(mapping) {
				continue
			}
			mapped, ok := TranslateOffset(
				offset,
				getOffset(mapping, fromRange),
				getOffset(mapping, toRange),
				getLength(mapping, fromRange),
				getLength(mapping, toRange),
			)
			if ok {
				results = append(results, MappedLocation{
					Offset:  mapped,
					Mapping: mapping,
				})
			}
		}
	}

	return results
}

func (m *SourceMap) findMatchingStartEnd(
	start uint32,
	end uint32,
	fallbackToAnyMatch bool,
	fromRange CodeRangeKey,
	filter func(*Mapping) bool,
) []MappedRange {
	toRange := otherRangeKey(fromRange)
	mappedStarts := make([]MappedLocation, 0)
	results := make([]MappedRange, 0)
	hadMatch := false

	for _, mappedStart := range m.findMatchingOffsets(start, fromRange) {
		if filter != nil && !filter(mappedStart.Mapping) {
			continue
		}
		mappedStarts = append(mappedStarts, mappedStart)
		mapping := mappedStart.Mapping
		mappedEnd, ok := TranslateOffset(
			end,
			getOffset(mapping, fromRange),
			getOffset(mapping, toRange),
			getLength(mapping, fromRange),
			getLength(mapping, toRange),
		)
		if ok {
			hadMatch = true
			results = append(results, MappedRange{
				MappedStart:  mappedStart.Offset,
				MappedEnd:    mappedEnd,
				StartMapping: mapping,
				EndMapping:   mapping,
			})
		}
	}

	if !hadMatch && fallbackToAnyMatch {
		endMatches := m.findMatchingOffsets(end, fromRange)
		for _, mappedStart := range mappedStarts {
			for _, mappedEnd := range endMatches {
				if mappedEnd.Offset < mappedStart.Offset {
					continue
				}
				if filter != nil && !filter(mappedEnd.Mapping) {
					continue
				}
				results = append(results, MappedRange{
					MappedStart:  mappedStart.Offset,
					MappedEnd:    mappedEnd.Offset,
					StartMapping: mappedStart.Mapping,
					EndMapping:   mappedEnd.Mapping,
				})
				break
			}
		}
	}

	return results
}

func (m *SourceMap) getMemoBasedOnRange(fromRange CodeRangeKey) *MappingMemo {
	if fromRange == SourceOffsets {
		if m.sourceCodeOffsetsMemo == nil {
			memo := m.createMemo(SourceOffsets)
			m.sourceCodeOffsetsMemo = &memo
		}
		return m.sourceCodeOffsetsMemo
	}
	if m.serviceCodeOffsetsMemo == nil {
		memo := m.createMemo(ServiceOffsets)
		m.serviceCodeOffsetsMemo = &memo
	}
	return m.serviceCodeOffsetsMemo
}

func (m *SourceMap) createMemo(key CodeRangeKey) MappingMemo {
	offsetsSet := collections.NewSetWithSizeHint[uint32](0)
	for _, mapping := range m.Mappings {
		offset := getOffset(&mapping, key)
		offsetsSet.Add(offset)
		offsetsSet.Add(offset + getLength(&mapping, key))
	}

	offsets := make([]uint32, 0, offsetsSet.Len())
	for offset := range offsetsSet.Keys() {
		offsets = append(offsets, offset)
	}
	slices.Sort(offsets)

	mappings := make([][]*Mapping, len(offsets))
	for i := range m.Mappings {
		mapping := &m.Mappings[i]
		startOffset := getOffset(mapping, key)
		length := getLength(mapping, key)
		endOffset := startOffset + length
		_, _, startMatch := BinarySearch(offsets, startOffset)
		_, _, endMatch := BinarySearch(offsets, endOffset)
		for j := startMatch; j <= endMatch; j++ {
			mappings[j] = append(mappings[j], mapping)
		}
	}

	return MappingMemo{offsets: offsets, mappings: mappings}
}

func otherRangeKey(key CodeRangeKey) CodeRangeKey {
	if key == SourceOffsets {
		return ServiceOffsets
	}
	return SourceOffsets
}

func getOffset(mapping *Mapping, key CodeRangeKey) uint32 {
	if key == SourceOffsets {
		return mapping.SourceOffset
	}
	return mapping.ServiceOffset
}

func getLength(mapping *Mapping, key CodeRangeKey) uint32 {
	if key == SourceOffsets {
		return mapping.SourceLength
	}
	if mapping.ServiceLength > 0 {
		return mapping.ServiceLength
	}
	return mapping.SourceLength
}

func TranslateOffset(
	start uint32,
	fromOffset uint32,
	toOffset uint32,
	fromLength uint32,
	toLength uint32,
) (uint32, bool) {
	if start >= fromOffset && start <= fromOffset+fromLength {
		return toOffset + min(start-fromOffset, toLength), true
	}

	return 0, false
}

func BinarySearch(values []uint32, searchValue uint32) (low int, high int, match int) {
	if len(values) == 0 {
		return 0, -1, 0
	}

	low = 0
	high = len(values) - 1

	for low <= high {
		mid := (low + high) / 2
		midValue := values[mid]
		if midValue < searchValue {
			low = mid + 1
		} else if midValue > searchValue {
			high = mid - 1
		} else {
			low = mid
			high = mid
			match = mid
			break
		}
	}

	finalLow := max(min(low, high, len(values)-1), 0)
	finalHigh := min(max(low, high, 0), len(values)-1)

	return finalLow, finalHigh, match
}
