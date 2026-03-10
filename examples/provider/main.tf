# Example: Complete Snyk Broker Setup
# This example shows how to set up a complete Snyk Broker deployment with GitHub Server App integration

terraform {
  required_providers {
    snyk = {
      source  = "snyk/snyk-broker"
      version = "~> 0.1"
    }
  }
}

# Configure the Snyk provider
# Authentication can be via API token or OAuth (service account)
provider "snyk" {
  # Option 1: API Token authentication
  # api_token = var.snyk_token

  # Option 2: OAuth authentication (recommended for automation)
  client_id     = var.snyk_client_id
  client_secret = var.snyk_client_secret

  # Region: us (default), eu, or au
  region = var.snyk_region
}

# Variables
variable "snyk_client_id" {
  description = "Snyk OAuth client ID"
  type        = string
  sensitive   = true
}

variable "snyk_client_secret" {
  description = "Snyk OAuth client secret"
  type        = string
  sensitive   = true
}

variable "snyk_region" {
  description = "Snyk region (us, eu, au)"
  type        = string
  default     = "us"
}

variable "snyk_org_id" {
  description = "Snyk organization ID"
  type        = string
}

variable "snyk_tenant_id" {
  description = "Snyk tenant ID"
  type        = string
}

variable "snyk_broker_app_id" {
  description = "Snyk Broker App ID (region-specific)"
  type        = string
}

variable "github_integration_id" {
  description = "GitHub integration ID in the Snyk organization"
  type        = string
}

variable "github_app_installation_id" {
  description = "GitHub Server App installation ID"
  type        = string
}

variable "github_app_id" {
  description = "GitHub Server App ID"
  type        = string
}

variable "github_app_client_id" {
  description = "GitHub Server App Client ID"
  type        = string
}

variable "github_server_url" {
  description = "GitHub Server URL (e.g., https://github.example.com)"
  type        = string
}

variable "github_api_url" {
  description = "GitHub Server API URL (e.g., https://github.example.com/api/v3)"
  type        = string
}

variable "broker_client_url" {
  description = "Broker client URL (e.g., https://broker.example.com:8000)"
  type        = string
}

# Step 1: Install the Snyk Broker App
resource "snyk_broker_app_install" "main" {
  org_id = var.snyk_org_id
  app_id = var.snyk_broker_app_id
}

# Step 2: Create a Broker Deployment
resource "snyk_broker_deployment" "main" {
  tenant_id  = var.snyk_tenant_id
  install_id = snyk_broker_app_install.main.install_id
  org_id     = var.snyk_org_id
  name       = "Production Broker"

  metadata = {
    cluster     = "us-east-1"
    environment = "production"
  }
}

# Step 3: Create a Credential Reference
# This tells the Broker which environment variable will contain the GitHub App private key PEM file path
resource "snyk_broker_credential" "github" {
  tenant_id                   = var.snyk_tenant_id
  install_id                  = snyk_broker_app_install.main.install_id
  deployment_id               = snyk_broker_deployment.main.id
  environment_variable_name   = "GITHUB_APP_PRIVATE_PEM_PATH"
  type                        = "github-server-app"
  comment                     = "GitHub Server App Private Key PEM Path for Broker"
}

# Step 4: Create a Broker Connection
resource "snyk_broker_connection" "github" {
  tenant_id     = var.snyk_tenant_id
  install_id    = snyk_broker_app_install.main.install_id
  deployment_id = snyk_broker_deployment.main.id
  name          = "GitHub Server App"
  type          = "github-server-app"

  configuration = {
    broker_client_url           = var.broker_client_url
    github                      = var.github_server_url
    github_api                  = var.github_api_url
    github_app_client_id        = var.github_app_client_id
    github_app_id               = var.github_app_id
    github_app_installation_id  = var.github_app_installation_id
    github_app_private_pem_path = snyk_broker_credential.github.id
  }
}

# Step 5: Connect Organization Integration
resource "snyk_broker_connection_integration" "github" {
  tenant_id      = var.snyk_tenant_id
  connection_id  = snyk_broker_connection.github.id
  org_id         = var.snyk_org_id
  type           = "github-server-app"
}

# Step 6: Create a Broker Context (optional)
# Contexts provide environment-specific configuration values for broker connections
# This example uses GitHub Server App with installation ID for authentication
resource "snyk_broker_context" "production" {
  tenant_id     = var.snyk_tenant_id
  install_id    = snyk_broker_app_install.main.install_id
  deployment_id = snyk_broker_deployment.main.id
  connection_id = snyk_broker_connection.github.id

  context = {
    github_installation_id = var.github_app_installation_id
  }
}

# Step 7: Associate Integration with Context (optional)
# This applies the context configuration to a specific organization's integration
resource "snyk_broker_context_integration" "github" {
  tenant_id      = var.snyk_tenant_id
  install_id     = snyk_broker_app_install.main.install_id
  context_id     = snyk_broker_context.production.id
  integration_id = var.github_integration_id
  org_id         = var.snyk_org_id
}

# Data Sources: List existing contexts
data "snyk_broker_connection_contexts" "github" {
  tenant_id     = var.snyk_tenant_id
  install_id    = snyk_broker_app_install.main.install_id
  connection_id = snyk_broker_connection.github.id
}

data "snyk_broker_deployment_contexts" "all" {
  tenant_id     = var.snyk_tenant_id
  install_id    = snyk_broker_app_install.main.install_id
  deployment_id = snyk_broker_deployment.main.id
}

# Outputs
output "deployment_id" {
  description = "The Broker deployment ID"
  value       = snyk_broker_deployment.main.id
}

output "deployment_client_id" {
  description = "OAuth client ID for running the Broker"
  value       = snyk_broker_deployment.main.client_id
  sensitive   = true
}

output "deployment_client_secret" {
  description = "OAuth client secret for running the Broker"
  value       = snyk_broker_deployment.main.client_secret
  sensitive   = true
}

output "connection_id" {
  description = "The Broker connection ID"
  value       = snyk_broker_connection.github.id
}

output "context_id" {
  description = "The Broker context ID"
  value       = snyk_broker_context.production.id
}

output "deployment_contexts" {
  description = "List of all contexts in the deployment"
  value       = data.snyk_broker_deployment_contexts.all.contexts
}

# Docker run command for the Broker
output "broker_docker_command" {
  description = "Docker command to run the Broker"
  value       = <<-EOT
    docker run --restart=always \
      -p 8000:8000 \
      -v /path/to/github-app-private-key.pem:/private-key.pem:ro \
      -e ACCEPT_CODE=true \
      -e DEPLOYMENT_ID=${snyk_broker_deployment.main.id} \
      -e CLIENT_ID=<from deployment_client_id output> \
      -e CLIENT_SECRET=<from deployment_client_secret output> \
      -e GITHUB_APP_PRIVATE_PEM_PATH=/private-key.pem \
      -e UNIVERSAL_BROKER_ENABLED=true \
      -e PORT=8000 \
      -e BROKER_HA_MODE_ENABLED=true \
      snyk/broker:universal
  EOT
}

