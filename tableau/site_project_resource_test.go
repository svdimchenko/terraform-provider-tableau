package tableau

import (
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/resource"
)

func TestAccSiteProjectResource(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing with specific site
			{
				Config: testAccSiteProjectResourceConfig("test-project", "ManagedByOwner"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tableau_site_project.test", "name", "test-project"),
					resource.TestCheckResourceAttr("tableau_site_project.test", "content_permissions", "ManagedByOwner"),
					resource.TestCheckResourceAttrSet("tableau_site_project.test", "id"),
					resource.TestCheckResourceAttrSet("tableau_site_project.test", "site"),
					resource.TestCheckResourceAttrSet("tableau_site_project.test", "last_updated"),
				),
			},
			// ImportState testing
			{
				ResourceName:            "tableau_site_project.test",
				ImportState:             true,
				ImportStateVerify:       true,
				ImportStateVerifyIgnore: []string{"last_updated"},
			},
			// Update and Read testing
			{
				Config: testAccSiteProjectResourceConfig("test-project-updated", "LockedToProject"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tableau_site_project.test", "name", "test-project-updated"),
					resource.TestCheckResourceAttr("tableau_site_project.test", "content_permissions", "LockedToProject"),
				),
			},
		},
	})
}

func TestAccSiteProjectResourceDefaultSite(t *testing.T) {
	resource.Test(t, resource.TestCase{
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{
			// Create and Read testing with default site
			{
				Config: testAccSiteProjectResourceDefaultSiteConfig("test-project-default", "ManagedByOwner"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tableau_site_project.test_default", "name", "test-project-default"),
					resource.TestCheckResourceAttr("tableau_site_project.test_default", "content_permissions", "ManagedByOwner"),
					resource.TestCheckResourceAttrSet("tableau_site_project.test_default", "id"),
					resource.TestCheckResourceAttrSet("tableau_site_project.test_default", "last_updated"),
				),
			},
			// Update content permissions
			{
				Config: testAccSiteProjectResourceDefaultSiteConfig("test-project-default", "LockedToProject"),
				Check: resource.ComposeAggregateTestCheckFunc(
					resource.TestCheckResourceAttr("tableau_site_project.test_default", "content_permissions", "LockedToProject"),
				),
			},
		},
	})
}

func testAccSiteProjectResourceConfig(name, contentPermissions string) string {
	return fmt.Sprintf(`
resource "tableau_site" "test" {
  name = "test-site"
  content_url = "test-site"
}

resource "tableau_site_project" "test" {
  name                = %[1]q
  site                = tableau_site.test.id
  description         = "Test project"
  content_permissions = %[2]q
}
`, name, contentPermissions)
}

func testAccSiteProjectResourceDefaultSiteConfig(name, contentPermissions string) string {
	return fmt.Sprintf(`
resource "tableau_site_project" "test_default" {
  name                = %[1]q
  description         = "Test project on default site"
  content_permissions = %[2]q
}
`, name, contentPermissions)
}
