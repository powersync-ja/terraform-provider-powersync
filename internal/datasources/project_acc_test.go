package datasources_test

import (
	"fmt"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/powersync/terraform-provider-powersync/internal/acctest"
)

func TestAccProjectDataSource_byID(t *testing.T) {
	name := acctest.RandName("tf-acc-ds")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectDataSourceConfig_byID(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("data.powersync_project.test", "name", name),
					resource.TestCheckResourceAttrPair("data.powersync_project.test", "id", "powersync_project.test", "id"),
					resource.TestCheckResourceAttr("data.powersync_project.test", "default_region", "eu"),
					resource.TestCheckResourceAttrSet("data.powersync_project.test", "vcs_mode"),
				),
			},
		},
	})
}

func TestAccProjectDataSource_byName(t *testing.T) {
	name := acctest.RandName("tf-acc-ds-name")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectDataSourceConfig_byName(name),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttrPair("data.powersync_project.test", "id", "powersync_project.test", "id"),
					resource.TestCheckResourceAttr("data.powersync_project.test", "name", name),
				),
			},
		},
	})
}

func TestAccProjectDataSource_neitherIDNorName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig() + fmt.Sprintf(`
data "powersync_project" "test" {
  org_id = %q
}`, acctest.OrgID()),
				ExpectError: regexp.MustCompile(`(?s)Exactly one of id or name`),
			},
		},
	})
}

func TestAccProjectDataSource_bothIDAndName(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: acctest.ProviderConfig() + fmt.Sprintf(`
data "powersync_project" "test" {
  org_id = %q
  id     = "some-id"
  name   = "some-name"
}`, acctest.OrgID()),
				ExpectError: regexp.MustCompile(`(?s)Exactly one of id or name`),
			},
		},
	})
}

func testAccProjectDataSourceConfig_byID(name string) string {
	return acctest.ProviderConfig() + fmt.Sprintf(`
resource "powersync_project" "test" {
  org_id = %q
  name   = %q
  region = "eu"
}

data "powersync_project" "test" {
  org_id = %q
  id     = powersync_project.test.id
}
`, acctest.OrgID(), name, acctest.OrgID())
}

func testAccProjectDataSourceConfig_byName(name string) string {
	return acctest.ProviderConfig() + fmt.Sprintf(`
resource "powersync_project" "test" {
  org_id = %q
  name   = %q
  region = "eu"
}

data "powersync_project" "test" {
  org_id     = %q
  name       = powersync_project.test.name
  depends_on = [powersync_project.test]
}
`, acctest.OrgID(), name, acctest.OrgID())
}
