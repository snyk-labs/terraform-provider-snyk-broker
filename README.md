# Snyk Broker Terraform Provider

A Terraform provider for managing [Snyk Universal Broker](https://docs.snyk.io/implementation-and-setup/enterprise-setup/snyk-broker/universal-broker) resources.

## Features

- **App Installation**: Install the Snyk Broker App to organizations
- **Deployments**: Create and manage Broker deployments
- **Credentials**: Define credential references for secure secret management
- **Connections**: Configure connections to SCMs, container registries, and other integrations
- **Integrations**: Link organization integrations to Broker connections
- **Bulk Migration**: Migrate multiple organizations to Universal Broker

## Requirements

- [Terraform](https://www.terraform.io/downloads.html) >= 1.0
- [Go](https://golang.org/doc/install) >= 1.21 (for building from source)
- Snyk API Token or OAuth Service Account credentials (when SA has Tenant Admin)

## Installation

### From Terraform Registry (Coming Soon)

```hcl
terraform {
  required_providers {
    snyk = {
      source  = "snyk/snyk-broker"
      version = "~> 0.1"
    }
  }
}
```

### Build from Source

```bash
# Clone the repository
git clone https://github.com/snyk/terraform-provider-snyk-broker.git
cd terraform-provider-snyk-broker

# Build and install locally
make install
```

## Provider Configuration

```hcl
provider "snyk" {
  # Option 1: API Token authentication
  api_token = var.snyk_token

  # Option 2: OAuth authentication (not yet supported until SA has Tenant Admin)
  # client_id     = var.snyk_client_id
  # client_secret = var.snyk_client_secret

  # Region: us (default), eu, or au
  region = "us"
}
```

### Environment Variables

The provider supports the following environment variables:

| Variable | Description |
|----------|-------------|
| `SNYK_TOKEN` | Snyk API token |
| `SNYK_CLIENT_ID` | OAuth client ID |
| `SNYK_CLIENT_SECRET` | OAuth client secret |
| `SNYK_REGION` | Snyk region (us, eu, au) |

## Resources

| Resource | Description |
|----------|-------------|
| `snyk_broker_app_install` | Installs the Broker App to an organization |
| `snyk_broker_deployment` | Creates a Broker deployment |
| `snyk_broker_credential` | Defines credential references |
| `snyk_broker_connection` | Configures Broker connections |
| `snyk_broker_connection_integration` | Links integrations to connections |
| `snyk_broker_bulk_migration` | Migrates organizations to Universal Broker |

## Data Sources

| Data Source | Description |
|-------------|-------------|
| `snyk_broker_deployments` | Lists deployments for a tenant |
| `snyk_broker_connections` | Lists connections for a deployment |
| `snyk_broker_connections_for_org` | Lists connections for an organization |
| `snyk_broker_connection_integrations` | Lists integrations for a connection |
| `snyk_broker_migration_orgs` | Lists organizations available for migration |

## Example Usage

```hcl
# Install the Broker App
resource "snyk_broker_app_install" "main" {
  org_id = var.snyk_org_id
  app_id = var.snyk_broker_app_id
}

# Create a deployment
resource "snyk_broker_deployment" "main" {
  tenant_id  = var.snyk_tenant_id
  install_id = snyk_broker_app_install.main.install_id
  org_id     = var.snyk_org_id
  name       = "Production Broker"

  metadata = {
    cluster = "us-east-1"
  }
}

# Create a credential reference
resource "snyk_broker_credential" "github" {
  tenant_id                   = var.snyk_tenant_id
  install_id                  = snyk_broker_app_install.main.install_id
  deployment_id               = snyk_broker_deployment.main.id
  environment_variable_name   = "GITHUB_TOKEN"
  type                        = "github"
  comment                     = "GitHub PAT for Broker"
}

# Create a connection
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

# Connect organization integration
resource "snyk_broker_connection_integration" "github" {
  tenant_id      = var.snyk_tenant_id
  connection_id  = snyk_broker_connection.github.id
  org_id         = var.snyk_org_id
  integration_id = var.github_integration_id
  type           = "github"
}
```

## Running the Broker

After creating resources with Terraform, run the Broker client:

```bash
docker run --restart=always \
  -p 8000:8000 \
  -e ACCEPT_CODE=true \
  -e DEPLOYMENT_ID=<deployment_id> \
  -e CLIENT_ID=<client_id> \
  -e CLIENT_SECRET=<client_secret> \
  -e GITHUB_TOKEN=<your-github-token> \
  -e UNIVERSAL_BROKER_ENABLED=true \
  -e PORT=8000 \
  -e BROKER_HA_MODE_ENABLED=true \
  snyk/broker:universal
```

## Development

### Building

```bash
make build
```

### Testing

```bash
# Run unit tests
make test

# Run with coverage
make test-coverage
```

### Installing Locally

```bash
make install
```

## License

Apache-2.0

