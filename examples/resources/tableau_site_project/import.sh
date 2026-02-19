# Import project by project name and site name
terraform import tableau_site_project.example "Analytics Project:prod-site"

# Import project by project name and site ID
terraform import tableau_site_project.example "Analytics Project:a1b2c3d4-e5f6-7890-abcd-ef1234567890"

# Import project from the default site (omit site identifier)
terraform import tableau_site_project.default_site "Default Site Project"
