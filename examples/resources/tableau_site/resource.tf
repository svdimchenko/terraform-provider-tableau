resource "tableau_site" "test" {
  name                = "test"
  content_url         = "Moo"
  recycle_bin_enabled = false # Default is false
}
