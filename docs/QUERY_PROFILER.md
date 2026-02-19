# Query Profiler - Performance Monitoring Guide

## Overview

The ZRP Query Profiler is a built-in tool for monitoring database query performance in development and production environments. It tracks query execution times, identifies slow queries, and provides performance statistics.

---

## Enabling the Profiler

### Environment Variables

Set these environment variables to control the profiler:

```bash
export QUERY_PROFILER_ENABLED=true
export QUERY_PROFILER_THRESHOLD_MS=100  # Threshold for "slow query" (default: 100ms)
```

### Initialization

The profiler is initialized in `main.go`:

```go
InitQueryProfiler(true, 100)  // enabled, 100ms threshold
```

---

## API Endpoints

### 1. Get Statistics

**Endpoint:** `GET /api/debug/query-stats`  
**Description:** Returns overall profiling statistics  
**Example Response:**

```json
{
  "enabled": true,
  "total_queries": 1523,
  "slow_queries": 12,
  "avg_duration": "15.3ms",
  "min_duration": "1.2ms",
  "max_duration": "450ms",
  "threshold": "100ms"
}
```

---

### 2. Get Slow Queries

**Endpoint:** `GET /api/debug/slow-queries`  
**Description:** Returns all queries that exceeded the threshold  
**Example Response:**

```json
{
  "slow_queries": [
    {
      "query": "SELECT * FROM inventory WHERE qty_on_hand > 0",
      "duration": "250ms",
      "timestamp": "2026-02-19T14:30:45Z",
      "args": "[0]"
    },
    {
      "query": "SELECT id FROM work_orders WHERE status = ?",
      "duration": "180ms",
      "timestamp": "2026-02-19T14:31:10Z",
      "args": "[in_progress]"
    }
  ],
  "count": 2,
  "threshold": "100ms"
}
```

---

### 3. Get All Queries

**Endpoint:** `GET /api/debug/all-queries`  
**Description:** Returns all recorded queries (up to 1000 most recent)  
**Example Response:**

```json
{
  "queries": [
    {
      "query": "SELECT id FROM users WHERE username = ?",
      "duration": "3.5ms",
      "timestamp": "2026-02-19T14:30:00Z",
      "args": "[admin]"
    }
    // ... more queries
  ],
  "count": 523
}
```

---

### 4. Reset Profiler

**Endpoint:** `POST /api/debug/query-reset`  
**Description:** Clears all recorded queries  
**Example Response:**

```json
{
  "message": "Profiler reset successfully"
}
```

---

## Log Files

### slow_queries.log

When enabled, slow queries are automatically logged to `slow_queries.log`:

```
[2026-02-19 14:30:45] SLOW QUERY (250ms): SELECT * FROM inventory WHERE qty_on_hand > 0 | Args: [0]
[2026-02-19 14:31:10] SLOW QUERY (180ms): SELECT id FROM work_orders WHERE status = ? | Args: [in_progress]
```

---

## Using the Profiler in Code

### Option 1: Direct Database Access (Untracked)

Standard database queries are NOT automatically profiled:

```go
rows, err := db.Query("SELECT * FROM users")  // Not profiled
```

### Option 2: Using Profiler Wrappers (Tracked)

To profile specific queries, use the profiler wrappers:

```go
import "time"

// Profile a Query
rows, err := ProfileQuery("SELECT * FROM users WHERE active = ?", 1)

// Profile a QueryRow
row := ProfileQueryRow("SELECT * FROM users WHERE id = ?", userID)

// Profile an Exec
result, err := ProfileExec("UPDATE users SET last_login = ? WHERE id = ?", time.Now(), userID)
```

---

## Performance Optimization Workflow

### 1. Enable Profiler in Development

```bash
export QUERY_PROFILER_ENABLED=true
export QUERY_PROFILER_THRESHOLD_MS=50
./zrp
```

### 2. Run Your Application

Perform normal operations or run load tests.

### 3. Check Slow Queries

```bash
curl http://localhost:8080/api/debug/slow-queries | jq
```

### 4. Analyze and Optimize

For each slow query:
- Check if indexes exist on WHERE/JOIN columns
- Verify EXPLAIN QUERY PLAN in SQLite
- Consider query restructuring
- Add composite indexes if needed

### 5. Verify Improvements

```bash
# Reset profiler
curl -X POST http://localhost:8080/api/debug/query-reset

# Re-run tests
# Check stats again
curl http://localhost:8080/api/debug/query-stats | jq
```

---

## SQLite Query Plan Analysis

Use SQLite's EXPLAIN QUERY PLAN to understand query execution:

```bash
sqlite3 zrp.db "EXPLAIN QUERY PLAN SELECT * FROM inventory WHERE ipn = 'CAP-001-0001';"
```

**Example Output:**

```
QUERY PLAN
`--SEARCH inventory USING INDEX sqlite_autoindex_inventory_1 (ipn=?)
```

Good indicators:
- `SEARCH ... USING INDEX` - Index is being used ✅
- `SCAN` - Full table scan ⚠️
- `USING COVERING INDEX` - Best case (all data in index) ✅

---

## Best Practices

### 1. Profile in Development, Monitor in Production

- **Development:** Low threshold (50-100ms) to catch all potential issues
- **Production:** Higher threshold (200-500ms) to focus on critical slowdowns

### 2. Regular Monitoring

Set up automated alerts when slow query count exceeds threshold:

```bash
#!/bin/bash
SLOW_COUNT=$(curl -s http://localhost:8080/api/debug/query-stats | jq '.slow_queries')
if [ "$SLOW_COUNT" -gt 10 ]; then
    echo "Alert: $SLOW_COUNT slow queries detected!"
fi
```

### 3. Periodic Reset

Reset the profiler daily/weekly to avoid memory buildup:

```bash
curl -X POST http://localhost:8080/api/debug/query-reset
```

### 4. Log Rotation

Rotate `slow_queries.log` to prevent disk space issues:

```bash
logrotate /etc/logrotate.d/zrp-slow-queries
```

---

## Troubleshooting

### Profiler Not Recording Queries

**Cause:** Direct `db.Query()` calls bypass profiler  
**Solution:** Use `ProfileQuery()`, `ProfileQueryRow()`, `ProfileExec()` wrappers

### High Memory Usage

**Cause:** Too many queries stored in memory (max 1000)  
**Solution:** Reset profiler regularly or reduce retention

### Missing Slow Queries in Log

**Cause:** Log file permissions or disk space  
**Solution:** Check file permissions and disk space

---

## Performance Targets

| Query Type | Target | Acceptable | Slow |
|------------|--------|------------|------|
| Primary key lookup | < 1ms | < 5ms | > 10ms |
| Indexed filter | < 5ms | < 20ms | > 50ms |
| JOIN (2-3 tables) | < 10ms | < 50ms | > 100ms |
| Complex aggregation | < 50ms | < 200ms | > 500ms |
| Full table scan | Avoid | < 100ms | > 200ms |

---

## Integration with Monitoring

### Prometheus Metrics (Future)

```go
// TODO: Export metrics
prometheus_slow_queries_total
prometheus_avg_query_duration
prometheus_query_count_total
```

### Grafana Dashboard (Future)

- Query duration histogram
- Slow query count over time
- Top 10 slowest queries

---

## See Also

- [DATABASE_PERFORMANCE_AUDIT.md](../DATABASE_PERFORMANCE_AUDIT.md)
- [SQLite Query Optimization](https://www.sqlite.org/optoverview.html)
- [SQLite Indexes](https://www.sqlite.org/queryplanner.html)

---

**Updated:** 2026-02-19  
**Version:** 1.0  
**Maintainer:** ZRP Development Team
