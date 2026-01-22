# Example: Bulk Migration
# This example shows how to migrate multiple organizations to Universal Broker

variable "orgs_to_migrate" {
  description = "List of organization IDs to migrate to Universal Broker"
  type        = list(string)
  default     = []
}

# Perform bulk migration of organizations
resource "snyk_broker_bulk_migration" "orgs" {
  count = length(var.orgs_to_migrate) > 0 ? 1 : 0

  tenant_id     = var.snyk_tenant_id
  install_id    = snyk_broker_app_install.main.install_id
  deployment_id = snyk_broker_deployment.main.id
  connection_id = snyk_broker_connection.github.id
  org_ids       = var.orgs_to_migrate
}

output "migration_status" {
  description = "Status of the bulk migration"
  value       = length(var.orgs_to_migrate) > 0 ? snyk_broker_bulk_migration.orgs[0].status : "No migration performed"
}

