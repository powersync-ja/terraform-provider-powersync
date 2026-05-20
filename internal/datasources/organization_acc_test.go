package datasources_test

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/powersync/terraform-provider-powersync/internal/acctest"
)

func TestAccOrganizationDataSource_basic(t *testing.T) {
	resource.Test(t, resource.TestCase{
		PreCheck:                 func() { acctest.PreCheck(t) },
		ProtoV6ProviderFactories: acctest.ProviderFactories,
		Steps: []resource.TestStep{
			{
				Config: testAccOrganizationDataSourceConfig(acctest.OrgID()),
				Check: resource.ComposeTestCheckFunc(
					resource.TestCheckResourceAttr("data.powersync_organization.test", "id", acctest.OrgID()),
					resource.TestCheckResourceAttrSet("data.powersync_organization.test", "name"),
				),
			},
		},
	})
}

func testAccOrganizationDataSourceConfig(orgID string) string {
	return acctest.ProviderConfig() + fmt.Sprintf(`
data "powersync_organization" "test" {
  id = %q
}
`, orgID)
}
