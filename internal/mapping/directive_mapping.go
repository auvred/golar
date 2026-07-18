package mapping

import "github.com/microsoft/typescript-go/pkg/core"

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
	ServiceMappings []IgnoreDirectiveMapping
	Used            bool
}

type DirectiveMap struct {
	IgnoreMappings      []IgnoreDirectiveMapping
	ExpectErrorMappings map[core.TextRange]ExpectErrorDirectiveUsage
	Used                int
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
	result := false
	for _, mapping := range d.IgnoreMappings {
		mappingRange := core.NewTextRange(
			int(mapping.ServiceOffset),
			int(mapping.ServiceOffset+mapping.ServiceLength),
		)
		if serviceRange.ContainedBy(mappingRange) {
			result = true
			break
		}
	}

	for id, usage := range d.ExpectErrorMappings {
		matched := false
		for _, mapping := range usage.ServiceMappings {
			mappingRange := core.NewTextRange(
				int(mapping.ServiceOffset),
				int(mapping.ServiceOffset+mapping.ServiceLength),
			)
			if serviceRange.ContainedBy(mappingRange) {
				matched = true
				break
			}
		}
		if matched {
			result = true
			// A `@vue-expect-error` region can cover several independent diagnostics
			// (e.g. a component's props and its event-handler bodies). It must suppress
			// all of them, not just the first, so ranges already marked used still
			// suppress further diagnostics. `Used` only decides whether the directive
			// is unused (TS2578), so the counter is incremented once per directive.
			if !usage.Used {
				usage.Used = true
				d.Used++
				d.ExpectErrorMappings[id] = usage
			}
			break
		}
	}

	return result
}

func (d *DirectiveMap) CollectUnused() []core.TextRange {
	if d.Used == len(d.ExpectErrorMappings) {
		return nil
	}
	res := make([]core.TextRange, 0, len(d.ExpectErrorMappings)-d.Used)
	for id, usage := range d.ExpectErrorMappings {
		if !usage.Used {
			res = append(res, id)
		}
	}

	return res
}
