# ZRP API Developer Guide

Welcome to the ZRP API! This guide provides practical examples and workflows for integrating with ZRP's REST API.

## Table of Contents

- [Getting Started](#getting-started)
- [Authentication](#authentication)
- [Common Workflows](#common-workflows)
- [Error Handling](#error-handling)
- [Rate Limiting](#rate-limiting)
- [Best Practices](#best-practices)

## Getting Started

### Base URL

```
http://localhost:9000/api/v1
```

Replace with your ZRP instance URL in production.

### Response Format

All successful responses are wrapped in a data envelope:

```json
{
  "data": { ... }
}
```

Paginated responses include metadata:

```json
{
  "data": [...],
  "meta": {
    "total": 150,
    "page": 1,
    "limit": 50
  }
}
```

Error responses:

```json
{
  "error": "Error message",
  "code": "ERROR_CODE"
}
```

## Authentication

ZRP supports two authentication methods:

### 1. Session Cookie Authentication (For Web UI)

Best for web applications and interactive sessions.

**Login:**

```bash
curl -X POST http://localhost:9000/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "username": "admin",
    "password": "your-password"
  }' \
  -c cookies.txt
```

**Use session in subsequent requests:**

```bash
curl http://localhost:9000/api/v1/parts \
  -b cookies.txt
```

**Check current user:**

```bash
curl http://localhost:9000/auth/me \
  -b cookies.txt
```

**Logout:**

```bash
curl -X POST http://localhost:9000/auth/logout \
  -b cookies.txt
```

### 2. API Key Authentication (For Programmatic Access)

Best for automation, integrations, and long-running scripts.

**Generate API Key (via UI or authenticated request):**

```bash
curl -X POST http://localhost:9000/api/v1/apikeys \
  -b cookies.txt \
  -H "Content-Type: application/json" \
  -d '{
    "name": "Integration Script",
    "expires_at": "2025-12-31T23:59:59Z"
  }'
```

Response:

```json
{
  "id": 1,
  "name": "Integration Script",
  "key": "zrp_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6",
  "key_prefix": "zrp_a1b2c3d4",
  "created_at": "2024-02-19T10:00:00Z",
  "enabled": 1,
  "message": "Store this key securely. It will not be shown again."
}
```

⚠️ **Important:** Save the `key` value immediately - it's only shown once!

**Use API Key in requests:**

```bash
curl http://localhost:9000/api/v1/parts \
  -H "Authorization: Bearer zrp_a1b2c3d4e5f6g7h8i9j0k1l2m3n4o5p6"
```

**List API Keys:**

```bash
curl http://localhost:9000/api/v1/apikeys \
  -H "Authorization: Bearer YOUR_API_KEY"
```

**Revoke API Key:**

```bash
curl -X DELETE http://localhost:9000/api/v1/apikeys/1 \
  -H "Authorization: Bearer YOUR_API_KEY"
```

## Common Workflows

### Workflow 1: Create a Work Order

This workflow demonstrates creating a work order for manufacturing a product.

**Step 1: Search for the part**

```bash
curl "http://localhost:9000/api/v1/parts?q=WIDGET-001" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

Response:

```json
{
  "data": [
    {
      "ipn": "WIDGET-001",
      "category": "Assemblies",
      "lifecycle_stage": "production",
      "fields": {
        "description": "Main Widget Assembly",
        "revision": "C"
      }
    }
  ]
}
```

**Step 2: Check BOM and inventory availability**

```bash
curl "http://localhost:9000/api/v1/parts/WIDGET-001/bom" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

```bash
curl "http://localhost:9000/api/v1/inventory?ipn=WIDGET-001" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

**Step 3: Create the work order**

```bash
curl -X POST http://localhost:9000/api/v1/workorders \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "ipn": "WIDGET-001",
    "quantity": 100,
    "priority": "high",
    "due_date": "2024-03-15",
    "assigned_to": "Manufacturing Team",
    "notes": "Rush order for customer ABC"
  }'
```

Response:

```json
{
  "data": {
    "id": 42,
    "wo_number": "WO-2024-042",
    "ipn": "WIDGET-001",
    "quantity": 100,
    "status": "planned",
    "priority": "high",
    "due_date": "2024-03-15",
    "assigned_to": "Manufacturing Team"
  }
}
```

**Step 4: Update work order status**

```bash
curl -X PUT http://localhost:9000/api/v1/workorders/42 \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "in-progress",
    "notes": "Started production on line 3"
  }'
```

### Workflow 2: Add Inventory from Purchase Order Receipt

**Step 1: List pending purchase orders**

```bash
curl "http://localhost:9000/api/v1/pos?status=sent" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

**Step 2: Receive items from PO**

```bash
curl -X POST http://localhost:9000/api/v1/pos/15/receive \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "location": "Warehouse-A",
    "lines": [
      {
        "ipn": "RES-0603-10K",
        "quantity_received": 5000,
        "lot_number": "LOT-2024-123"
      },
      {
        "ipn": "CAP-0805-10UF",
        "quantity_received": 3000,
        "lot_number": "LOT-2024-124"
      }
    ]
  }'
```

This automatically:
- Creates inventory records
- Updates PO line item received quantities
- Generates audit trail entries

**Step 3: Verify inventory was added**

```bash
curl "http://localhost:9000/api/v1/inventory?ipn=RES-0603-10K&location=Warehouse-A" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

### Workflow 3: Generate and Implement an ECO

**Step 1: Create pending part changes**

```bash
curl -X POST http://localhost:9000/api/v1/parts/WIDGET-001/changes \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "change_type": "bom_update",
    "description": "Replace R5 with higher wattage resistor",
    "changes": {
      "bom": [
        {
          "child_ipn": "RES-1206-10K-0.5W",
          "quantity": 1,
          "reference": "R5",
          "notes": "Upgrade from 0.25W to 0.5W"
        }
      ]
    }
  }'
```

**Step 2: Create ECO from changes**

```bash
curl -X POST http://localhost:9000/api/v1/parts/WIDGET-001/changes/create-eco \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Upgrade R5 to 0.5W resistor",
    "description": "Field reports show R5 running hot. Upgrading to 0.5W part.",
    "priority": "high",
    "category": "design"
  }'
```

Response:

```json
{
  "data": {
    "id": 18,
    "eco_number": "ECO-2024-018",
    "title": "Upgrade R5 to 0.5W resistor",
    "status": "draft",
    "affected_parts": ["WIDGET-001"]
  }
}
```

**Step 3: Approve ECO (requires manager/admin role)**

```bash
curl -X POST http://localhost:9000/api/v1/ecos/18/approve \
  -H "Authorization: Bearer YOUR_ADMIN_API_KEY"
```

**Step 4: Implement ECO**

```bash
curl -X POST http://localhost:9000/api/v1/ecos/18/implement \
  -H "Authorization: Bearer YOUR_API_KEY"
```

This automatically:
- Applies all pending changes to affected parts
- Increments part revisions
- Updates BOM
- Creates audit trail
- Sends notifications

### Workflow 4: Search Parts and Check Market Pricing

**Step 1: Search for resistor parts**

```bash
curl "http://localhost:9000/api/v1/parts?category=Resistors&q=10k" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

**Step 2: Get market pricing for a part (if Digi-Key/Mouser configured)**

```bash
curl "http://localhost:9000/api/v1/parts/RES-0603-10K/market-pricing" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

Response:

```json
{
  "data": {
    "ipn": "RES-0603-10K",
    "mpn": "RC0603FR-0710KL",
    "distributors": {
      "digikey": {
        "price_breaks": [
          {"quantity": 1, "unit_price": 0.10},
          {"quantity": 100, "unit_price": 0.05},
          {"quantity": 1000, "unit_price": 0.02}
        ],
        "stock": 150000,
        "lead_time_days": 0
      },
      "mouser": {
        "price_breaks": [
          {"quantity": 1, "unit_price": 0.11},
          {"quantity": 100, "unit_price": 0.054}
        ],
        "stock": 95000,
        "lead_time_days": 1
      }
    },
    "recommended_vendor": "digikey",
    "recommended_price": 0.02
  }
}
```

**Step 3: Update part cost based on market pricing**

```bash
curl -X POST http://localhost:9000/api/v1/prices \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "ipn": "RES-0603-10K",
    "vendor_id": 1,
    "unit_price": 0.02,
    "quantity_break": 1000,
    "currency": "USD",
    "notes": "Updated from Digi-Key pricing"
  }'
```

## Error Handling

### Common HTTP Status Codes

| Code | Meaning | Description |
|------|---------|-------------|
| 200 | OK | Request successful |
| 201 | Created | Resource created successfully |
| 400 | Bad Request | Invalid request body or parameters |
| 401 | Unauthorized | Missing or invalid authentication |
| 403 | Forbidden | Insufficient permissions |
| 404 | Not Found | Resource does not exist |
| 429 | Too Many Requests | Rate limit exceeded |
| 500 | Internal Server Error | Server error |

### Error Response Format

```json
{
  "error": "Part with IPN 'ABC-123' already exists",
  "code": "DUPLICATE_IPN"
}
```

### Common Error Codes

- `UNAUTHORIZED` - Authentication required or invalid
- `FORBIDDEN` - Insufficient permissions
- `NOT_FOUND` - Resource not found
- `DUPLICATE_IPN` - Part IPN already exists
- `INVALID_BOM` - BOM contains errors (circular references, etc.)
- `INSUFFICIENT_STOCK` - Not enough inventory for transaction
- `VALIDATION_ERROR` - Request validation failed

### Example Error Handling (Python)

```python
import requests

API_KEY = "zrp_your_api_key_here"
BASE_URL = "http://localhost:9000/api/v1"

headers = {
    "Authorization": f"Bearer {API_KEY}",
    "Content-Type": "application/json"
}

try:
    response = requests.post(
        f"{BASE_URL}/parts",
        headers=headers,
        json={
            "ipn": "NEW-PART-001",
            "category": "Electronics",
            "fields": {"description": "New component"}
        }
    )
    
    response.raise_for_status()  # Raises HTTPError for 4xx/5xx
    
    data = response.json()
    print(f"Part created: {data['data']['ipn']}")
    
except requests.exceptions.HTTPError as e:
    if e.response.status_code == 400:
        error = e.response.json()
        if error.get('code') == 'DUPLICATE_IPN':
            print("Part already exists!")
        else:
            print(f"Validation error: {error.get('error')}")
    elif e.response.status_code == 401:
        print("Authentication failed - check API key")
    elif e.response.status_code == 403:
        print("Permission denied - insufficient privileges")
    else:
        print(f"HTTP Error {e.response.status_code}: {e}")
        
except requests.exceptions.RequestException as e:
    print(f"Request failed: {e}")
```

## Rate Limiting

### Login Rate Limiting

Login attempts are rate-limited to prevent brute force attacks:

- **Limit:** 5 attempts per minute per IP address
- **Response:** HTTP 429 with error message
- **Retry:** Wait 60 seconds before retrying

```json
{
  "error": "Too many login attempts. Try again in a minute.",
  "code": "RATE_LIMITED"
}
```

### API Key Rate Limiting

API keys currently have no rate limits, but best practices include:

- Implement exponential backoff on errors
- Cache frequently accessed data
- Use bulk endpoints when available
- Avoid polling - use webhooks (if available)

## Best Practices

### 1. Use API Keys for Automation

- **Do:** Use API keys for scripts, integrations, and CI/CD
- **Don't:** Use session cookies in long-running background processes

### 2. Handle Pagination Properly

For large datasets, always handle pagination:

```python
def fetch_all_parts(category=None):
    parts = []
    page = 1
    limit = 100
    
    while True:
        response = requests.get(
            f"{BASE_URL}/parts",
            headers=headers,
            params={"category": category, "page": page, "limit": limit}
        )
        
        data = response.json()
        parts.extend(data['data'])
        
        # Check if we've fetched all pages
        if len(data['data']) < limit:
            break
            
        page += 1
    
    return parts
```

### 3. Use Bulk Endpoints When Available

Bulk endpoints are more efficient than individual requests:

```bash
# Good: Bulk inventory update
curl -X POST http://localhost:9000/api/v1/inventory/bulk-update \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -H "Content-Type: application/json" \
  -d '{
    "updates": [
      {"id": 1, "quantity": 150},
      {"id": 2, "quantity": 200},
      {"id": 3, "quantity": 75}
    ]
  }'

# Avoid: Multiple individual requests
# curl -X PUT .../inventory/1 ...
# curl -X PUT .../inventory/2 ...
# curl -X PUT .../inventory/3 ...
```

### 4. Filter Server-Side, Not Client-Side

```bash
# Good: Filter on the server
curl "http://localhost:9000/api/v1/parts?category=Resistors&lifecycle_stage=production" \
  -H "Authorization: Bearer YOUR_API_KEY"

# Bad: Fetch everything and filter client-side
# curl "http://localhost:9000/api/v1/parts" | jq '.data[] | select(.category=="Resistors")'
```

### 5. Validate Data Before Sending

```python
def create_work_order(ipn, quantity, due_date):
    # Validate before sending
    if quantity <= 0:
        raise ValueError("Quantity must be positive")
    
    if not ipn:
        raise ValueError("IPN is required")
    
    # Additional validation: check if part exists
    part_response = requests.get(
        f"{BASE_URL}/parts/{ipn}",
        headers=headers
    )
    
    if part_response.status_code == 404:
        raise ValueError(f"Part {ipn} does not exist")
    
    # Create work order
    response = requests.post(
        f"{BASE_URL}/workorders",
        headers=headers,
        json={
            "ipn": ipn,
            "quantity": quantity,
            "due_date": due_date
        }
    )
    
    return response.json()
```

### 6. Use Transactions for Related Operations

When performing related operations (e.g., creating a work order and reserving inventory), handle failures gracefully:

```python
def create_wo_with_reservation(ipn, quantity):
    # Create work order
    wo_response = requests.post(
        f"{BASE_URL}/workorders",
        headers=headers,
        json={"ipn": ipn, "quantity": quantity}
    )
    
    if wo_response.status_code != 201:
        raise Exception("Failed to create work order")
    
    wo_id = wo_response.json()['data']['id']
    
    # Reserve inventory
    try:
        inv_response = requests.post(
            f"{BASE_URL}/inventory/transact",
            headers=headers,
            json={
                "ipn": ipn,
                "quantity": -quantity,
                "transaction_type": "out",
                "notes": f"Reserved for WO-{wo_id}"
            }
        )
        
        if inv_response.status_code != 200:
            # Rollback: cancel work order
            requests.put(
                f"{BASE_URL}/workorders/{wo_id}",
                headers=headers,
                json={"status": "cancelled", "notes": "Insufficient inventory"}
            )
            raise Exception("Insufficient inventory")
            
    except Exception as e:
        # Cleanup on failure
        requests.put(
            f"{BASE_URL}/workorders/{wo_id}",
            headers=headers,
            json={"status": "cancelled", "notes": str(e)}
        )
        raise
    
    return wo_id
```

### 7. Leverage Search for Discovery

Use the global search endpoint for fuzzy matching:

```bash
# Search across all entities
curl "http://localhost:9000/api/v1/search?q=capacitor+10uf" \
  -H "Authorization: Bearer YOUR_API_KEY"
```

Response includes parts, docs, ECOs, and work orders matching the query.

### 8. Monitor API Key Usage

Periodically review API key usage:

```bash
curl http://localhost:9000/api/v1/apikeys \
  -H "Authorization: Bearer YOUR_ADMIN_API_KEY" \
  | jq '.data[] | {name, last_used, enabled}'
```

Revoke unused or compromised keys immediately.

## Additional Resources

- **OpenAPI Spec:** `/docs/api-spec.yaml` - Full API specification
- **Quick Reference:** `/docs/API_QUICK_REFERENCE.md` - Endpoint cheat sheet
- **Legacy API Docs:** `/docs/API.md` - Original documentation
- **WebSocket API:** Contact support for real-time notifications

## Support

For API issues or feature requests:

- Check the GitHub issues
- Review audit logs: `GET /api/v1/audit`
- Enable debug mode for query profiling: `GET /api/v1/debug/slow-queries`

---

**Last Updated:** 2024-02-19  
**API Version:** 1.0.0
