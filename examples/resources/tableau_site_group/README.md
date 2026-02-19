# Tableau Site Group Resource

This example demonstrates how to import Active Directory groups to Tableau sites using the `tableau_site_group` resource.

## Features

- Import AD groups to default or specific Tableau sites
- Configure minimum site roles and license grant modes
- Support for synchronous and asynchronous imports
- Automatic domain extraction from group names
- Site-specific authentication handling

## Usage Scenarios

### 1. Default Site Import (Synchronous)

Imports an AD group to the default site configured in the provider. The import completes synchronously.

### 2. Specific Site Import (Asynchronous)

Imports an AD group to a specific site using async mode.
The provider waits for the background job to complete before marking the resource as created.

### 3. Domain in Group Name

Demonstrates automatic domain extraction when the group name includes the domain prefix (e.g., `domain\groupname`).

## Import

To import an existing AD group that's already in Tableau:

```bash
terraform import tableau_site_group.example "GroupName:SiteIdentifier"
```

Where `SiteIdentifier` can be either:

- Site name (e.g., `prod-site`)
- Site ID (e.g., `a1b2c3d4-e5f6-7890-abcd-ef1234567890`)

## Notes

- The AD group must exist in Active Directory before import
- Async mode is recommended for large AD groups
- Site groups cannot be updated; changes require recreation
- If site is omitted, the default site from provider config is used
