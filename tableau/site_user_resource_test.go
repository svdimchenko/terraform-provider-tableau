package tableau

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccSiteUserResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing with specific site
			{
				Config: testAccSiteUserResourceConfig("test-user", "Creator"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tableau_site_user.test", "name", "test-user"),
					resource.TestCheckResourceAttr("tableau_site_user.test", "role", "Creator"),
					resource.TestCheckResourceAttrSet("tableau_site_user.test", "id"),
					resource.TestCheckResourceAttrSet("tableau_site_user.test", "site"),
					resource.TestCheckResourceAttrSet("tableau_site_user.test", "last_updated"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "tableau_site_user.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
			// Update and Read testing
			{
				Config: testAccSiteUserResourceConfig("test-user", "Explorer"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tableau_site_user.test", "role", "Explorer"),
				),
			},
		},
	})
}

func TestAccSiteUserResourceDefaultSite(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing with default site
			{
				Config: testAccSiteUserResourceDefaultSiteConfig("test-user-default", "Creator"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tableau_site_user.test_default", "name", "test-user-default"),
					resource.TestCheckResourceAttr("tableau_site_user.test_default", "role", "Creator"),
					resource.TestCheckResourceAttrSet("tableau_site_user.test_default", "id"),
					resource.TestCheckResourceAttrSet("tableau_site_user.test_default", "last_updated"),
				),
			},
			// Update role
			{
				Config: testAccSiteUserResourceDefaultSiteConfig("test-user-default", "Viewer"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tableau_site_user.test_default", "role", "Viewer"),
				),
			},
		},
	})
}

func testAccSiteUserResourceConfig(name, role string) string {
	return fmt.Sprintf(`
resource "tableau_site" "test" {
  name = "test-site"
  content_url = "test-site"
}

resource "tableau_site_user" "test" {
  name = %[1]q
  site = tableau_site.test.id
  role = %[2]q
}
`, name, role)
}

func testAccSiteUserResourceDefaultSiteConfig(name, role string) string {
	return fmt.Sprintf(`
resource "tableau_site_user" "test_default" {
  name = %[1]q
  role = %[2]q
}
`, name, role)
}
