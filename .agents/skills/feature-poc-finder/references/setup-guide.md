# Source Setup Guide

Reference for configuring the data sources used by this skill. Load this file when a user reports that a source is unavailable and needs setup instructions.

---

## GitHub CLI

**Check status:**
```bash
gh auth status
```

**Install:**
```bash
# macOS
brew install gh

# Ubuntu / Debian
sudo apt install gh
```

**Authenticate:**
```bash
gh auth login
```

Follow the interactive prompts — select `GitHub.com`, `HTTPS`, and authenticate via browser or token.

---

## DevRev MCP

**Check:** Call `devrev → get_self`. If it errors, MCP is not connected.

**Connect:**

Run `/rzp-discover:connect-mcps` inside Claude Code. This sets up the MCP connection automatically.

**Manual setup (if the above doesn't work):**

1. Generate a Personal Access Token:
   - Go to **DevRev → Settings → Account → Personal Access Token**
   - Create a token with the required scopes

2. Add an entry to your project's `.claude/mcp.json`:
   ```json
   {
     "mcpServers": {
       "devrev": {
         "type": "http",
         "url": "https://api.devrev.ai/mcp/v1",
         "headers": {
           "Authorization": "Bearer <YOUR_PAT_TOKEN>"
         }
       }
     }
   }
   ```

3. Restart Claude Code to pick up the new config.

---

## Slack MCP

**Check:** Call `slack_search_public_and_private` with `query: test` and `limit: 1`. If it errors, MCP is not connected.

**Connect:**

Run `/rzp-discover:connect-mcps` inside Claude Code.

> **Note (Razorpay-specific):** The Slack MCP app requires IT approval before it can be activated. If `/rzp-discover:connect-mcps` fails or the session stays unauthenticated, raise a request in the **#tech_it** Slack channel asking for the Slack MCP app to be approved for your account.

Once approved, run `/rzp-discover:connect-mcps` again to complete the connection.
