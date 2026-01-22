# Example: Using Data Sources
# This example shows how to use data sources to query existing Broker resources

# List all deployments for a tenant
data "snyk_broker_deployments" "all" {
  tenant_id = var.snyk_tenant_id
}

# List all connections for a specific deployment
data "snyk_broker_connections" "deployment" {
  tenant_id     = var.snyk_tenant_id
  install_id    = snyk_broker_app_install.main.install_id
  deployment_id = snyk_broker_deployment.main.id
}

# List connections for a specific organization
data "snyk_broker_connections_for_org" "org" {
  org_id = var.snyk_org_id
}

# List integrations using a connection
data "snyk_broker_connection_integrations" "github" {
  tenant_id     = var.snyk_tenant_id
  connection_id = snyk_broker_connection.github.id
}

# List organizations available for bulk migration
data "snyk_broker_migration_orgs" "available" {
  tenant_id     = var.snyk_tenant_id
  install_id    = snyk_broker_app_install.main.install_id
  deployment_id = snyk_broker_deployment.main.id
  connection_id = snyk_broker_connection.github.id
}

# Outputs
output "all_deployments" {
  description = "All Broker deployments"
  value       = data.snyk_broker_deployments.all.deployments
}

output "deployment_connections" {
  description = "Connections for the deployment"
  value       = data.snyk_broker_connections.deployment.connections
}

output "org_connections" {
  description = "Connections for the organization"
  value       = data.snyk_broker_connections_for_org.org.connections
}

output "connection_integrations" {
  description = "Integrations using the GitHub connection"
  value       = data.snyk_broker_connection_integrations.github.integrations
}

output "migration_eligible_orgs" {
  description = "Organizations eligible for bulk migration"
  value       = data.snyk_broker_migration_orgs.available.organizations
}

