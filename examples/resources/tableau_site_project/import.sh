# Import project by project name and site name
terraform import tableau_site_project.example "Analytics Project:prod-site"

# Import project by project ID and site ID
terraform import tableau_site_project.example "a1b2c3d4-e5f6-7890-abcd-ef1234567890:b2c3d4e5-f6a7-8901-bcde-f12345678901"

# Import project by project name and site ID
terraform import tableau_site_project.example "Analytics Project:a1b2c3d4-e5f6-7890-abcd-ef1234567890"

# Import project from the default site (omit site identifier)
terraform import tableau_site_project.default_site "Default Site Project"

# Helper: Find project ID (luid) using SQL query on Tableau Server database
# SELECT p.name, s.name AS site_name, p.luid
# FROM projects p
# JOIN sites s ON p.site_id = s.id
