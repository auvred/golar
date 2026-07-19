package mapping

import (
	"testing"

	"github.com/microsoft/typescript-go/pkg/core"
)

func TestExpectErrorSuppressesMultipleDiagnosticsInRegion(t *testing.T) {
	dm := NewDirectiveMap(nil, []ExpectErrorDirectiveMapping{
		{SourceOffset: 0, SourceLength: 10, ServiceOffset: 100, ServiceLength: 100},
	})

	first := core.NewTextRange(110, 115)
	second := core.NewTextRange(150, 160)

	if !dm.IsServiceRangeIgnored(first) {
		t.Fatal("first diagnostic in the region should be suppressed")
	}
	if !dm.IsServiceRangeIgnored(second) {
		t.Fatal("second diagnostic in the same region should also be suppressed (regression: only the first was)")
	}
	if unused := dm.CollectUnused(); len(unused) != 0 {
		t.Fatalf("directive consumed diagnostics, expected 0 unused, got %d", len(unused))
	}
}

func TestExpectErrorReportedUnusedWhenRegionHasNoDiagnostic(t *testing.T) {
	dm := NewDirectiveMap(nil, []ExpectErrorDirectiveMapping{
		{SourceOffset: 0, SourceLength: 10, ServiceOffset: 100, ServiceLength: 100},
	})

	if dm.IsServiceRangeIgnored(core.NewTextRange(300, 310)) {
		t.Fatal("diagnostic outside the region should not be suppressed")
	}
	if unused := dm.CollectUnused(); len(unused) != 1 {
		t.Fatalf("directive matched nothing, expected 1 unused, got %d", len(unused))
	}
}

func TestIgnoredRangeDoesNotConsumeExpectError(t *testing.T) {
	dm := NewDirectiveMap(
		[]IgnoreDirectiveMapping{{ServiceOffset: 100, ServiceLength: 100}},
		[]ExpectErrorDirectiveMapping{{SourceOffset: 0, SourceLength: 10, ServiceOffset: 100, ServiceLength: 100}},
	)

	if !dm.IsServiceRangeIgnored(core.NewTextRange(110, 120)) {
		t.Fatal("diagnostic in an ignored generated range should be suppressed")
	}
	if unused := dm.CollectUnused(); len(unused) != 1 {
		t.Fatalf("ignored diagnostic should not consume the directive, expected 1 unused, got %d", len(unused))
	}
}

func TestMostSpecificExpectErrorOwnsOverlappingDiagnostic(t *testing.T) {
	outer := core.NewTextRange(0, 10)
	dm := NewDirectiveMap(nil, []ExpectErrorDirectiveMapping{
		{SourceOffset: 0, SourceLength: 10, ServiceOffset: 100, ServiceLength: 100},
		{SourceOffset: 20, SourceLength: 10, ServiceOffset: 125, ServiceLength: 20},
	})

	if !dm.IsServiceRangeIgnored(core.NewTextRange(130, 135)) {
		t.Fatal("diagnostic in overlapping directive regions should be suppressed")
	}
	unused := dm.CollectUnused()
	if len(unused) != 1 || unused[0] != outer {
		t.Fatalf("inner directive should own the diagnostic, expected only outer directive unused, got %v", unused)
	}
}

func TestUsedInnerExpectErrorMarkerDoesNotConsumeOuter(t *testing.T) {
	outer := core.NewTextRange(0, 10)
	inner := core.NewTextRange(20, 30)
	dm := NewDirectiveMap(nil, []ExpectErrorDirectiveMapping{
		{SourceOffset: 0, SourceLength: 10, ServiceOffset: 200, ServiceLength: 20},
		{SourceOffset: 20, SourceLength: 10, ServiceOffset: 100, ServiceLength: 20},
	})

	dm.IsServiceRangeIgnored(core.NewTextRange(105, 110))
	dm.ProcessExpectErrorMarker(inner, core.NewTextRange(205, 210))

	unused := dm.CollectUnused()
	if len(unused) != 1 || unused[0] != outer {
		t.Fatalf("used inner marker should not consume outer directive, expected only outer unused, got %v", unused)
	}
}

func TestUnusedInnerExpectErrorMarkerConsumesOuter(t *testing.T) {
	inner := core.NewTextRange(20, 30)
	dm := NewDirectiveMap(nil, []ExpectErrorDirectiveMapping{
		{SourceOffset: 0, SourceLength: 10, ServiceOffset: 200, ServiceLength: 20},
		{SourceOffset: 20, SourceLength: 10, ServiceOffset: 100, ServiceLength: 20},
	})

	dm.ProcessExpectErrorMarker(inner, core.NewTextRange(205, 210))

	if unused := dm.CollectUnused(); len(unused) != 0 {
		t.Fatalf("unused inner marker should be suppressed by and consume outer directive, got %v", unused)
	}
}

func TestTopLevelExpectErrorMarkerRemainsUnused(t *testing.T) {
	owner := core.NewTextRange(0, 10)
	dm := NewDirectiveMap(nil, []ExpectErrorDirectiveMapping{
		{SourceOffset: 0, SourceLength: 10, ServiceOffset: 100, ServiceLength: 20},
	})

	dm.ProcessExpectErrorMarker(owner, core.NewTextRange(200, 210))

	unused := dm.CollectUnused()
	if len(unused) != 1 || unused[0] != owner {
		t.Fatalf("top-level marker should leave its directive unused, got %v", unused)
	}
}
