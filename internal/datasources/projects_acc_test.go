package datasources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/powersync/terraform-provider-powersync/internal/acctest"
)

func TestAccProjectsDataSource_basic(t *testing.T) {
	name := acctest.RandName("tf-acc-list")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectsDataSourceConfig(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					// The list must contain the project we just created.
					// We can't assert on length/index because other projects
					// exist in the org outside of this test; instead, look the
					// created project up by name from the list using a `for`
					// expression in an output and check that.
					resource.TestCheckOutput("found_project_name", name),
				),
			},
		},
	})
}

func testAccProjectsDataSourceConfig(name string) string {
	return acctest.ProviderConfig() + fmt.Sprintf(`
resource "powersync_project" "test" {
  org_id = %q
  name   = %q
  region = "eu"
}

data "powersync_projects" "all" {
  org_id     = %q
  depends_on = [powersync_project.test]
}

output "found_project_name" {
  value = one([
    for p in data.powersync_projects.all.projects :
    p.name if p.id == powersync_project.test.id
  ])
}
`, acctest.OrgID(), name, acctest.OrgID())
}
