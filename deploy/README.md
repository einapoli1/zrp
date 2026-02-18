# ZRP Docker Deployment

This directory contains deployment files for ZRP in containerized environments.

## Files

- `portainer-stack.yml` - Portainer stack file for deployment via Portainer UI or API

## Local Development with Docker

```bash
# Build and run with docker-compose
docker-compose up --build

# Or build the image manually
docker build -t zrp:0.2.0 .
docker run -p 9000:9000 -v zrp_data:/app/data -v zrp_uploads:/app/uploads zrp:0.2.0
```

## Portainer Deployment

### Via Portainer Web UI

1. Navigate to http://containers.zonit.com:9443
2. Login with your credentials
3. Go to "Stacks" → "Add Stack"
4. Name: `zrp`
5. Copy the contents of `deploy/portainer-stack.yml`
6. Click "Deploy the stack"

### Via Portainer API

You need an API access token from Portainer. Get it from:
Settings → Users → [Your User] → Access tokens

```bash
# Set your API token
PORTAINER_TOKEN="your-api-access-token-here"
PORTAINER_URL="http://containers.zonit.com:9443"

# Deploy the stack
curl -X POST "${PORTAINER_URL}/api/stacks?type=2&method=string&endpointId=1" \
  -H "X-API-Key: ${PORTAINER_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "zrp",
    "stackFileContent": "'"$(cat deploy/portainer-stack.yml | sed 's/"/\\"/g' | tr '\n' ' ')"'",
    "env": []
  }'
```

To update an existing stack:
```bash
# Get stack ID first
STACK_ID=$(curl -s -H "X-API-Key: ${PORTAINER_TOKEN}" \
  "${PORTAINER_URL}/api/stacks" | jq -r '.[] | select(.Name=="zrp") | .Id')

# Update the stack
curl -X PUT "${PORTAINER_URL}/api/stacks/${STACK_ID}?endpointId=1" \
  -H "X-API-Key: ${PORTAINER_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "stackFileContent": "'"$(cat deploy/portainer-stack.yml | sed 's/"/\\"/g' | tr '\n' ' ')"'",
    "env": [],
    "prune": true
  }'
```

## Environment Variables

Customize these in the stack file or via Portainer environment variables:

- `ZRP_COMPANY_NAME` - Company name for PDF documents (default: "ZRP Manufacturing")
- `ZRP_COMPANY_EMAIL` - Contact email for PDF documents (default: "admin@zrp.local")

## Volumes

- `zrp-data` - SQLite database storage
- `zrp-uploads` - File attachments and uploads

## Health Check

The container includes a health check that tests the web server endpoint. It will show as healthy once the application is responding to HTTP requests.

## Ports

- Port 9000 - ZRP web interface and API