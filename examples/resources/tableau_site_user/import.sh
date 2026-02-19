# Site users can be imported using the format "username:siteID" or "username:siteName"
terraform import tableau_site_user.example "john.doe:site-id-123"
terraform import tableau_site_user.example "john.doe:My Site Name"

# Import user from the default site (omit site identifier)
terraform import tableau_site_user.default_site "jane.smith"
