# Import existing AD group by group name and site name
terraform import tableau_site_group.example "Tableau-Analysts:prod-site"

# Import existing AD group by group name and site ID
terraform import tableau_site_group.example "Tableau-Analysts:a1b2c3d4-e5f6-7890-abcd-ef1234567890"
