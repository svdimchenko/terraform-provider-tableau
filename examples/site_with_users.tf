# Example showing site creation with automatic provider user addition
# and managing additional site users

terraform {
  required_providers {
    tableau = {
      source  = "svdimchenko/tableau"
      version = "~> 0.1"
    }
  }
}

provider "tableau" {
  server_url     = "https://your-tableau-server.com"
  server_version = "3.19"
  username       = "admin@company.com"
  password       = "your-password"
  site           = ""  # Default site for initial authentication
}

# Create a new site - provider user will be automatically added with SiteAdministratorCreator role
resource "tableau_site" "development" {
  name        = "Development Environment"
  content_url = "dev-env"
}

# Add additional users to the site
resource "tableau_site_user" "developer1" {
  name = "john.developer"
  site = tableau_site.development.id
  role = "Creator"
}

resource "tableau_site_user" "analyst1" {
  name = "jane.analyst"
  site = tableau_site.development.id
  role = "Explorer"
}

# Example of ServerAdministrator role (requires two-step creation)
resource "tableau_site_user" "site_admin" {
  name = "site.admin"
  site = tableau_site.development.id
  role = "ServerAdministrator"
}

# Output the site ID for reference
output "development_site_id" {
  value = tableau_site.development.id
}