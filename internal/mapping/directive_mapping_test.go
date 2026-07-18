package mapping

import (
	"testing"

	"github.com/microsoft/typescript-go/pkg/core"
)

// A single `@vue-expect-error` region can cover several independent diagnostics
// (e.g. a component's props and its event-handler bodies). All of them must be
// suppressed, not just the first one encountered.
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

// A directive that covers no diagnostics must still be reported as unused (TS2578).
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
