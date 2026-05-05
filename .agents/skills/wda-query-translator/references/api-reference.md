# WDA Query Translator — API Reference

Complete reference for all WDA Query Builder SDK types, methods, and constants.

---

## Package Import

```go
import (
    query_builder "github.com/razorpay/goutils/wda/query-builder"
    wda_lib "github.com/razorpay/goutils/wda/rpc/wda-service/query_creation_library/v1"
)
```

---

## Constants

### SortOrder
```go
query_builder.Desc  // "desc"
query_builder.Asc   // "asc"
```

### Operator (logical connectors between filters)
```go
query_builder.And  // "and"
query_builder.Or   // "or"
query_builder.Nil  // "" — used for the FIRST filter in a clause
```

### Symbol (comparison operators)
```go
query_builder.Eq           // "eq"           → SQL: =
query_builder.Neq          // "neq"          → SQL: != / <>
query_builder.Gt           // "gt"           → SQL: >
query_builder.Gte          // "gte"          → SQL: >=
query_builder.Lt           // "lt"           → SQL: <
query_builder.Lte          // "lte"          → SQL: <=
query_builder.Like         // "like"         → SQL: LIKE
query_builder.NotLike      // "notlike"      → SQL: NOT LIKE
query_builder.Between      // "between"      → SQL: BETWEEN (requires 2 values)
query_builder.In           // "in"           → SQL: IN (requires N values)
query_builder.NotIn        // "notin"        → SQL: NOT IN (requires N values)
query_builder.Null         // "null"         → SQL: IS NULL (no values needed)
query_builder.NotNull      // "notnull"      → SQL: IS NOT NULL (no values needed)
query_builder.JsonContains // "jsoncontains" → SQL: JSON_CONTAINS
```

### IndexHintType
```go
query_builder.Use    // "USE"    → USE INDEX
query_builder.Ignore // "IGNORE" → IGNORE INDEX
query_builder.Force  // "FORCE"  → FORCE INDEX
```

---

## WdaQueryBuilder Methods

### Constructor

```go
func NewWdaQueryBuilder() WdaQueryBuilder
```
Returns a builder with defaults: size=100, skip=0, cluster="admin", queryTimeout=180s.

### Build

```go
func (wqb *WdaQueryBuilder) Build() wda_lib.WDAQueryRequest
```
Returns the final protobuf request object.

---

### Fetch Methods (SELECT clause)

```go
// SELECT field
func (wqb *WdaQueryBuilder) AddFetch(resource string, field string) *WdaQueryBuilder

// SELECT FUNCTION(field)  — e.g., COUNT(*), SUM(amount)
func (wqb *WdaQueryBuilder) AddFetchWithFunction(resource string, field string,
    function string) *WdaQueryBuilder

// SELECT field AS alias
func (wqb *WdaQueryBuilder) AddFetchWithAlias(resource string, field string,
    alias string) *WdaQueryBuilder

// SELECT FUNCTION(field) AS alias  — e.g., COUNT(*) AS total
func (wqb *WdaQueryBuilder) AddFetchWithFuncAlias(resource string, field string,
    function string, alias string) *WdaQueryBuilder

// SELECT FUNCTION(field) OPERATOR OPERAND AS alias  — e.g., SUM(amount) / 100 AS rupees
func (wqb *WdaQueryBuilder) AddFetchWithFuncAliasArthm(resource string, field string,
    function string, alias string, arthmOpr string, operand string) *WdaQueryBuilder
```

**Supported SQL functions:** `count`, `sum`, `avg`, `min`, `max`, `first`, `last`, `group_concat`, `abs`, `round`, `ceil`, `floor`, `coalesce`

**Supported arithmetic operators:** `+`, `-`, `*`, `/`

**Notes:**
- `resource` = table name (or `db.table` for cross-DB queries)
- `field` = column name, or `"*"` for wildcard
- Multiple `AddFetch` calls on the same resource accumulate columns
- All fetch methods return `*WdaQueryBuilder` for chaining

---

### Resource Methods (FROM / JOIN clause)

```go
// FROM resource1
func (wqb *WdaQueryBuilder) AddResource(resource1 string) *WdaQueryBuilder

// FROM resource1 JOIN resource2 ON condition
func (wqb *WdaQueryBuilder) AddResourceWithJoin(resource1 string, resource2 string,
    jointype string, on string) *WdaQueryBuilder

// ... JOIN resource2 ON condition  (chains onto existing join)
func (wqb *WdaQueryBuilder) AddResourceWithSecondaryJoin(resource2 string,
    jointype string, on string) *WdaQueryBuilder

// ... JOIN resource2 ON condition USE/FORCE/IGNORE INDEX (...)
func (wqb *WdaQueryBuilder) AddResourceWithIndexHint(resource2 string,
    jointype string, on string, indexHints ...*wda_lib.IndexHint) *WdaQueryBuilder

// Apply index hints to the first resource
func (wqb *WdaQueryBuilder) AddQueryIndexHints(indexHints ...*wda_lib.IndexHint) *WdaQueryBuilder
```

**Join types:** `"inner"`, `"left"`, `"right"`, `"cross"`

**Resource naming:**
- Same database: use bare table name + set `Namespace()` → `"payments"`
- Cross-database: use `db.table` format, don't set `Namespace()` → `"api.payments"`

---

### Filter Methods (WHERE clause)

```go
// WHERE resource.field SYMBOL value
func (wqb *WdaQueryBuilder) AddFilter(resourceName string, fieldName string,
    values []interface{}, symbol Symbol, operator Operator) (*WdaQueryBuilder, error)

// WHERE JSON_CONTAINS(resource.field, value, path)
func (wqb *WdaQueryBuilder) AddFilterWithPath(resourceName string, fieldName string,
    values []interface{}, symbol Symbol, path string, operator Operator) (*WdaQueryBuilder, error)

// WHERE ... AND (subfilter1 OR subfilter2)
func (wqb *WdaQueryBuilder) WithSubFilter(operator Operator,
    subFilters ...wda_lib.WDAFilters) (*WdaQueryBuilder, error)
```

**Helper to create filter objects for sub-filters:**
```go
func GetWdaFilter(resourceName string, fieldName string,
    values []interface{}, symbol Symbol, operator Operator) (wda_lib.WDAFilters, error)
```

**Value formats by symbol:**
| Symbol | `values` parameter |
|---|---|
| Eq, Neq, Gt, Gte, Lt, Lte, Like, NotLike | `[]interface{}{singleValue}` |
| In, NotIn | `[]interface{}{val1, val2, ...}` |
| Between | `[]interface{}{lowerBound, upperBound}` |
| Null, NotNull | `[]interface{}{}` (empty) |
| JsonContains | `[]interface{}{jsonValue}` |

---

### Sorting Methods (ORDER BY clause)

```go
func (wqb *WdaQueryBuilder) AddSort(resource string, field string,
    order SortOrder) *WdaQueryBuilder
```

---

### Grouping Methods (GROUP BY clause)

```go
func (wqb *WdaQueryBuilder) AddGroup(resource string, field string) *WdaQueryBuilder
```

---

### Having Methods (HAVING clause)

```go
// HAVING resource.field SYMBOL value
func (wqb *WdaQueryBuilder) AddHaving(resourceName string, fieldName string,
    values []interface{}, symbol Symbol, operator Operator) (*WdaQueryBuilder, error)

// HAVING JSON_CONTAINS(...)
func (wqb *WdaQueryBuilder) AddHavingWithPath(resourceName string, fieldName string,
    values []interface{}, symbol Symbol, path string, operator Operator) (*WdaQueryBuilder, error)

// HAVING ... AND (sub1 OR sub2)
func (wqb *WdaQueryBuilder) WithSubHaving(operator Operator,
    subHavings ...wda_lib.WDAFilters) (*WdaQueryBuilder, error)
```

**Helper to create having objects:**
```go
func GetWdaHaving(resourceName string, fieldName string,
    values []interface{}, symbol Symbol, operator Operator) (wda_lib.WDAFilters, error)
```

**HAVING validation rules:**
- HAVING requires GROUP BY — WDA rejects queries with HAVING but no GROUP BY
- HAVING field names can reference aliases defined in the fetch (SELECT) clause
- The `resourceName` parameter is still required even for aliases (use the table name)

---

### Pagination & Configuration

```go
// LIMIT — max 10,000
func (wqb *WdaQueryBuilder) Size(size int64) (*WdaQueryBuilder, error)

// OFFSET
func (wqb *WdaQueryBuilder) Skip(skip int64) *WdaQueryBuilder

// Database name (e.g., "api", "api-beta")
func (wqb *WdaQueryBuilder) Namespace(nameSpace string) *WdaQueryBuilder

// Target cluster: "admin" or "merchant"
func (wqb *WdaQueryBuilder) Cluster(cluster string) *WdaQueryBuilder

// Query timeout in seconds (default 180, only meaningful for async)
func (wqb *WdaQueryBuilder) QueryTimeout(queryTime int64) *WdaQueryBuilder

// Client-provided query identifier for tracking
func (wqb *WdaQueryBuilder) ClientQueryUID(clientQueryUID string) *WdaQueryBuilder
```

---

### Index Hint Helper

```go
func GetIndexHint(hintType IndexHintType, indexNames []string) wda_lib.IndexHint
```

---

## WdaBatchQueryBuilder

For executing multiple queries in a single request.

```go
func NewWdaBatchQueryBuilder() WdaBatchQueryBuilder
func (bqb *WdaBatchQueryBuilder) AddQuery(qb *WdaQueryBuilder) *WdaBatchQueryBuilder
func (bqb *WdaBatchQueryBuilder) Build() wda_lib.BatchWDAQueryRequest
```

---

## WdaAsyncReqBuilder

For submitting asynchronous queries with callback.

```go
func NewWdaAsyncReqBuilder() WdaAsyncReqBuilder
func (arb *WdaAsyncReqBuilder) Request(req *wda_lib.WDAQueryRequest) *WdaAsyncReqBuilder
func (arb *WdaAsyncReqBuilder) Callback(callback string) *WdaAsyncReqBuilder
func (arb *WdaAsyncReqBuilder) Build() wda_lib.AsyncRequest
```

---

## WdaRegstrQryBuilder

For pre-registered raw SQL queries (fallback for unsupported SQL features).

```go
func NewWdaRegstrQryBuilder() WdaRegstrQryBuilder
func (rqb *WdaRegstrQryBuilder) SQLStatement(statement string) *WdaRegstrQryBuilder
func (rqb *WdaRegstrQryBuilder) Cluster(cluster string) *WdaRegstrQryBuilder
func (rqb *WdaRegstrQryBuilder) Params(params map[string]interface{}) *WdaRegstrQryBuilder
func (rqb *WdaRegstrQryBuilder) AddParam(key string, value interface{}) *WdaRegstrQryBuilder
func (rqb *WdaRegstrQryBuilder) Build() wda_lib.RegisteredQueryObject
```

Parameter placeholders in SQL: `@val1`, `@val2`, etc.

---

## WDA Client Methods

```go
// Initialize
func NewWdaClient(serverURL, username, password string, httpClient *http.Client) WdaClient

// Sync query
func (c *WdaClient) Query(ctx context.Context, req *wda_lib.WDAQueryRequest) (*wda_lib.TidbSqlResponse, error)

// Batch query
func (c *WdaClient) BatchQuery(ctx context.Context, req *wda_lib.BatchWDAQueryRequest) (*wda_lib.TidbSqlResponse, error)

// Async query
func (c *WdaClient) AsyncQuery(ctx context.Context, req *wda_lib.AsyncRequest) (*wda_lib.AsyncTptObject, error)
func (c *WdaClient) AsyncStatus(ctx context.Context, req *wda_lib.AsyncTptObject) (*wda_lib.AsyncTptObject, error)
func (c *WdaClient) AsyncResponseFetch(ctx context.Context, req *wda_lib.AsyncTptObject) (*wda_lib.TidbSqlResponse, error)

// Convenience methods
func (c *WdaClient) FetchByIDs(ctx context.Context, namespace, resource, idField string,
    ids []string, columns []string, limit, skip int, cluster string) (*wda_lib.TidbSqlResponse, error)
func (c *WdaClient) FetchByPrimaryKeys(ctx context.Context, namespace, resource, pkField string,
    values []string, columns []string, limit, skip int, cluster string) (*wda_lib.TidbSqlResponse, error)
func (c *WdaClient) FetchByCreatedAt(ctx context.Context, namespace, resource string,
    startTime, endTime int64, order query_builder.SortOrder, columns []string,
    limit, skip int, cluster string) (*wda_lib.TidbSqlResponse, error)
func (c *WdaClient) FetchByUpdatedAt(ctx context.Context, namespace, resource string,
    startTime, endTime int64, order query_builder.SortOrder, columns []string,
    limit, skip int, cluster string) (*wda_lib.TidbSqlResponse, error)
```

### Response Structure
```go
type TidbSqlResponse struct {
    Size      int64              // Number of rows returned
    Time      string             // Query execution time
    Entities  []*structpb.Struct // Result rows
    Requestid string             // Request tracking ID
}
```

---

## Generated Protobuf Types (for reference)

### WDAQueryRequest
```protobuf
message WDAQueryRequest {
    map<string, RepeatedWDAFetch> fetch = 1;
    repeated WDAResources resources = 2;
    repeated WDAFilters filters = 3;
    repeated WDASorting sort = 4;
    repeated WDASorting group = 5;
    repeated WDAFilters having = 6;
    int64 size = 7;
    int64 skip = 8;
    string namespace = 9;
    string cluster = 10;
    int64 querytimeout = 11;
    string clientqueryuid = 12;
}
```

### WDAFetch
```protobuf
message WDAFetch {
    string field = 1;
    string function = 2;
    string alias = 3;
    string arthm_opr = 4;
    string operand = 5;
}
```

### WDAResources
```protobuf
message WDAResources {
    string resource1 = 1;
    string resource2 = 2;
    string join_type = 3;
    string on = 4;
    repeated IndexHint index_hint = 5;
}
```

### WDAFilters
```protobuf
message WDAFilters {
    string operator = 1;
    string resource_name = 2;
    string field_name = 3;
    google.protobuf.ListValue value = 4;
    string symbol = 5;
    string path = 6;
    repeated WDAFilters sub_filter = 7;
}
```

### WDASorting
```protobuf
message WDASorting {
    string resource = 1;
    string field = 2;
    string order = 3;
    string function = 4;
}
```
