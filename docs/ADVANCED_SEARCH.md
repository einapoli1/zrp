# Advanced Search and Filtering

ZRP now includes comprehensive advanced search and filtering capabilities across all major entity types.

## Features

### 1. **Multi-Field Search**
Search across multiple fields simultaneously. For example, searching for "capacitor" in Parts will search across:
- IPN (Part Number)
- Description
- Vendor
- Manufacturer Part Number (MPN)

### 2. **Advanced Operators**
Support for powerful search operators:

| Operator | Syntax | Description | Example |
|----------|--------|-------------|---------|
| **Equals** | `field:value` or `field=value` | Exact match | `status:open` |
| **Not Equals** | `field!=value` | Exclude matches | `status!=closed` |
| **Contains** | `field:value` or `field:*value*` | Substring match | `title:*capacitor*` |
| **Starts With** | `field:value*` | Prefix match | `ipn:CAP*` |
| **Ends With** | `field:*value` | Suffix match | `ipn:*001` |
| **Greater Than** | `field>value` | Numeric/date comparison | `qty>100` |
| **Less Than** | `field<value` | Numeric/date comparison | `qty<50` |
| **Greater or Equal** | `field>=value` | Inclusive comparison | `qty>=100` |
| **Less or Equal** | `field<=value` | Inclusive comparison | `qty<=50` |
| **In List** | `field:val1,val2,val3` | Match any value | `status:open,pending` |
| **Between** | Special filter builder | Range query | Date or numeric ranges |
| **Is Null** | Special filter | Null check | Empty fields |
| **Is Not Null** | Special filter | Non-null check | Required fields |

### 3. **Filter Combinations**
Combine multiple filters with AND/OR logic:
- `status:open AND priority:high` - Find high priority open items
- `status:open OR status:pending` - Find items in either status

### 4. **Quick Filters**
Pre-configured filters for common searches:

#### Parts
- **Active Parts** - Parts with status=active
- **Obsolete Parts** - Parts marked as obsolete

#### Work Orders
- **Open Work Orders** - WOs with status open or in_progress
- **High Priority** - High priority uncompleted WOs
- **Overdue** - WOs past due date that aren't completed

#### Inventory
- **Low Stock** - Items at or below reorder point
- **Out of Stock** - Items with zero quantity

#### ECOs
- **Pending ECOs** - ECOs in draft or pending status
- **Approved ECOs** - ECOs approved for implementation
- **High Priority** - High priority non-rejected ECOs

#### NCRs
- **Open NCRs** - NCRs with open status
- **Critical NCRs** - Critical or major severity open NCRs

### 5. **Saved Searches**
Save your frequently used search configurations:
- Save filter combinations with custom names
- Share saved searches across the team (public searches)
- Quickly reload saved searches from dropdown

### 6. **Search History**
The system tracks your recent searches (last 5-10) for quick access to previous queries.

### 7. **Sorting**
Sort results by any field in ascending or descending order.

## Usage Examples

### Basic Text Search
```
capacitor
```
Searches across all relevant fields for "capacitor"

### Field-Specific Search
```
status:open
```
Find items with status exactly "open"

### Wildcard Search
```
ipn:CAP*
```
Find all IPNs starting with "CAP"

### Advanced Operators in Search Bar
```
status:open priority>3
```
Find open items with priority greater than 3

### Multiple Filters
Use the Advanced Search panel to build complex queries:
1. Click "Advanced" button
2. Add filters with + button
3. Select field, operator, and value
4. Choose AND/OR between filters
5. Click "Apply Filters"

### Save Common Searches
1. Build your search with filters
2. Click "Save Search"
3. Give it a name (e.g., "Critical Open Issues")
4. Optionally make it public to share with team
5. Click Save

### Using Quick Filters
Click any quick filter button to instantly apply pre-configured searches.

## Supported Entity Types

Advanced search is available for:
- **Parts** - Search by IPN, description, category, vendor, MPN
- **Work Orders** - Search by WO#, status, date range, assigned to
- **Inventory** - Search by part, location, stock levels
- **ECOs** - Search by number, status, priority, affected parts
- **NCRs** - Search by ID, status, severity, affected components
- **Devices** - Search by serial number, customer, location
- **Purchase Orders** - Search by PO#, vendor, status

## Performance

Advanced search includes optimized database indexes on all searchable fields:
- Target response time: <500ms for searches on 10,000+ records
- Indexes on status, priority, dates, and foreign keys
- Efficient query plan generation with proper WHERE clauses

## Filter Chips

Active filters are displayed as removable chips below the search bar:
- Click X on any chip to remove that filter
- Click "Clear All" to reset all filters

## API Reference

### POST /api/v1/search/advanced
Execute advanced search

**Request Body:**
```json
{
  "entity_type": "parts",
  "filters": [
    {
      "field": "status",
      "operator": "eq",
      "value": "active",
      "andOr": "AND"
    },
    {
      "field": "qty_on_hand",
      "operator": "gt",
      "value": 0
    }
  ],
  "search_text": "capacitor",
  "sort_by": "ipn",
  "sort_order": "asc",
  "limit": 50,
  "offset": 0
}
```

**Response:**
```json
{
  "data": [...],
  "total": 123,
  "page": 1,
  "page_size": 50,
  "total_pages": 3
}
```

### GET /api/v1/search/quick-filters?entity_type=parts
Get quick filter presets for entity type

### GET /api/v1/search/history?entity_type=parts&limit=10
Get recent search history

### GET /api/v1/saved-searches?entity_type=parts
Get saved searches (user's + public)

### POST /api/v1/saved-searches
Create a saved search

### DELETE /api/v1/saved-searches?id={id}
Delete a saved search (only if you own it)

## Best Practices

1. **Start Simple** - Use basic text search first, then add filters as needed
2. **Use Quick Filters** - Leverage pre-built filters for common queries
3. **Save Frequent Searches** - Save searches you use regularly
4. **Share Knowledge** - Make useful searches public for your team
5. **Combine Operators** - Use multiple filters to narrow results precisely
6. **Check History** - Reuse recent successful searches from history

## Tips and Tricks

- **Wildcards**: Use `*` for wildcard matching (e.g., `CAP*` or `*001`)
- **Dates**: Use ISO format for date comparisons (YYYY-MM-DD)
- **Case Insensitive**: Text searches are case-insensitive
- **Numeric Ranges**: Use between operator or combine >= and <=
- **Status Lists**: Use IN operator for multiple status values
- **Boolean Logic**: Combine AND/OR to create complex queries
- **Empty Fields**: Use "Is Null" to find missing data

## Future Enhancements

Planned improvements:
- Full-text search (FTS5) for faster text searches
- Search suggestions and autocomplete
- Export search results
- Scheduled saved searches (email reports)
- Search analytics (most popular searches)
- Advanced date expressions (last week, this month, etc.)
