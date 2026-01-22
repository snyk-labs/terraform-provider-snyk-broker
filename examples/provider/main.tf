# Example: Complete Snyk Broker Setup
# This example shows how to set up a complete Snyk Broker deployment with GitHub integration

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
# This tells the Broker which environment variable will contain the GitHub token
resource "snyk_broker_credential" "github" {
  tenant_id                   = var.snyk_tenant_id
  install_id                  = snyk_broker_app_install.main.install_id
  deployment_id               = snyk_broker_deployment.main.id
  environment_variable_name   = "GITHUB_TOKEN"
  type                        = "github"
  comment                     = "GitHub Personal Access Token for Broker"
}

# Step 4: Create a Broker Connection
resource "snyk_broker_connection" "github" {
  tenant_id     = var.snyk_tenant_id
  install_id    = snyk_broker_app_install.main.install_id
  deployment_id = snyk_broker_deployment.main.id
  name          = "GitHub Enterprise"
  type          = "github"

  configuration = {
    github_token      = snyk_broker_credential.github.id
    broker_client_url = "https://broker.example.com:8000"
  }
}

# Step 5: Connect Organization Integration
resource "snyk_broker_connection_integration" "github" {
  tenant_id      = var.snyk_tenant_id
  connection_id  = snyk_broker_connection.github.id
  org_id         = var.snyk_org_id
  type           = "github"
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

# Docker run command for the Broker
output "broker_docker_command" {
  description = "Docker command to run the Broker"
  value       = <<-EOT
    docker run --restart=always \
      -p 8000:8000 \
      -e ACCEPT_CODE=true \
      -e DEPLOYMENT_ID=${snyk_broker_deployment.main.id} \
      -e CLIENT_ID=<from deployment_client_id output> \
      -e CLIENT_SECRET=<from deployment_client_secret output> \
      -e GITHUB_TOKEN=<your-github-token> \
      -e UNIVERSAL_BROKER_ENABLED=true \
      -e PORT=8000 \
      -e BROKER_HA_MODE_ENABLED=true \
      snyk/broker:universal
  EOT
}

