package mapping

import (
	"slices"

	"github.com/microsoft/typescript-go/pkg/core"
)

type IgnoreDirectiveMapping struct {
	ServiceOffset uint32
	ServiceLength uint32
}

type ExpectErrorDirectiveMapping struct {
	SourceOffset  uint32
	ServiceOffset uint32
	SourceLength  uint32
	ServiceLength uint32
}

type ExpectErrorDirectiveUsage struct {
	ServiceMappings  []IgnoreDirectiveMapping
	Used             bool
	UnusedSuppressed bool
}

type DirectiveMap struct {
	IgnoreMappings      []IgnoreDirectiveMapping
	ExpectErrorMappings map[core.TextRange]ExpectErrorDirectiveUsage
}

func NewDirectiveMap(ignore []IgnoreDirectiveMapping, expectError []ExpectErrorDirectiveMapping) DirectiveMap {
	e := map[core.TextRange]ExpectErrorDirectiveUsage{}
	for _, dir := range expectError {
		id := core.NewTextRange(int(dir.SourceOffset), int(dir.SourceOffset+dir.SourceLength))
		usage, ok := e[id]
		if !ok {
			usage = ExpectErrorDirectiveUsage{}
		}
		usage.ServiceMappings = append(usage.ServiceMappings, IgnoreDirectiveMapping{
			ServiceOffset: dir.ServiceOffset,
			ServiceLength: dir.ServiceLength,
		})
		e[id] = usage
	}

	return DirectiveMap{
		IgnoreMappings:      ignore,
		ExpectErrorMappings: e,
	}
}

func (d *DirectiveMap) IsServiceRangeIgnored(serviceRange core.TextRange) bool {
	for _, mapping := range d.IgnoreMappings {
		mappingRange := core.NewTextRange(
			int(mapping.ServiceOffset),
			int(mapping.ServiceOffset+mapping.ServiceLength),
		)
		if serviceRange.ContainedBy(mappingRange) {
			return true
		}
	}

	matchedID, matchedUsage, matched := d.findExpectError(serviceRange, nil)
	if !matched {
		return false
	}

	d.markUsed(matchedID, matchedUsage)
	return true
}

func (d *DirectiveMap) ProcessExpectErrorMarker(ownerID core.TextRange, serviceRange core.TextRange) {
	ownerUsage, ok := d.ExpectErrorMappings[ownerID]
	if !ok || ownerUsage.Used || ownerUsage.UnusedSuppressed {
		return
	}

	parentID, parentUsage, matched := d.findExpectError(serviceRange, &ownerID)
	if !matched {
		return
	}

	d.markUsed(parentID, parentUsage)
	ownerUsage.UnusedSuppressed = true
	d.ExpectErrorMappings[ownerID] = ownerUsage
}

func (d *DirectiveMap) findExpectError(serviceRange core.TextRange, excludedID *core.TextRange) (core.TextRange, ExpectErrorDirectiveUsage, bool) {
	matched := false
	var matchedID core.TextRange
	var matchedUsage ExpectErrorDirectiveUsage
	var matchedServiceLength uint32
	for id, usage := range d.ExpectErrorMappings {
		if excludedID != nil && id == *excludedID {
			continue
		}
		for _, mapping := range usage.ServiceMappings {
			mappingRange := core.NewTextRange(
				int(mapping.ServiceOffset),
				int(mapping.ServiceOffset+mapping.ServiceLength),
			)
			if serviceRange.ContainedBy(mappingRange) {
				sourceLength := id.End() - id.Pos()
				matchedSourceLength := matchedID.End() - matchedID.Pos()
				if !matched ||
					mapping.ServiceLength < matchedServiceLength ||
					(mapping.ServiceLength == matchedServiceLength && sourceLength < matchedSourceLength) ||
					(mapping.ServiceLength == matchedServiceLength && sourceLength == matchedSourceLength && id.Pos() > matchedID.Pos()) {
					matched = true
					matchedID = id
					matchedUsage = usage
					matchedServiceLength = mapping.ServiceLength
				}
			}
		}
	}

	return matchedID, matchedUsage, matched
}

func (d *DirectiveMap) markUsed(id core.TextRange, usage ExpectErrorDirectiveUsage) {
	if usage.Used {
		return
	}
	usage.Used = true
	d.ExpectErrorMappings[id] = usage
}

func (d *DirectiveMap) CollectUnused() []core.TextRange {
	res := make([]core.TextRange, 0, len(d.ExpectErrorMappings))
	for id, usage := range d.ExpectErrorMappings {
		if !usage.Used && !usage.UnusedSuppressed {
			res = append(res, id)
		}
	}
	slices.SortFunc(res, func(a, b core.TextRange) int {
		if diff := a.Pos() - b.Pos(); diff != 0 {
			return diff
		}
		return a.End() - b.End()
	})

	return res
}
