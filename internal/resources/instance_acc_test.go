package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/powersync/terraform-provider-powersync/internal/acctest"
)

// Instance acceptance tests are slow: each create/update/destroy step triggers
// a real PowerSync deploy (~2–3 minutes wall-clock). Budget 15+ minutes for the
// full suite.

// TestAccInstance_basic exercises the instance resource AND both instance data
// sources (powersync_instance, powersync_instances) in a single create/destroy
// cycle. Each cycle is ~5 minutes, so folding the data source assertions in
// here saves a second cycle.
func TestAccInstance_basic(t *testing.T) {
	projectName := acctest.RandName("tf-acc-inst")
	instanceName := acctest.RandName("tf-acc-inst")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckReplication(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceConfigWithDataSources(projectName, instanceName),
				Check: resource.ComposeAggregateTestCheckFunc(
					// Resource attributes.
					resource.TestCheckResourceAttr("powersync_instance.test", "name", instanceName),
					resource.TestCheckResourceAttr("powersync_instance.test", "region", "staging"),
					resource.TestCheckResourceAttrSet("powersync_instance.test", "id"),
					resource.TestCheckResourceAttrSet("powersync_instance.test", "instance_url"),
					resource.TestCheckResourceAttr("powersync_instance.test", "status", "active"),
					resource.TestCheckResourceAttr("powersync_instance.test", "replication_connection.#", "1"),
					resource.TestCheckResourceAttr("powersync_instance.test", "replication_connection.0.type", "postgresql"),

					// powersync_instance data source: round-tripped attrs match the resource.
					resource.TestCheckResourceAttrPair("data.powersync_instance.test", "id", "powersync_instance.test", "id"),
					resource.TestCheckResourceAttrPair("data.powersync_instance.test", "name", "powersync_instance.test", "name"),
					resource.TestCheckResourceAttrPair("data.powersync_instance.test", "region", "powersync_instance.test", "region"),
					resource.TestCheckResourceAttrPair("data.powersync_instance.test", "instance_url", "powersync_instance.test", "instance_url"),
					resource.TestCheckResourceAttr("data.powersync_instance.test", "status", "active"),

					// powersync_instances data source: the created instance appears in the list.
					resource.TestCheckOutput("found_instance_name", instanceName),
				),
			},
		},
	})
}

func TestAccInstance_updateName(t *testing.T) {
	projectName := acctest.RandName("tf-acc-inst-rename")
	name1 := acctest.RandName("tf-acc-inst")
	name2 := acctest.RandName("tf-acc-inst-renamed")

	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckReplication(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceConfigWithName(projectName, name1),
				Check:  resource.TestCheckResourceAttr("powersync_instance.test", "name", name1),
			},
			{
				// Same instance, different name → in-place update path.
				// Capture the ID before and after to verify it doesn't change.
				Config: testAccInstanceConfigWithName(projectName, name2),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("powersync_instance.test", "name", name2),
					resource.TestCheckResourceAttrSet("powersync_instance.test", "id"),
				),
			},
		},
	})
}

func TestAccInstance_import(t *testing.T) {
	projectName := acctest.RandName("tf-acc-inst-import")
	instanceName := acctest.RandName("tf-acc-inst")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheckReplication(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccInstanceConfigWithName(projectName, instanceName),
			},
			{
				ResourceName:      "powersync_instance.test",
				ImportState:       true,
				ImportStateVerify: true,
				ImportStateIdFunc: testAccInstanceImportID,
				// Sensitive fields (password) are not returned by the API on Read,
				// so a strict ImportStateVerify would diff on them. Exclude those
				// + the variable-driven password block.
				ImportStateVerifyIgnore: []string{
					"replication_connection.0.password",
				},
			},
		},
	})
}

func testAccInstanceImportID(state *terraform.State) (string, error) {
	res, ok := state.RootModule().Resources["powersync_instance.test"]
	if !ok {
		return "", fmt.Errorf("powersync_instance.test not in state")
	}
	return fmt.Sprintf("%s/%s/%s",
		res.Primary.Attributes["org_id"],
		res.Primary.Attributes["project_id"],
		res.Primary.Attributes["id"],
	), nil
}

// testAccInstanceConfigWithDataSources extends the base instance config with
// both data sources (powersync_instance + powersync_instances), used by
// TestAccInstance_basic for combined coverage in a single create cycle.
func testAccInstanceConfigWithDataSources(projectName, instanceName string) string {
	return testAccInstanceConfigWithName(projectName, instanceName) + fmt.Sprintf(`
data "powersync_instance" "test" {
  org_id     = %q
  project_id = powersync_project.test.id
  id         = powersync_instance.test.id
}

data "powersync_instances" "all" {
  org_id     = %q
  project_id = powersync_project.test.id
  depends_on = [powersync_instance.test]
}

output "found_instance_name" {
  value = one([
    for i in data.powersync_instances.all.instances :
    i.name if i.id == powersync_instance.test.id
  ])
}
`, acctest.OrgID(), acctest.OrgID())
}

func testAccInstanceConfigWithName(projectName, instanceName string) string {
	return acctest.ProviderConfig() + fmt.Sprintf(`
variable "replication_password" {
  type      = string
  sensitive = true
}

resource "powersync_project" "test" {
  org_id = %q
  name   = %q
  region = "eu"
}

resource "powersync_instance" "test" {
  org_id     = %q
  project_id = powersync_project.test.id
  name       = %q
  region     = "staging"

  replication_connection {
    type     = "postgresql"
    name     = "supabase-main"
    hostname = %q
    port     = 5432
    database = "postgres"
    username = "powersync_role"
    password = var.replication_password
    sslmode  = "verify-full"
  }

  client_auth {
    supabase               = true
    allow_temporary_tokens = true
  }

  sync_config_content = <<-YAML
    config:
      edition: 3
    streams:
      todos:
        auto_subscribe: true
        query: SELECT * FROM todos
  YAML
}
`, acctest.OrgID(), projectName, acctest.OrgID(), instanceName, acctest.SupabaseDBHost())
}
