# Fixing Concurrent Map Panics in Error Libraries

**Critical pre-step before adding goroutine panic recovery.**

## Check if Fix is Needed

**Step 1: Check if goutils/errors.New is used**
```bash
# Check for errors.New usage from goutils
grep -r "errors\.New(" --include="*.go" | grep -v vendor | head -10
```

If `errors.New()` is from `github.com/razorpay/goutils/errors` → **FIX IS REQUIRED**

**Why:** The goutils/errors library has a concurrent map access bug. When `errors.New()` (reads map without lock) and `errors.Register()` (writes map with lock) run concurrently, it causes a fatal crash that cannot be caught by panic recovery.

## How to Fix

**Step 2: Find all files that define error code constants**
```bash
# Search for files with error code constants
find . -name "*.go" -type f | xargs grep -l 'const.*ERROR' | grep -v vendor | grep -v _test.go

# Common file names:
# - error_code.go, trace.go, errors.go, constants.go
# - Packages: internal/errors, internal/trace, internal/rzperrors, pkg/errors
```

List out ALL files found. These need error registration.

**Step 3: Register all error codes at startup**

For **EACH file** containing error constants found in Step 2:

### 3.1: Create an error code map

Add at the end of the file:
```go
// AllErrorCodes maps all error code constants to descriptions
// Add new error codes to this map when adding const declarations above
var AllErrorCodes = map[string]string{
    CodeSerializationError: "Serialization error",
    CodeDatabaseError:      "Database error",
    // ... all other error codes from this file
}
```

### 3.2: Create a registration file

Create `register.go` in same package:
```go
package errors  // or trace, or whatever the package name is

import (
    "github.com/razorpay/goutils/errors"
)

// RegisterAllErrors pre-registers all error codes from this package
// to prevent concurrent map read/write panics.
// MUST be called during bootstrap before any goroutines spawn.
func RegisterAllErrors() {
    for code := range AllErrorCodes {
        registerError(code)
    }
}

func registerError(code string) {
    public := &errors.Public{
        Code:        code,
        Description: AllErrorCodes[code],
    }
    errors.Register(errors.IdentifierCode(code), public, errors.ErrorCode(code))
}
```

### 3.3: Call registration in bootstrap/init

In `bootstrap.go` or `main.go`:
```go
func Init(env string, mode string, sqlDriver *sql.DB) error {
    var err error

    // CRITICAL: Pre-register ALL error codes BEFORE any concurrent operations
    // This prevents concurrent map read/write panics in goutils/errors library
    // Must be called before any goroutines spawn or errors.New() is called

    // Register from all packages that define error codes
    rzperrors.RegisterAllErrors()  // if you have this package
    trace.RegisterAllTraces()       // if you have this package
    // ... register from any other error definition packages

    // ... rest of initialization
}
```

### 3.4: Verify no duplicates

```bash
# Check for duplicate error code values in the map
go build ./...
# Will fail with "duplicate key" errors if any exist
```

## Maintenance

When adding a new error code:
1. Add `const` declaration (existing pattern)
2. Add entry to `AllErrorCodes` map in same file
3. That's it - auto-registers on next startup

**This step is MANDATORY** if using goutils/errors library. Skip goroutine panic recovery if this isn't fixed first, as the service will continue crashing from concurrent map panics.
