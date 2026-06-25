package acctest

import (
	"context"
	"log"
	"os"
	"strings"
	"time"

	"github.com/powersync/terraform-provider-powersync/internal/client"
)

// SweepPrefix is the resource-name prefix that marks a resource as test-created
// and safe to sweep. Every acceptance test resource must use a name beginning
// with this prefix (via RandName) so the sweeper can identify it.
const SweepPrefix = "tf-acc"

// Sweep finds and destroys all PowerSync resources under the test org whose
// names start with SweepPrefix. It logs progress and continues on per-resource
// failures so a single broken project doesn't strand the rest.
//
// Sweep is best-effort: failures during sweep are logged, never fatal. The next
// sweep run will catch anything that escaped. This is the canonical pattern
// from HashiCorp's official providers (see e.g. terraform-provider-aws).
//
// The function is a no-op when TF_ACC is unset or the required env vars are
// missing — `go test` without acceptance flags must not touch any API.
func Sweep(ctx context.Context) {
	if os.Getenv("TF_ACC") != "1" {
		return
	}
	token := os.Getenv(EnvAdminToken)
	orgID := os.Getenv(EnvOrgID)
	if token == "" || orgID == "" {
		// Pre-checks in individual tests will surface the missing env vars
		// with a clear message; nothing to sweep against.
		return
	}

	c := client.New(AccountsURL(), ManagementURL(), token)

	projects, _, err := c.ListProjects(ctx, orgID)
	if err != nil {
		log.Printf("[sweep] failed to list projects: %v", err)
		return
	}

	swept := 0
	for _, p := range projects {
		if !strings.HasPrefix(p.Name, SweepPrefix) {
			continue
		}
		log.Printf("[sweep] cleaning project %q (%s)", p.Name, p.ID)
		sweepProject(ctx, c, orgID, p.ID, p.Name)
		swept++
	}
	if swept == 0 {
		log.Printf("[sweep] no leaked %s-* resources found", SweepPrefix)
	} else {
		log.Printf("[sweep] processed %d leaked project(s)", swept)
	}
}

// sweepProject destroys every instance under the project, waits for the
// destroys to complete, then deletes the project itself.
func sweepProject(ctx context.Context, c *client.Client, orgID, projectID, projectName string) {
	instances, err := c.ListInstances(ctx, orgID, projectID)
	if err != nil {
		log.Printf("[sweep]   list instances in %s failed: %v", projectID, err)
		// Best effort — try to delete the project anyway; it'll fail if there
		// are still instances under it, which we'll log.
	}

	for _, inst := range instances {
		log.Printf("[sweep]   destroying instance %q (%s)", inst.Name, inst.ID)
		opID, err := c.DestroyInstance(ctx, orgID, projectID, inst.ID)
		if err != nil {
			log.Printf("[sweep]     destroy call failed: %v", err)
			continue
		}
		// Short timeout — sweep should not hold up the test run.
		waitCtx, cancel := context.WithTimeout(ctx, 5*time.Minute)
		if err := c.WaitForOperation(waitCtx, orgID, projectID, inst.ID, opID, 5*time.Minute); err != nil {
			log.Printf("[sweep]     destroy operation didn't complete in 5m: %v", err)
		}
		cancel()
	}

	if err := c.DeleteProject(ctx, orgID, projectID); err != nil {
		log.Printf("[sweep]   delete project %s failed: %v", projectID, err)
		return
	}
	log.Printf("[sweep]   deleted project %s", projectID)
}
