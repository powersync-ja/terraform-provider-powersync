package resources_test

import (
	"context"
	"os"
	"testing"

	"github.com/powersync/terraform-provider-powersync/internal/acctest"
)

// TestMain runs before any test in this package. With TF_ACC=1 set, it sweeps
// any tf-acc-* resources left over from prior runs (e.g. tests that crashed
// during destroy) before running the current suite — so each acceptance run
// starts from a clean slate.
//
// Without TF_ACC=1, Sweep is a no-op; this is just a passthrough.
func TestMain(m *testing.M) {
	acctest.Sweep(context.Background())
	os.Exit(m.Run())
}
