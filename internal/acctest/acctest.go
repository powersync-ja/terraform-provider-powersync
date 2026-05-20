// Package acctest holds helpers shared by acceptance tests across packages.
// Anything in here must work when imported from `*_test.go` files compiled with TF_ACC=1.
package acctest

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/acctest"
	"github.com/powersync/terraform-provider-powersync/internal/provider"
)

// EnvAdminToken is the env var holding the PowerSync PAT used for acceptance tests.
const EnvAdminToken = "PS_PAT_TOKEN"

// EnvOrgID is the env var holding the staging organization ID under which
// acceptance tests create their resources.
const EnvOrgID = "POWERSYNC_TEST_ORG_ID"

// EnvReplicationPassword is the env var holding the password for the
// powersync_role on the source DB used by instance acceptance tests.
const EnvReplicationPassword = "TF_VAR_replication_password"

// EnvSupabaseDBHost is the env var holding the Supabase Postgres hostname used
// as the replication source for instance acceptance tests.
const EnvSupabaseDBHost = "POWERSYNC_TEST_SUPABASE_DB_HOST"

// EnvAccountsURL optionally overrides the accounts API URL. Defaults to staging.
const EnvAccountsURL = "POWERSYNC_TEST_ACCOUNTS_URL"

// EnvManagementURL optionally overrides the management API URL. Defaults to staging.
const EnvManagementURL = "POWERSYNC_TEST_MANAGEMENT_URL"

// DefaultAccountsURL is the staging accounts service URL used unless overridden.
const DefaultAccountsURL = "https://accounts.staging.powersync.com"

// DefaultManagementURL is the staging management API URL used unless overridden.
const DefaultManagementURL = "https://powersync-api.staging.journeyapps.com"

// ProviderFactories registers the in-process provider used by acceptance tests.
// Reference it from a TestCase as `ProtoV6ProviderFactories: acctest.ProviderFactories`.
var ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"powersync": providerserver.NewProtocol6WithError(provider.New("acctest")()),
}

// PreCheck fails the test fast if required env vars are not set, with a helpful
// message that names them. Call from `PreCheck: func() { acctest.PreCheck(t) }`.
func PreCheck(t *testing.T) {
	t.Helper()
	required := []string{EnvAdminToken, EnvOrgID}
	for _, k := range required {
		if os.Getenv(k) == "" {
			t.Fatalf("required env var %q is not set — acceptance tests need it; export it before running TF_ACC=1 go test", k)
		}
	}
}

// PreCheckReplication fails the test if env vars for instance/replication tests
// are missing. Use it for `TestAccInstance_*` tests in addition to `PreCheck`.
func PreCheckReplication(t *testing.T) {
	t.Helper()
	PreCheck(t)
	for _, k := range []string{EnvReplicationPassword, EnvSupabaseDBHost} {
		if os.Getenv(k) == "" {
			t.Fatalf("required env var %q is not set — instance acceptance tests need a source DB; export it before running", k)
		}
	}
}

// OrgID returns the staging organization ID for tests. Must be called after PreCheck.
func OrgID() string {
	return os.Getenv(EnvOrgID)
}

// SupabaseDBHost returns the Supabase Postgres hostname for instance tests.
func SupabaseDBHost() string {
	return os.Getenv(EnvSupabaseDBHost)
}

// AccountsURL returns the accounts URL to use in test provider blocks.
func AccountsURL() string {
	if v := os.Getenv(EnvAccountsURL); v != "" {
		return v
	}
	return DefaultAccountsURL
}

// ManagementURL returns the management URL to use in test provider blocks.
func ManagementURL() string {
	if v := os.Getenv(EnvManagementURL); v != "" {
		return v
	}
	return DefaultManagementURL
}

// RandName returns a unique resource name suffix for acceptance tests so
// concurrent runs (and crash-leak cleanup) don't collide.
func RandName(prefix string) string {
	return prefix + "-" + acctest.RandString(8)
}

// ProviderConfig is a snippet of HCL that configures the provider against the
// staging URLs. Tests prepend this to their per-case config.
func ProviderConfig() string {
	return `
provider "powersync" {
  accounts_url   = "` + AccountsURL() + `"
  management_url = "` + ManagementURL() + `"
}
`
}
