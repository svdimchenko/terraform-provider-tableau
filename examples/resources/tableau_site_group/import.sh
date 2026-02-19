# Import existing AD group by group name and site name
terraform import tableau_site_group.example "Tableau-Analysts:prod-site"

# Import existing AD group by group name and site ID
terraform import tableau_site_group.example "Tableau-Analysts:a1b2c3d4-e5f6-7890-abcd-ef1234567890"

# Import existing AD group from default site (omit site identifier)
terraform import tableau_site_group.default_site "Tableau-Analysts"
