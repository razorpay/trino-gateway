# WDA Query Translator — Examples

Complete SQL-to-WDA translation examples covering all common patterns.

---

## Example 1: Simple SELECT with Filter

**SQL:**
```sql
SELECT * FROM coupons WHERE code NOT LIKE '%TEST%' ORDER BY entity_id DESC LIMIT 100;
```

**WDA Go Code:**
```go
builder := query_builder.NewWdaQueryBuilder()

// SELECT *
builder.AddFetch("coupons", "*")

// FROM coupons
builder.AddResource("coupons")

// WHERE code NOT LIKE '%TEST%'
builder.AddFilter("coupons", "code", []interface{}{"%TEST%"}, query_builder.NotLike, query_builder.Nil)

// ORDER BY entity_id DESC
builder.AddSort("coupons", "entity_id", query_builder.Desc)

// LIMIT 100, database = api-beta, cluster = admin
builder.Namespace("api-beta").Cluster("admin").Skip(0).Size(100)

request := builder.Build()
```

---

## Example 2: INNER JOIN with Alias

**SQL:**
```sql
SELECT m.id, m.name, m.email, p.id AS payments_id, p.merchant_id AS m_id
FROM api-beta.merchants m
INNER JOIN api-beta.payments p ON p.merchant_id = m.id
WHERE p.merchant_id = 'CgmJ7mopI795Pz'
ORDER BY p.created_at DESC
LIMIT 100 OFFSET 10;
```

**WDA Go Code:**
```go
builder := query_builder.NewWdaQueryBuilder()

// SELECT columns with aliases
builder.AddFetch("merchants", "id").
    AddFetch("merchants", "name").
    AddFetch("merchants", "email").
    AddFetchWithAlias("payments", "id", "payments_id").
    AddFetchWithAlias("payments", "merchant_id", "m_id")

// FROM merchants INNER JOIN payments ON ...
builder.AddResourceWithJoin("merchants", "payments", "inner",
    "payments.merchant_id = merchants.id")

// WHERE payments.merchant_id = 'CgmJ7mopI795Pz'
builder.AddFilter("payments", "merchant_id",
    []interface{}{"CgmJ7mopI795Pz"}, query_builder.Eq, query_builder.Nil)

// ORDER BY payments.created_at DESC
builder.AddSort("payments", "created_at", query_builder.Desc)

// LIMIT 100 OFFSET 10
builder.Namespace("api-beta").Cluster("admin").Skip(10).Size(100)

request := builder.Build()
```

---

## Example 3: Cross-Database JOIN with Aggregation

**SQL:**
```sql
SELECT COUNT(api.payments.*) AS total_orders
FROM api.payments
INNER JOIN prod_pg_router.orders ON api.payments.order_id = prod_pg_router.orders.id
WHERE api.payments.created_at >= 1669028400
  AND api.payments.created_at <= 1669032000
LIMIT 100 OFFSET 10;
```

**WDA Go Code:**
```go
builder := query_builder.NewWdaQueryBuilder()

// SELECT COUNT(*) AS total_orders — cross-DB uses full db.table as resource key
builder.AddFetchWithFuncAlias("api.payments", "*", "count", "total_orders")

// FROM api.payments INNER JOIN prod_pg_router.orders ON ...
builder.AddResourceWithJoin("api.payments", "prod_pg_router.orders", "inner",
    "api.payments.order_id = prod_pg_router.orders.id")

// WHERE — first filter uses Nil operator, second uses And
builder.AddFilter("api.payments", "created_at",
    []interface{}{1669028400}, query_builder.Gte, query_builder.Nil)
builder.AddFilter("api.payments", "created_at",
    []interface{}{1669032000}, query_builder.Lte, query_builder.And)

// No Namespace — embedded in resource names for cross-DB
builder.Cluster("admin").Skip(10).Size(100)

request := builder.Build()
```

---

## Example 4: GROUP BY with COUNT

**SQL:**
```sql
SELECT COUNT(*) FROM coupons
WHERE code NOT LIKE '%TEST%'
GROUP BY merchant_id;
```

**WDA Go Code:**
```go
builder := query_builder.NewWdaQueryBuilder()

// SELECT COUNT(*)
builder.AddFetchWithFunction("coupons", "*", "count")

// FROM coupons
builder.AddResource("coupons")

// WHERE code NOT LIKE '%TEST%'
builder.AddFilter("coupons", "code", []interface{}{"%TEST%"}, query_builder.NotLike, query_builder.Nil)

// GROUP BY merchant_id
builder.AddGroup("coupons", "merchant_id")

builder.Namespace("api-beta").Cluster("admin")

request := builder.Build()
```

---

## Example 5: SubFilters (Parenthesized OR Conditions)

**SQL:**
```sql
SELECT COUNT(*) FROM coupons
WHERE code NOT LIKE '%TEST%'
  AND (entity_type = 'promotion' OR entity_id = 'IyxMZbvdWaMThj');
```

**WDA Go Code:**
```go
// Step 1: Create sub-filter conditions
filter1, _ := query_builder.GetWdaFilter("coupons", "entity_type",
    []interface{}{"promotion"}, query_builder.Eq, query_builder.Nil)
filter2, _ := query_builder.GetWdaFilter("coupons", "entity_id",
    []interface{}{"IyxMZbvdWaMThj"}, query_builder.Eq, query_builder.Or)

// Step 2: Build query
builder := query_builder.NewWdaQueryBuilder()
builder.AddFetchWithFunction("coupons", "*", "count")
builder.AddResource("coupons")

// Regular filter
builder.AddFilter("coupons", "code", []interface{}{"%TEST%"}, query_builder.NotLike, query_builder.Nil)

// Grouped sub-filter (the AND connects this group to the previous filter)
builder.WithSubFilter(query_builder.And, filter1, filter2)

builder.Namespace("api-beta").Cluster("admin")

request := builder.Build()
```

**Key pattern:** Inside the sub-filter group, the first condition uses `Nil` and subsequent ones use their logical operator (`Or` here). The `WithSubFilter` call's first argument (`And`) connects the group to the rest of the WHERE clause.

---

## Example 6: Secondary JOIN (Three-Table JOIN)

**SQL:**
```sql
SELECT m.id, m.name, m.email, p.id AS payments_id, p.merchant_id AS m_id
FROM merchants
INNER JOIN payments ON payments.merchant_id = merchants.id
INNER JOIN merchant_details ON merchants.id = merchant_details.merchant_id
WHERE payments.merchant_id = 'CgmJ7mopI795Pz'
ORDER BY payments.created_at DESC
LIMIT 100 OFFSET 10;
```

**WDA Go Code:**
```go
builder := query_builder.NewWdaQueryBuilder()

builder.AddFetch("merchants", "id").
    AddFetch("merchants", "name").
    AddFetch("merchants", "email").
    AddFetchWithAlias("payments", "id", "payments_id").
    AddFetchWithAlias("payments", "merchant_id", "m_id")

// First JOIN
builder.AddResourceWithJoin("merchants", "payments", "inner",
    "payments.merchant_id = merchants.id")

// Second JOIN (secondary — no resource1, chains onto existing join)
builder.AddResourceWithSecondaryJoin("merchant_details", "inner",
    "merchants.id=merchant_details.merchant_id")

builder.AddFilter("payments", "merchant_id",
    []interface{}{"CgmJ7mopI795Pz"}, query_builder.Eq, query_builder.Nil)

builder.AddSort("payments", "created_at", query_builder.Desc)
builder.Namespace("api-beta").Cluster("admin").Skip(10).Size(100)

request := builder.Build()
```

---

## Example 7: Index Hints

**SQL:**
```sql
SELECT * FROM payments USE INDEX (payments_created_at_index)
WHERE created_at >= 1718085823
ORDER BY created_at DESC
LIMIT 100;
```

**WDA Go Code:**
```go
index1 := query_builder.GetIndexHint(query_builder.Use, []string{"payments_created_at_index"})

builder := query_builder.NewWdaQueryBuilder()
builder.AddFetch("payments", "*")
builder.AddResource("payments")
builder.AddQueryIndexHints(&index1)

builder.AddFilter("payments", "created_at",
    []interface{}{1718085823}, query_builder.Gte, query_builder.Nil)
builder.AddSort("payments", "created_at", query_builder.Desc)

builder.Namespace("api").Cluster("admin").Size(100)

request := builder.Build()
```

---

## Example 8: GROUP BY with HAVING

**SQL:**
```sql
SELECT merchant_id, SUM(amount) AS total_amount, COUNT(*) AS payment_count
FROM payments
WHERE created_at >= 1609459200
GROUP BY merchant_id
HAVING total_amount > 10000 AND payment_count >= 5
ORDER BY total_amount DESC
LIMIT 100;
```

**WDA Go Code:**
```go
builder := query_builder.NewWdaQueryBuilder()

// SELECT columns with aggregations
builder.AddFetch("payments", "merchant_id")
builder.AddFetchWithFuncAlias("payments", "amount", "sum", "total_amount")
builder.AddFetchWithFuncAlias("payments", "*", "count", "payment_count")

// FROM
builder.AddResource("payments")

// WHERE
builder.AddFilter("payments", "created_at",
    []interface{}{1609459200}, query_builder.Gte, query_builder.Nil)

// GROUP BY
builder.AddGroup("payments", "merchant_id")

// HAVING — uses alias names from SELECT, first uses Nil, second uses And
builder.AddHaving("payments", "total_amount",
    []interface{}{10000}, query_builder.Gt, query_builder.Nil)
builder.AddHaving("payments", "payment_count",
    []interface{}{5}, query_builder.Gte, query_builder.And)

// ORDER BY
builder.AddSort("payments", "total_amount", query_builder.Desc)

builder.Namespace("api").Cluster("admin").Size(100)

request := builder.Build()
```

---

## Example 9: IN and NOT IN

**SQL:**
```sql
SELECT * FROM payments
WHERE status NOT IN ('failed', 'cancelled')
  AND merchant_id IN ('mid_1', 'mid_2', 'mid_3');
```

**WDA Go Code:**
```go
builder := query_builder.NewWdaQueryBuilder()

builder.AddFetch("payments", "*")
builder.AddResource("payments")

// NOT IN — pass all values as a single slice
builder.AddFilter("payments", "status",
    []interface{}{"failed", "cancelled"}, query_builder.NotIn, query_builder.Nil)

// IN
builder.AddFilter("payments", "merchant_id",
    []interface{}{"mid_1", "mid_2", "mid_3"}, query_builder.In, query_builder.And)

builder.Namespace("api").Cluster("admin")

request := builder.Build()
```

---

## Example 10: BETWEEN

**SQL:**
```sql
SELECT * FROM payments
WHERE amount BETWEEN 1000 AND 50000
  AND created_at >= 1609459200
ORDER BY created_at ASC
LIMIT 500;
```

**WDA Go Code:**
```go
builder := query_builder.NewWdaQueryBuilder()

builder.AddFetch("payments", "*")
builder.AddResource("payments")

// BETWEEN — pass both bounds as a 2-element slice
builder.AddFilter("payments", "amount",
    []interface{}{1000, 50000}, query_builder.Between, query_builder.Nil)

builder.AddFilter("payments", "created_at",
    []interface{}{1609459200}, query_builder.Gte, query_builder.And)

builder.AddSort("payments", "created_at", query_builder.Asc)

builder.Namespace("api").Cluster("admin").Size(500)

request := builder.Build()
```

---

## Example 11: IS NULL / IS NOT NULL

**SQL:**
```sql
SELECT * FROM merchants
WHERE activated_at IS NOT NULL
  AND suspended_at IS NULL
LIMIT 200;
```

**WDA Go Code:**
```go
builder := query_builder.NewWdaQueryBuilder()

builder.AddFetch("merchants", "*")
builder.AddResource("merchants")

// IS NOT NULL — no values needed, pass empty slice
builder.AddFilter("merchants", "activated_at",
    []interface{}{}, query_builder.NotNull, query_builder.Nil)

// IS NULL
builder.AddFilter("merchants", "suspended_at",
    []interface{}{}, query_builder.Null, query_builder.And)

builder.Namespace("api").Cluster("admin").Size(200)

request := builder.Build()
```

---

## Example 12: Arithmetic in SELECT

**SQL:**
```sql
SELECT SUM(amount) / 100 AS amount_in_rupees FROM payments
WHERE merchant_id = 'mid_123'
GROUP BY merchant_id;
```

**WDA Go Code:**
```go
builder := query_builder.NewWdaQueryBuilder()

// Arithmetic: SUM(amount) / 100 AS amount_in_rupees
builder.AddFetchWithFuncAliasArthm("payments", "amount", "sum", "amount_in_rupees", "/", "100")

builder.AddResource("payments")
builder.AddFilter("payments", "merchant_id",
    []interface{}{"mid_123"}, query_builder.Eq, query_builder.Nil)
builder.AddGroup("payments", "merchant_id")

builder.Namespace("api").Cluster("admin")

request := builder.Build()
```

---

## Example 13: Batch Queries

**SQL (two separate queries to execute together):**
```sql
-- Query 1
SELECT id, name FROM users WHERE id = 1;
-- Query 2
SELECT id, amount FROM orders WHERE user_id = 1;
```

**WDA Go Code:**
```go
// Build Query 1
qb1 := query_builder.NewWdaQueryBuilder()
qb1.AddFetch("users", "id")
qb1.AddFetch("users", "name")
qb1.AddResource("users")
qb1.AddFilter("users", "id", []interface{}{1}, query_builder.Eq, query_builder.Nil)

// Build Query 2
qb2 := query_builder.NewWdaQueryBuilder()
qb2.AddFetch("orders", "id")
qb2.AddFetch("orders", "amount")
qb2.AddResource("orders")
qb2.AddFilter("orders", "user_id", []interface{}{1}, query_builder.Eq, query_builder.Nil)

// Batch them together
bqb := query_builder.NewWdaBatchQueryBuilder()
bqb.AddQuery(&qb1)
bqb.AddQuery(&qb2)

batchRequest := bqb.Build()
```

---

## Example 14: Registered Query (Raw SQL Fallback)

For SQL that cannot be expressed with the builder (subqueries, UNION, window functions, etc.):

```go
rqb := query_builder.NewWdaRegstrQryBuilder()
rqb.SQLStatement("SELECT merchant_id, COUNT(*) as cnt FROM api.payments WHERE created_at >= @val1 AND created_at <= @val2 GROUP BY merchant_id HAVING cnt > @val3")
rqb.Cluster("admin")
rqb.AddParam("val1", 1609459200)
rqb.AddParam("val2", 1612137600)
rqb.AddParam("val3", 10)

registeredQuery := rqb.Build()
```

**Note:** Registered queries must be pre-approved by WDA admins before execution. Use this only when the builder API cannot express the query.

---

## Example 15: Executing the Query via WDA Client

```go
import (
    "context"
    "net/http"

    query_builder "github.com/razorpay/goutils/wda/query-builder"
    wda_client "github.com/razorpay/goutils/wda/wda-client"
)

func main() {
    // Initialize client
    client := wda_client.NewWdaClient(
        "http://wda-service.razorpay.com:9400",  // WDA service URL
        "your_username",                           // Auth username
        "your_password",                           // Auth password
        &http.Client{},
    )

    // Build query
    builder := query_builder.NewWdaQueryBuilder()
    builder.AddFetch("payments", "*")
    builder.AddResource("payments")
    builder.AddFilter("payments", "merchant_id",
        []interface{}{"mid_123"}, query_builder.Eq, query_builder.Nil)
    builder.Namespace("api").Cluster("admin").Size(100)

    request := builder.Build()

    // Execute
    ctx := context.Background()
    response, err := client.Query(ctx, &request)
    if err != nil {
        // handle error
    }

    // response.Size — number of rows
    // response.Time — query execution time
    // response.Entities — result rows as []structpb.Struct
}
```

---

## Common Convenience Methods (No Builder Needed)

The WDA client provides helper methods for frequent patterns:

```go
// Fetch records by IDs
response, err := client.FetchByIDs(ctx, "api", "payments", "id",
    []string{"pay_1", "pay_2"}, nil, 100, 0, "admin")

// Fetch by IDs with specific columns
columns := []string{"id", "amount", "status", "merchant_id"}
response, err := client.FetchByIDs(ctx, "api", "payments", "id",
    []string{"pay_1", "pay_2"}, columns, 100, 0, "admin")

// Fetch by date range
response, err := client.FetchByCreatedAt(ctx, "api", "payments",
    startTime, endTime, query_builder.Desc, nil, 100, 0, "admin")

// Fetch by primary keys
response, err := client.FetchByPrimaryKeys(ctx, "api", "payments", "merchant_id",
    []string{"mid_1", "mid_2"}, nil, 100, 0, "admin")
```
