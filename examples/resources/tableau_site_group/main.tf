terraform {
  required_providers {
    tableau = {
      source  = "svdimchenko/tableau"
      version = "<version>"
    }
  }
}

provider "tableau" {
  server_url     = "https://your-tableau-server.com"
  server_version = "3.21"
  username       = "admin"
  password       = "password"
  site           = ""
}

# Import AD group to default site (synchronous)
resource "tableau_site_group" "default_site_group" {
  name               = "Tableau-Analysts"
  domain_name        = "office.company.com"
  minimum_site_role  = "Viewer"
  grant_license_mode = "onLogin"
}

# Import AD group to specific site (asynchronous)
resource "tableau_site_group" "site_group" {
  name               = "Tableau-Developers"
  site               = "dev-site"
  domain_name        = "office.company.com"
  minimum_site_role  = "Creator"
  grant_license_mode = "onLogin"
  async_mode         = true
}

# Import with domain in group name
resource "tableau_site_group" "group_with_domain" {
  name               = "office.company.com\\Tableau-Users"
  site               = "prod-site"
  minimum_site_role  = "Explorer"
  grant_license_mode = "onSync"
}
