package datasources_test

import (
	"context"
	"os"
	"testing"

	"github.com/powersync/terraform-provider-powersync/internal/acctest"
)

// TestMain runs before any test in this package. See acctest.Sweep for what it
// cleans up. No-op when TF_ACC is unset.
func TestMain(m *testing.M) {
	acctest.Sweep(context.Background())
	os.Exit(m.Run())
}
