# Manufacturing Workflow

This document describes the comprehensive manufacturing workflow implemented in ZRP, covering work order management, material kitting, serial number tracking, and inventory integration.

## Overview

The manufacturing workflow follows this sequence:
1. **Create Work Order** → Define what to build and how many units
2. **Kit Materials** → Reserve required materials from inventory 
3. **Generate/Assign Serials** → Track individual units being built
4. **Execute Production** → Build, test, and track progress
5. **Complete Work Order** → Update inventory and release reservations

## Work Order States

Work orders follow a strict state machine with these valid transitions:

```
draft → open → in_progress → completed
  ↓       ↓         ↓
cancelled  cancelled  on_hold → in_progress
                       ↓
                     cancelled
```

### Status Definitions

- **draft**: Work order created but not ready for production
- **open**: Ready for production, materials can be kitted
- **in_progress**: Production started, materials kitted
- **on_hold**: Temporarily paused, can resume to in_progress
- **completed**: Finished, inventory updated, reservations released
- **cancelled**: Terminated, reservations released

## Material Kitting

### Purpose
Material kitting reserves inventory items needed for a work order, ensuring materials are available when production starts.

### API Endpoint
```
POST /api/v1/workorders/{id}/kit
```

### Process
1. **Check BOM Requirements**: Determine material needs based on assembly BOM
2. **Verify Availability**: Check `qty_on_hand - qty_reserved` for each item
3. **Create Reservations**: Update `qty_reserved` in inventory table
4. **Update Work Order**: Change status from `open` to `in_progress` if successful
5. **Return Results**: Show kitting status for each material

### Kitting Statuses
- **kitted**: Full quantity reserved successfully
- **partial**: Some quantity reserved, but not enough available
- **shortage**: No quantity available for reservation
- **error**: Database error during reservation

### Example Response
```json
{
  "wo_id": "WO001",
  "status": "kitted",
  "kitted_at": "2024-01-15T10:30:00Z",
  "items": [
    {
      "ipn": "PART-001",
      "required": 10.0,
      "on_hand": 15.0,
      "reserved": 5.0,
      "kitted": 10.0,
      "status": "kitted"
    },
    {
      "ipn": "PART-002", 
      "required": 5.0,
      "on_hand": 3.0,
      "reserved": 0.0,
      "kitted": 3.0,
      "status": "partial"
    }
  ]
}
```

## Serial Number Management

### Purpose
Track individual units being built within a work order, enabling traceability and test result correlation.

### Database Schema
```sql
CREATE TABLE wo_serials (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    wo_id TEXT NOT NULL,
    serial_number TEXT UNIQUE NOT NULL,
    status TEXT NOT NULL DEFAULT 'assigned',
    notes TEXT
);
```

### API Endpoints

#### Get Serials
```
GET /api/v1/workorders/{id}/serials
```

#### Add Serial
```
POST /api/v1/workorders/{id}/serials
Content-Type: application/json

{
  "serial_number": "ASY123456789012",  // Optional - auto-generated if empty
  "status": "assigned",                 // Optional - defaults to "assigned"
  "notes": "Special handling required"  // Optional
}
```

### Serial Number Generation
When `serial_number` is not provided, the system auto-generates using:
- **Format**: `{ASSEMBLY_PREFIX}{YYMMDDHHMMSS}`
- **Example**: `ASY240115103045` (Assembly ASY-001 created Jan 15, 2024 at 10:30:45)

### Serial Statuses
- **assigned**: Serial number created and assigned to work order
- **in_progress**: Unit is being built
- **testing**: Unit is undergoing testing
- **completed**: Unit passed all tests and is ready
- **failed**: Unit failed testing and needs rework
- **scrapped**: Unit cannot be repaired and is discarded

## Inventory Integration

### On Work Order Completion
When a work order status changes to `completed`, the system automatically:

1. **Add Finished Goods**
   - Create inventory record for assembly IPN if it doesn't exist
   - Add `qty` units to `qty_on_hand` for the assembly
   - Log transaction with type `receive`, reference `{WO_ID}`

2. **Consume Materials**
   - For each material with `qty_reserved > 0`:
   - Calculate consumption: `qty_reserved × wo_qty`
   - Subtract from `qty_on_hand`
   - Reset `qty_reserved` to 0
   - Log transaction with type `issue`, reference `{WO_ID}`

3. **Audit Trail**
   - All inventory changes are logged in `inventory_transactions`
   - Audit log captures work order completion event
   - Change tracking records before/after snapshots

### Transaction Examples
```sql
-- Finished goods transaction
INSERT INTO inventory_transactions (
    ipn, type, qty, reference, notes, created_at
) VALUES (
    'ASY-001', 'receive', 10, 'WO001', 'WO WO001 completion', '2024-01-15 10:30:00'
);

-- Material consumption transaction  
INSERT INTO inventory_transactions (
    ipn, type, qty, reference, notes, created_at
) VALUES (
    'PART-001', 'issue', 20, 'WO001', 'WO WO001 material consumption', '2024-01-15 10:30:00'
);
```

## Yield and Scrap Tracking

Work orders now support yield tracking with additional fields:

### Database Fields
- `qty`: Total units planned
- `qty_good`: Units that passed all tests
- `qty_scrap`: Units that were scrapped
- **Yield Calculation**: `(qty_good / qty) × 100`

### Usage
- Update `qty_good` and `qty_scrap` as production progresses
- System calculates yield percentage automatically
- Yield data appears on work order detail page
- Useful for process improvement and quality metrics

## Status Enforcement

The system enforces valid state transitions to prevent invalid work order states:

### Validation Rules
1. **Draft Work Orders**: Can only transition to `open` or `cancelled`
2. **Terminal States**: `completed` and `cancelled` cannot transition to any other state
3. **Hold State**: `on_hold` can resume to `in_progress` or be cancelled
4. **Sequential Flow**: Must progress through logical sequence

### API Response
Invalid transitions return HTTP 400 with error message:
```json
{
  "error": "invalid transition from completed to open"
}
```

## Testing Integration

Work order detail pages now link to the testing module:
- **Link**: `/testing?wo_id={WO_ID}`
- **Purpose**: Record test results for work order serials
- **Integration**: Test results can reference work order serials for traceability

## Error Handling

### Material Kitting Errors
- **Insufficient Inventory**: Returns partial kitting results
- **Database Errors**: Rolls back all reservations, returns error status
- **Invalid Work Order**: Returns 404 if work order doesn't exist

### Serial Number Errors
- **Duplicate Serial**: Returns validation error for existing serial numbers
- **Invalid Work Order**: Returns 404 for non-existent work orders
- **Completed Work Orders**: Returns 400 when trying to add serials to completed WOs

### Inventory Integration Errors
- **Transaction Failures**: All inventory updates are wrapped in database transactions
- **Rollback**: If any step fails, entire completion process is rolled back
- **Logging**: All errors are logged with full context for debugging

## Best Practices

### Work Order Management
1. **Plan First**: Create work orders in `draft` status to plan materials
2. **Kit Early**: Kit materials when ready to start production
3. **Track Serials**: Generate serials before starting build process
4. **Update Progress**: Regularly update `qty_good` and `qty_scrap` counts

### Material Management
1. **Check Availability**: Review BOM status before kitting
2. **Generate POs**: Use "Generate PO" for shortage items
3. **Kit Only When Ready**: Don't kit materials too far in advance
4. **Reserve Properly**: Kitted materials are reserved and unavailable for other WOs

### Quality Control
1. **Serial Tracking**: Assign unique serials to all units
2. **Test Integration**: Link test results to specific serials
3. **Yield Monitoring**: Track good vs. scrap quantities
4. **Status Updates**: Keep serial statuses current throughout build

### Inventory Accuracy
1. **Complete Work Orders**: Always complete WOs to update inventory
2. **Monitor Transactions**: Review inventory transaction log regularly  
3. **Audit Regularly**: Compare physical inventory to system records
4. **Handle Exceptions**: Address any transaction errors promptly

## Troubleshooting

### Common Issues

**Materials Not Kitting**
- Check inventory `qty_on_hand` vs. requirements
- Verify no other reservations are conflicting
- Ensure work order is in `open` status

**Serials Not Generating**
- Check for duplicate serial numbers in database
- Verify work order is not `completed` or `cancelled`
- Review auto-generation logic for assembly IPN format

**Inventory Not Updating on Completion**
- Verify work order status changed to `completed`
- Check transaction log for completion entries
- Look for database constraint violations
- Review error logs for transaction rollbacks

**Invalid Status Transitions**
- Review state machine diagram above
- Check current work order status
- Ensure transition is valid per rules
- Consider if work order should be reset to earlier state

For additional support, check the audit log and transaction history to understand the sequence of events leading to any issues.