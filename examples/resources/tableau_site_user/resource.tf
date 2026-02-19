# Import user from Active Directory to a specific site
resource "tableau_site_user" "example" {
  name = "john.doe"
  site = tableau_site.example.id
  role = "Creator"
}

# Import user to the default site (omit site attribute)
resource "tableau_site_user" "default_site" {
  name = "jane.smith"
  role = "Explorer"
}

# Import user with ServerAdministrator role
resource "tableau_site_user" "admin" {
  name = "admin.user"
  site = tableau_site.example.id
  role = "ServerAdministrator"
}
