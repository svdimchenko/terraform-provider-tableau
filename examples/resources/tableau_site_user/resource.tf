resource "tableau_site_user" "example" {
  name = "john.doe"
  site = tableau_site.example.id
  role = "Creator"
}