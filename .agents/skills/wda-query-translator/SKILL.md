---
name: wda-query-translator
description: "Translate standard SQL queries into WDA Query Builder Go code using the goutils/wda/query-builder SDK, and validate the translation against the WDA stage API. Use when users want to: (1) Convert SQL to WDA SDK Go code, (2) Generate WDA Query JSON from SQL, (3) Validate WDA query translations against the stage API. Trigger phrases include 'translate SQL to WDA', 'WDA query builder', 'convert SQL to WDA SDK', 'SQL to WDA'."
user-invocable: true
disable-model-invocation: true
---

# WDA Query Translator — SQL to WDA SDK

**Purpose:** Translate standard SQL queries into WDA Query Builder Go code using the `github.com/razorpay/goutils/wda/query-builder` SDK, and validate the translation against the WDA stage API.

---

## How to Use

When a user provides a SQL query:
1. Load this `SKILL.md` (you're reading it now)
2. Load `reference.md` for complete API mapping and operator reference
3. Load `examples.md` for translation patterns and edge cases
4. Parse the SQL and produce Go code using the WDA Query Builder SDK
5. Also produce the **WDA Query JSON** (see JSON Format section below)
6. **Validate** the translation using the WDA stage API (see Validation section below)

---

## Quick Reference — SQL to WDA Mapping

| SQL Clause | WDA Builder Method |
|---|---|
| `SELECT col` | `AddFetch(resource, field)` |
| `SELECT col AS alias` | `AddFetchWithAlias(resource, field, alias)` |
| `SELECT COUNT(col)` | `AddFetchWithFunction(resource, field, "count")` |
| `SELECT SUM(col) AS alias` | `AddFetchWithFuncAlias(resource, field, "sum", alias)` |
| `SELECT SUM(col) / 100 AS alias` | `AddFetchWithFuncAliasArthm(resource, field, "sum", alias, "/", "100")` |
| `FROM table` | `AddResource(table)` |
| `FROM t1 JOIN t2 ON ...` | `AddResourceWithJoin(t1, t2, joinType, onCondition)` |
| Secondary JOIN | `AddResourceWithSecondaryJoin(t2, joinType, onCondition)` |
| `WHERE col = val` | `AddFilter(resource, field, []interface{}{val}, Eq, Nil)` |
| `AND col > val` | `AddFilter(resource, field, []interface{}{val}, Gt, And)` |
| `OR col < val` | `AddFilter(resource, field, []interface{}{val}, Lt, Or)` |
| `WHERE (a OR b)` | `WithSubFilter(And, filter1, filter2)` |
| `GROUP BY col` | `AddGroup(resource, field)` |
| `HAVING alias > val` | `AddHaving(resource, alias, []interface{}{val}, Gt, Nil)` |
| `ORDER BY col DESC` | `AddSort(resource, field, Desc)` |
| `LIMIT n` | `Size(n)` |
| `OFFSET n` | `Skip(n)` |
| Database name | `Namespace(dbName)` |
| TiDB cluster target | `Cluster("admin"` or `"merchant")` |
| `USE INDEX (idx)` | `AddQueryIndexHints(&indexHint)` |

## Symbol Mapping — SQL Operators to WDA Symbols

| SQL | WDA Symbol Constant |
|---|---|
| `=` | `query_builder.Eq` |
| `!=` / `<>` | `query_builder.Neq` |
| `>` | `query_builder.Gt` |
| `>=` | `query_builder.Gte` |
| `<` | `query_builder.Lt` |
| `<=` | `query_builder.Lte` |
| `LIKE` | `query_builder.Like` |
| `NOT LIKE` | `query_builder.NotLike` |
| `BETWEEN` | `query_builder.Between` |
| `IN` | `query_builder.In` |
| `NOT IN` | `query_builder.NotIn` |
| `IS NULL` | `query_builder.Null` |
| `IS NOT NULL` | `query_builder.NotNull` |
| `JSON_CONTAINS` | `query_builder.JsonContains` |

## Logical Operator Mapping

| Position in SQL | WDA Go Constant | WDA JSON Value |
|---|---|---|
| First condition in WHERE | `query_builder.Nil` | `"_"` |
| `AND` between conditions | `query_builder.And` | `"and"` |
| `OR` between conditions | `query_builder.Or` | `"or"` |

---

## Translation Rules

### Rule 1: Resource Naming
- If the SQL uses `db.table` format (e.g., `api.payments`), use the full `db.table` as the resource name and do NOT set `Namespace()`
- If the SQL uses bare table names (e.g., `payments`), use the table name as resource and set `Namespace("db_name")`
- The resource name used in `AddFetch`, `AddFilter`, `AddSort`, `AddGroup`, `AddHaving` must match the resource name used in `AddResource` / `AddResourceWithJoin`

### Rule 2: First Filter Has Nil Operator
- The first filter in a WHERE clause always uses `query_builder.Nil` as the operator
- Subsequent filters use `query_builder.And` or `query_builder.Or`

### Rule 3: SubFilters for Grouped Conditions
- SQL parenthesized groups like `WHERE x = 1 AND (y = 2 OR z = 3)` require:
  1. Create individual filters with `GetWdaFilter()`
  2. Group them with `WithSubFilter(operator, filters...)`
- The first filter inside a sub-filter group uses `query_builder.Nil`
- Subsequent filters inside the group use their logical operator (`Or`, `And`)

### Rule 4: HAVING Requires GROUP BY
- WDA validates that HAVING is only used with GROUP BY
- HAVING field names typically reference aliases defined in the SELECT (fetch) clause
- Use `AddHaving()` similarly to `AddFilter()` but for aggregate conditions

### Rule 5: Defaults
- Default `size`: 100 (LIMIT)
- Default `skip`: 0 (OFFSET)
- Default `cluster`: "admin"
- Default `queryTimeout`: 180 seconds
- Max `size`: 10,000

### Rule 6: JOIN Ordering
- First join uses `AddResourceWithJoin(resource1, resource2, joinType, on)`
- Additional joins on existing tables use `AddResourceWithSecondaryJoin(resource2, joinType, on)`

### Rule 7: Cross-Database Joins
- When joining across databases, use full `db.table` format for both resources
- Do NOT set `Namespace()` — it's embedded in resource names
- Example: `AddResourceWithJoin("api.payments", "prod_pg_router.orders", "inner", "api.payments.order_id = prod_pg_router.orders.id")`

### Rule 8: Index Hints
- Create hint with `query_builder.GetIndexHint(hintType, []string{indexName})`
- Apply with `AddQueryIndexHints(&hint)` — applies to the first resource only
- For JOIN index hints, use `AddResourceWithIndexHint()`

---

## Output Template

When translating, produce **both** Go code and the WDA Query JSON, then validate.

### Step 1: Go Code

```go
import (
    query_builder "github.com/razorpay/goutils/wda/query-builder"
)

func buildQuery() wda_lib.WDAQueryRequest {
    builder := query_builder.NewWdaQueryBuilder()

    // SELECT clause
    builder.AddFetch(...)

    // FROM clause
    builder.AddResource(...)  // or AddResourceWithJoin(...)

    // WHERE clause
    builder.AddFilter(...)

    // GROUP BY clause (if applicable)
    builder.AddGroup(...)

    // HAVING clause (if applicable)
    builder.AddHaving(...)

    // ORDER BY clause (if applicable)
    builder.AddSort(...)

    // Configuration
    builder.Namespace("...").Cluster("...").Size(...).Skip(...)

    return builder.Build()
}
```

### Step 2: WDA Query JSON

Produce the equivalent JSON using camelCase keys and `"_"` for Nil operator (see "WDA Query JSON Format" section below).

### Step 3: Validate

Call the WDA stage API to convert the JSON back to SQL and compare with the original (see "Validation Step" section below). Ask the user for stage credentials if not already provided.

---

## Limitations

These SQL features are **NOT supported** by WDA Query Builder:
- `DISTINCT` — not available
- `UNION` / `INTERSECT` / `EXCEPT` — not available
- Subqueries in SELECT/FROM/WHERE — not available
- `CASE WHEN` expressions — not available
- Window functions (`ROW_NUMBER`, `RANK`, etc.) — not available
- `EXISTS` / `NOT EXISTS` — not available
- Multiple `LIKE` patterns — use separate filters with `Or`
- Complex arithmetic in WHERE — limited to fetch clause arithmetic only
- `LIMIT` > 10,000 — rejected by validation

If a SQL query uses unsupported features, clearly state the limitation and suggest the closest WDA alternative or recommend using Registered Queries (`WdaRegstrQryBuilder`) which accept raw SQL.

---

## WDA Query JSON Format

When producing the WDA Query JSON (for API validation or direct use), use **camelCase** keys matching the Twirp/protobuf JSON serialization. This is the format accepted by the WDA API.

### Key format rules:
- **Nil operator** serializes as `"_"` (underscore), NOT empty string
- **Fetch fields** use `"query"` as the array key (not `"fields"`)
- **Filter keys** are camelCase: `resourceName`, `fieldName`, `subFilter`
- **Join keys** are camelCase: `joinType`
- **Other keys**: `namespace`, `size`, `skip`, `cluster`, `querytimeout`

### JSON Template:

```json
{
  "fetch": {
    "<resource>": {
      "query": [
        { "field": "<column>" },
        { "field": "<column>", "function": "count", "alias": "total" }
      ]
    }
  },
  "resources": [
    {
      "resource1": "<table1>",
      "resource2": "<table2>",
      "joinType": "inner",
      "on": "<table1>.col = <table2>.col"
    }
  ],
  "filters": [
    {
      "operator": "_",
      "resourceName": "<table>",
      "fieldName": "<column>",
      "value": ["<val>"],
      "symbol": "eq"
    },
    {
      "operator": "and",
      "subFilter": [
        { "operator": "_", "resourceName": "<t>", "fieldName": "<c>", "value": ["v1"], "symbol": "eq" },
        { "operator": "or", "resourceName": "<t>", "fieldName": "<c>", "value": ["v2"], "symbol": "eq" }
      ]
    }
  ],
  "sort": [
    { "resource": "<table>", "field": "<column>", "order": "desc" }
  ],
  "group": [
    { "resource": "<table>", "field": "<column>" }
  ],
  "having": [
    { "operator": "_", "resourceName": "<table>", "fieldName": "<alias>", "value": [1000], "symbol": "gt" }
  ],
  "namespace": "<db_name>",
  "size": 100,
  "skip": 0,
  "cluster": "admin",
  "querytimeout": 180
}
```

### Operator values in JSON:
| Go Constant | JSON Value |
|---|---|
| `query_builder.Nil` | `"_"` |
| `query_builder.And` | `"and"` |
| `query_builder.Or` | `"or"` |

### Symbol values in JSON:
| Go Constant | JSON Value |
|---|---|
| `query_builder.Eq` | `"eq"` |
| `query_builder.Neq` | `"neq"` |
| `query_builder.Gt` | `"gt"` |
| `query_builder.Gte` | `"gte"` |
| `query_builder.Lt` | `"lt"` |
| `query_builder.Lte` | `"lte"` |
| `query_builder.Like` | `"like"` |
| `query_builder.NotLike` | `"notlike"` |
| `query_builder.Between` | `"between"` |
| `query_builder.In` | `"in"` |
| `query_builder.NotIn` | `"notin"` |
| `query_builder.Null` | `"null"` |
| `query_builder.NotNull` | `"notnull"` |
| `query_builder.JsonContains` | `"jsoncontains"` |

---

## Validation Step

After translating SQL to WDA Go code and JSON, **validate** the translation by calling the WDA stage API to convert the JSON back to SQL and comparing the result with the original query.

### Validation Workflow

1. **Ask user for credentials**: If not already provided, use `AskUserQuestion` to request the WDA stage Basic Auth credentials (Base64-encoded `username:password`).

2. **Call the WDA MySQLQueryBuilder API**: Send the generated WDA Query JSON to the stage endpoint using `curl`:

```bash
curl -s -X POST 'https://wda-service.concierge.stage.razorpay.in/twirp/rzp.wda.query_creation_library.v1.WDAQueryCreator/WDAMySQLQueryBuilder' --header 'Content-Type: application/json' --header 'Authorization: Basic <credentials>' --data '<wda_query_json>'
```

3. **Parse the response**: The API returns a JSON response containing the generated SQL query. Extract the SQL from the response.

4. **Compare**: Compare the API-generated SQL with the original user SQL query:
   - Normalize both (remove extra whitespace, lowercase keywords) for comparison
   - Check that the same tables, columns, joins, filters, groupings, and ordering are present
   - Report any **semantic differences** — the SQL won't be character-identical but should be logically equivalent
   - Flag differences as either:
     - **Cosmetic**: whitespace, keyword casing, alias positioning (OK)
     - **Semantic**: missing/extra conditions, wrong operators, different tables (NEEDS FIX)

5. **Report validation result**:
   - Show the API-returned SQL to the user
   - State whether the translation is **VALID** (semantically equivalent) or **INVALID** (has differences)
   - If invalid, explain the discrepancy and offer a corrected translation

### Validation Notes
- The API only validates that the JSON is well-formed and can produce SQL — it does NOT execute the query
- Cross-database queries (no namespace, `db.table` resource names) work with this endpoint
- If the API returns an error, the JSON structure is likely malformed — fix the JSON and retry
- The stage endpoint requires VPN/network access to `*.stage.razorpay.in`
