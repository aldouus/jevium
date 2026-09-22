package agent

import (
	"github.com/aldous/jevium/internal/page"
	"reflect"
	"strings"
)

// Chrome's first page-key component is performance.timeOrigin. Native snapshots
// lack a document identity, so reused accessibility IDs cannot prove continuity.
func sameDocument(before, after page.Page) bool {
	if before.URL != after.URL || before.BundleID != after.BundleID || strings.HasPrefix(before.URL, "app://") {
		return false
	}
	a, aok := before.PageKey.([]any)
	b, bok := after.PageKey.([]any)
	return aok && bok && len(a) > 0 && len(b) > 0 && a[0] != nil && reflect.DeepEqual(a[0], b[0])
}
