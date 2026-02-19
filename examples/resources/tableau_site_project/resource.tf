# Create project on a specific site
resource "tableau_site_project" "example" {
  name                = "Analytics Project"
  site                = tableau_site.example.id
  description         = "Project for analytics workbooks"
  content_permissions = "ManagedByOwner"
}

# Create project on the default site (omit site attribute)
resource "tableau_site_project" "default_site" {
  name                = "Default Site Project"
  description         = "Project on default site"
  content_permissions = "LockedToProject"
}

# Create nested project with specific owner
resource "tableau_site_project" "nested" {
  name                = "Sub Project"
  site                = tableau_site.example.id
  parent_project_id   = tableau_site_project.example.id
  description         = "Nested project"
  content_permissions = "ManagedByOwner"
  owner_id            = tableau_site_user.owner.id
}
