# Path Detection Reference

This reference describes the exact workflow for detecting the helmfile directory path during skill execution.

## ⚠️ CRITICAL: Helm Chart Location

**Once the helmfile directory is detected, ALWAYS create/update helm charts in `<detected-helmfile-dir>/charts/<application-name>/` ONLY.**

The detected helmfile directory should be within the kube-manifests repository, and charts MUST be placed in the `charts/` subdirectory within it.

## When to Run Path Detection

Run path detection at the START of any deployment, validation, or debugging operation that requires access to helmfile directory.

## Detection Workflow

### Step 1: Read Configuration File

```bash
# Read config.json from skill directory
cat agent-skills/infrastructure/skills/devstack/config.json
```

**Expected Output**:
```json
{
  "helmfile_directory": "/full/path/to/helmfile",
  "auto_detect": true,
  "fallback_paths": [...]
}
```

**Error Handling**:
- If config.json doesn't exist → Use default auto-detection with standard fallback paths
- If config.json is invalid JSON → Report error and ask user to fix it
- If missing required fields → Use defaults

### Step 2: Try Primary Path

```bash
# Check if configured path exists
if [ -d "<helmfile_directory>" ] && [ -f "<helmfile_directory>/helmfile.yaml" ]; then
  echo "✅ Found helmfile at: <helmfile_directory>"
  cd <helmfile_directory>
else
  echo "⚠️ Configured path not found, trying fallback paths..."
fi
```

### Step 3: Try Fallback Paths (if auto_detect: true)

If primary path fails and `auto_detect` is enabled:

```bash
# Try each fallback path relative to repository root
for path in "${fallback_paths[@]}"; do
  if [ -d "$path" ] && [ -f "$path/helmfile.yaml" ]; then
    echo "✅ Found helmfile at: $path"
    cd "$path"
    break
  fi
done
```

**Default Fallback Paths** (if not in config):
1. `kube-manifests/helmfile`
2. `helmfile`
3. `../kube-manifests/helmfile`

### Step 4: Auto-Clone if Not Found

If all paths fail, automatically clone the kube-manifests repository using the best available method:

**Clone priority order (MUST follow this order):**
1. **GitHub MCP server** *(highest priority)* — If a GitHub MCP server is configured (e.g. `@modelcontextprotocol/server-github`), use the MCP tool to clone `razorpay/kube-manifests`. This is the preferred method as it handles authentication automatically via the MCP server's configured token.
2. **`gh` CLI** *(fallback)* — If GitHub MCP is not available, try `gh repo clone razorpay/kube-manifests -- --depth 1 --single-branch`. Requires `gh auth login` to be completed.
3. **`git` CLI** *(fallback)* — If `gh` is not available or not authenticated, try `git clone --depth 1 --single-branch git@github.com:razorpay/kube-manifests.git`. Requires SSH key access to GitHub.
4. **Fail with message** — If none of the above work, inform the user that cloning failed and provide manual clone instructions.

**Priority 1: GitHub MCP server** — Check if a GitHub MCP tool is available. If yes, use it to clone the repository directly. Skip to verification step after clone.

**Priority 2-3: Shell fallbacks** — If GitHub MCP is not available, use the following bash commands:

```bash
echo "⚠️ Helmfile directory not found locally. Auto-cloning kube-manifests repository..."

CLONE_SUCCESS=false

# Priority 2: Try gh CLI (handles auth seamlessly)
if command -v gh &> /dev/null && gh auth status &> /dev/null; then
  echo "🔍 Using gh CLI to clone..."
  gh repo clone razorpay/kube-manifests -- --depth 1 --single-branch && CLONE_SUCCESS=true
fi

# Priority 3: Try git CLI
if [ "$CLONE_SUCCESS" = false ] && command -v git &> /dev/null; then
  echo "🔍 Using git to clone..."
  git clone --depth 1 --single-branch git@github.com:razorpay/kube-manifests.git && CLONE_SUCCESS=true
fi

# Verify clone result
if [ "$CLONE_SUCCESS" = true ] && [ -d "kube-manifests/helmfile" ] && [ -f "kube-manifests/helmfile/helmfile.yaml" ]; then
  FOUND_PATH="kube-manifests/helmfile"
  echo "✅ Successfully cloned and found helmfile at: $FOUND_PATH"
  cd "$FOUND_PATH"
else
  echo "❌ Unable to clone kube-manifests repository."
  echo "   Please clone it manually and re-run:"
  echo "   git clone --depth 1 --single-branch git@github.com:razorpay/kube-manifests.git"
  exit 1
fi
```

> **Note**: The `--depth 1 --single-branch` flags keep the clone fast and lightweight.

> **Note**: The skill no longer stops and asks the user to clone manually. It auto-clones using the best available method.

## Reporting Path to User

**Always** report which path is being used at the start of operations:

```
🔍 Using helmfile directory: /Users/parag.dudeja/.../kube-manifests/helmfile
```

This helps users verify the correct path is being used.

## Implementation Examples

### Example 1: Successful Primary Path

```
🔍 Reading configuration from config.json...
✅ Found helmfile at: /Users/parag.dudeja/Documents/Work/rzp-repos/my-service/kube-manifests/helmfile
📂 Changed to helmfile directory

Proceeding with deployment...
```

### Example 2: Fallback Path Used

```
🔍 Reading configuration from config.json...
⚠️ Primary path not found: /old/path/helmfile
🔍 Auto-detection enabled, trying fallback paths...
✅ Found helmfile at: kube-manifests/helmfile (relative to repo root)
📂 Changed to helmfile directory

Proceeding with deployment...
```

### Example 3: Path Not Found — Auto-Clone

```
🔍 Reading configuration from config.json...
⚠️ Primary path not found: /Users/parag.dudeja/wrong/path/helmfile
🔍 Auto-detection enabled, trying fallback paths...
⚠️ Helmfile directory not found locally. Auto-cloning kube-manifests...
Cloning into 'kube-manifests'...
✅ Successfully cloned. Using helmfile at: kube-manifests/helmfile
📂 Changed to helmfile directory

Proceeding with deployment...
```

## Code Template for Path Detection

Here's the exact bash code to implement path detection:

```bash
#!/bin/bash

# Path Detection for Devstack Skill
CONFIG_FILE="agent-skills/infrastructure/skills/devstack/config.json"
FOUND_PATH=""

echo "🔍 Detecting helmfile directory..."

# Step 1: Read config if exists
if [ -f "$CONFIG_FILE" ]; then
  HELMFILE_DIR=$(cat "$CONFIG_FILE" | grep -o '"helmfile_directory"[^,}]*' | cut -d'"' -f4)
  AUTO_DETECT=$(cat "$CONFIG_FILE" | grep -o '"auto_detect"[^,}]*' | awk '{print $2}')

  # Step 2: Try primary path
  if [ -n "$HELMFILE_DIR" ] && [ -d "$HELMFILE_DIR" ] && [ -f "$HELMFILE_DIR/helmfile.yaml" ]; then
    FOUND_PATH="$HELMFILE_DIR"
    echo "✅ Found helmfile at: $FOUND_PATH"
  else
    echo "⚠️ Primary path not found: $HELMFILE_DIR"

    # Step 3: Try fallbacks if auto_detect enabled
    if [ "$AUTO_DETECT" = "true" ]; then
      echo "🔍 Auto-detection enabled, trying fallback paths..."

      FALLBACK_PATHS=(
        "kube-manifests/helmfile"
        "helmfile"
        "../kube-manifests/helmfile"
      )

      for path in "${FALLBACK_PATHS[@]}"; do
        if [ -d "$path" ] && [ -f "$path/helmfile.yaml" ]; then
          FOUND_PATH="$path"
          echo "✅ Found helmfile at: $FOUND_PATH"
          break
        fi
      done
    fi
  fi
else
  echo "⚠️ Config file not found, using auto-detection..."

  # Default fallback paths
  FALLBACK_PATHS=(
    "kube-manifests/helmfile"
    "helmfile"
    "../kube-manifests/helmfile"
  )

  for path in "${FALLBACK_PATHS[@]}"; do
    if [ -d "$path" ] && [ -f "$path/helmfile.yaml" ]; then
      FOUND_PATH="$path"
      echo "✅ Found helmfile at: $FOUND_PATH"
      break
    fi
  done
fi

# Step 4: Handle result
if [ -n "$FOUND_PATH" ]; then
  echo "📂 Changing to helmfile directory: $FOUND_PATH"
  cd "$FOUND_PATH" || exit 1
  echo ""
  echo "Ready to proceed with helmfile operations."
else
  # Auto-clone kube-manifests if not found
  echo ""
  echo "⚠️ Helmfile directory not found locally. Auto-cloning kube-manifests..."
  CLONE_SUCCESS=false

  # Try gh CLI first (handles auth seamlessly)
  if command -v gh &> /dev/null && gh auth status &> /dev/null; then
    echo "🔍 Using gh CLI to clone..."
    gh repo clone razorpay/kube-manifests -- --depth 1 --single-branch && CLONE_SUCCESS=true
  fi

  # Fallback to git CLI
  if [ "$CLONE_SUCCESS" = false ] && command -v git &> /dev/null; then
    echo "🔍 Using git to clone..."
    git clone --depth 1 --single-branch git@github.com:razorpay/kube-manifests.git && CLONE_SUCCESS=true
  fi

  if [ "$CLONE_SUCCESS" = true ] && [ -d "kube-manifests/helmfile" ] && [ -f "kube-manifests/helmfile/helmfile.yaml" ]; then
    FOUND_PATH="kube-manifests/helmfile"
    echo "✅ Successfully cloned. Using helmfile at: $FOUND_PATH"
    cd "$FOUND_PATH" || exit 1
  else
    echo "❌ Unable to clone kube-manifests repository."
    echo "   Please clone it manually: git clone --depth 1 --single-branch git@github.com:razorpay/kube-manifests.git"
    exit 1
  fi
fi
```

## Best Practices for Path Detection

### 1. Always Report the Path
Users need to know which path is being used to debug issues.

### 2. Try Primary First
Respect the user's explicit configuration before trying fallbacks.

### 3. Be Verbose During Auto-Detection
Show which paths are being tried so users understand what's happening.

### 4. Auto-Clone Before Failing
If no path found, attempt to auto-clone the repo (gh CLI → git). Only fail with instructions if cloning also fails.

### 5. Validate Path Contents
Don't just check if directory exists - verify `helmfile.yaml` is present.

### 6. Use Absolute Paths When Possible
Resolve relative paths to absolute to avoid confusion about working directory.

## Common Issues and Solutions

### Issue: Path works for one user but not another

**Cause**: Absolute paths in config.json are user-specific

**Detection**:
```bash
if [[ "$HELMFILE_DIR" == /Users/* ]] || [[ "$HELMFILE_DIR" == /home/* ]]; then
  echo "⚠️ Warning: Using user-specific absolute path"
  echo "   Consider using relative paths for team portability"
fi
```

### Issue: Path changes after git pull

**Cause**: Config not committed or merge conflict

**Detection**:
```bash
if [ ! -f "$CONFIG_FILE" ]; then
  echo "⚠️ Warning: config.json not found"
  echo "   This file should be committed to version control"
fi
```

### Issue: Symlinks breaking path

**Cause**: Path contains symlinked directories

**Solution**:
```bash
# Resolve symlinks
RESOLVED_PATH=$(cd "$HELMFILE_DIR" && pwd -P)
echo "📂 Resolved path: $RESOLVED_PATH"
```

## Testing Path Detection

### Test Case 1: Valid Primary Path
```bash
# Setup
echo '{"helmfile_directory": "/valid/path/helmfile", "auto_detect": true}' > config.json
mkdir -p /valid/path/helmfile
touch /valid/path/helmfile/helmfile.yaml

# Expected: Uses primary path
```

### Test Case 2: Invalid Primary, Valid Fallback
```bash
# Setup
echo '{"helmfile_directory": "/invalid/path", "auto_detect": true}' > config.json
mkdir -p kube-manifests/helmfile
touch kube-manifests/helmfile/helmfile.yaml

# Expected: Uses kube-manifests/helmfile
```

### Test Case 3: No Valid Path
```bash
# Setup
echo '{"helmfile_directory": "/invalid/path", "auto_detect": true}' > config.json

# Expected: Error message with instructions
```

### Test Case 4: Auto-Detect Disabled
```bash
# Setup
echo '{"helmfile_directory": "/invalid/path", "auto_detect": false}' > config.json

# Expected: Error immediately, no fallback attempts
```

## Integration with Other Subskills

All subskills that need helmfile access should:

1. **Start with path detection**: Run the detection workflow before any helmfile operations
2. **Use the detected path**: Store in variable and reuse throughout operation
3. **Report the path**: Show user which path is being used
4. **Handle errors**: Auto-clone if path not found; stop only if cloning also fails

Example workflow:
```
Deployment Subskill
├─ Run path detection
├─ cd to detected path
├─ Validate service configuration
├─ Execute helmfile sync
└─ Monitor deployment
```

---

**Version**: 1.0.0
**Last Updated**: 2026-01-22
