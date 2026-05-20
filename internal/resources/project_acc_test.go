package resources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/terraform"
	"github.com/powersync/terraform-provider-powersync/internal/acctest"
)

func TestAccProject_basic(t *testing.T) {
	name := acctest.RandName("tf-acc")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectConfig(name, "eu"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("powersync_project.test", "name", name),
					resource.TestCheckResourceAttr("powersync_project.test", "region", "eu"),
					resource.TestCheckResourceAttrSet("powersync_project.test", "id"),
					resource.TestCheckResourceAttr("powersync_project.test", "org_id", acctest.OrgID()),
				),
			},
		},
	})
}

func TestAccProject_updateName(t *testing.T) {
	name1 := acctest.RandName("tf-acc")
	name2 := acctest.RandName("tf-acc-renamed")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectConfig(name1, "eu"),
				Check:  resource.TestCheckResourceAttr("powersync_project.test", "name", name1),
			},
			{
				// Same resource, different name → in-place update path.
				Config: testAccProjectConfig(name2, "eu"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("powersync_project.test", "name", name2),
					// id must NOT change (no destroy + create).
					resource.TestCheckResourceAttrSet("powersync_project.test", "id"),
				),
			},
		},
	})
}

func TestAccProject_import(t *testing.T) {
	name := acctest.RandName("tf-acc-import")
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccProjectConfig(name, "eu"),
			},
			{
				ResourceName:                         "powersync_project.test",
				ImportState:                          true,
				ImportStateVerify:                    true,
				ImportStateIdFunc:                    testAccProjectImportID,
				ImportStateVerifyIdentifierAttribute: "id",
			},
		},
	})
}

func testAccProjectImportID(state *terraform.State) (string, error) {
	res, ok := state.RootModule().Resources["powersync_project.test"]
	if !ok {
		return "", fmt.Errorf("powersync_project.test not in state")
	}
	return fmt.Sprintf("%s/%s", res.Primary.Attributes["org_id"], res.Primary.Attributes["id"]), nil
}

func testAccProjectConfig(name, region string) string {
	return acctest.ProviderConfig() + fmt.Sprintf(`
resource "powersync_project" "test" {
  org_id = %q
  name   = %q
  region = %q
}
`, acctest.OrgID(), name, region)
}
