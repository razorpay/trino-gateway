---
name: database
description: Standards for SQL interactions within a Domain package.
---

# Database Standards

## 1. Domain Repository
Database logic belongs in `repository.go` inside the domain package.
It should interact via a `Repository` struct (or interface if needed for testing).

## 2. No ORMs
Use `database/sql` with raw queries.

## 3. Mandatory Context
Every method signature MUST start with `ctx context.Context`.

## 4. Error Wrapping
Wrap errors with the package/domain name for clarity:

Example: 
```go
// internal/talk/repository.go
func (r *Repository) Get(ctx context.Context, id uuid.UUID) (*Talk, error) {
    // ...
    if err != nil {
        return nil, fmt.Errorf("get talk: %w", err)
    }
}
```