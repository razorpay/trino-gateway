#!/bin/bash
# Validates agent-ready plugin structure and outputs C1, C2, C3 scores.
# Usage: bash validate-context.sh <repo-dir>

TARGET="${1:-.}"
PASS=0
FAIL=0
WARN=0

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
BOLD='\033[1m'
RESET='\033[0m'

ok()   { echo -e "  ${GREEN}✓${RESET}  $1"; PASS=$((PASS+1)); }
fail() { echo -e "  ${RED}✗${RESET}  $1"; FAIL=$((FAIL+1)); }
warn() { echo -e "  ${YELLOW}?${RESET}  $1"; WARN=$((WARN+1)); }

check_file()    { [ -f "$TARGET/$1" ]; }
check_dir()     { [ -d "$TARGET/$1" ]; }
check_symlink() { [ -L "$TARGET/$1" ]; }
check_symlink_target() {
    local link="$TARGET/$1"
    local expected="$2"
    [ -L "$link" ] && readlink "$link" | grep -q "$expected"
}

REPO_NAME=$(basename "$(cd "$TARGET" && git rev-parse --show-toplevel 2>/dev/null || pwd)")
last_commit_date() { git -C "$TARGET" log --follow -1 --format="%ct" -- "$1" 2>/dev/null; }

echo ""
echo -e "${BOLD}═══════════════════════════════════════════${RESET}"
echo -e "${BOLD}  AGENT-READY VALIDATION: $REPO_NAME${RESET}"
echo -e "${BOLD}═══════════════════════════════════════════${RESET}"

# ── Directory Structure ──────────────────────────────────────────────────────
echo ""; echo -e "${BOLD}  Directory Structure${RESET}"

for d in ".agents/skills/repo-skill" ".agents/skills/repo-skill/core" \
          ".agents/skills/repo-skill/modules/domain" \
          ".agents/skills/repo-skill/modules/technical" \
          ".agents/skills/repo-skill/modules/integration" \
          ".claude" ".claude/rules"; do
    check_dir "$d" && ok "$d/" || fail "$d/"
done

check_file ".claude/settings.json" && ok ".claude/settings.json" || fail ".claude/settings.json"
check_file ".claude/.gitignore"    && ok ".claude/.gitignore"    || fail ".claude/.gitignore"

# ── Agentfill / Symlinks ─────────────────────────────────────────────────────
echo ""; echo -e "${BOLD}  Agentfill / Symlinks${RESET}"

check_dir ".agents/polyfills/agentsmd" \
    && ok ".agents/polyfills/agentsmd/ (hook scripts)" \
    || fail ".agents/polyfills/agentsmd/ — agentfill not installed"

if check_symlink ".claude/skills"; then
    if check_symlink_target ".claude/skills" ".agents/skills"; then
        ok ".claude/skills → ../.agents/skills (symlink correct)"
    else
        ok ".claude/skills symlink present (target: $(readlink "$TARGET/.claude/skills"))"
    fi
elif check_dir ".claude/skills"; then
    ok ".claude/skills directory present"
elif check_dir ".agents/skills" && check_dir ".agents/polyfills/agentsmd"; then
    warn ".claude/skills symlink missing — .agents/skills/ + agentfill present (add symlink: ln -s ../.agents/skills .claude/skills)"
else
    fail ".claude/skills missing"
fi

[ -d "$TARGET/.cursor" ] && { check_symlink ".cursor/skills" && ok ".cursor/skills symlink" || warn ".cursor/skills symlink missing"; }
[ -d "$TARGET/.gemini" ] && { check_symlink ".gemini/skills" && ok ".gemini/skills symlink" || warn ".gemini/skills symlink missing"; }

# ── Extracted Knowledge ──────────────────────────────────────────────────────
echo ""; echo -e "${BOLD}  Extracted Knowledge${RESET}"

check_file ".agents/skills/repo-skill/core/boundaries.md"   && ok "core/boundaries.md"   || fail "core/boundaries.md"
check_file ".agents/skills/repo-skill/core/quick-ref.md"    && ok "core/quick-ref.md"    || fail "core/quick-ref.md"

DOMAIN_COUNT=$(find "$TARGET/.agents/skills/repo-skill/modules/domain" -name "*.md" 2>/dev/null | wc -l | tr -d ' ')
[ "$DOMAIN_COUNT" -gt 0 ] && ok "domain/ — $DOMAIN_COUNT entity file(s)" || fail "domain/ — no entity files extracted"

check_file ".agents/skills/repo-skill/modules/technical-patterns.md"         && ok "technical-patterns.md"  || fail "technical-patterns.md"
check_file ".agents/skills/repo-skill/modules/integration/service-contracts.md" && ok "service-contracts.md" || fail "service-contracts.md"
check_file ".agents/skills/repo-skill/modules/integration/external-deps.md"     && ok "external-deps.md"     || fail "external-deps.md"

# ── AGENTS.md ────────────────────────────────────────────────────────────────
echo ""; echo -e "${BOLD}  AGENTS.md${RESET}"
if check_file "AGENTS.md"; then
    ok "AGENTS.md"
else
    fail "AGENTS.md missing"
fi

NESTED_AGENTS=$(find "$TARGET" -name "AGENTS.md" -not -path "$TARGET/AGENTS.md" \
    -not -path "*/.git/*" -not -path "*/.agents/*" 2>/dev/null | wc -l | tr -d ' ')
[ "$NESTED_AGENTS" -gt 0 ] && ok "Nested AGENTS.md — $NESTED_AGENTS file(s)" || warn "No nested AGENTS.md files"

# ── Skill Manifest + Version ─────────────────────────────────────────────────
echo ""; echo -e "${BOLD}  Skill Manifest & Version${RESET}"
check_file ".agents/skills/repo-skill/SKILL.md"  && ok "repo-skill/SKILL.md"          || fail "repo-skill/SKILL.md missing"

# ── Naming Convention ────────────────────────────────────────────────────────
echo ""; echo -e "${BOLD}  Naming Convention${RESET}"
if check_file "CLAUDE.md" || check_symlink "CLAUDE.md"; then
    ok "CLAUDE.md (uppercase)"
elif check_file "claude.md"; then
    fail "claude.md must be renamed to CLAUDE.md"
elif check_file "AGENTS.md" && check_dir ".agents/polyfills/agentsmd"; then
    warn "CLAUDE.md missing — AGENTS.md present with agentfill polyfill (context injected at runtime)"
else
    fail "CLAUDE.md missing"
fi

# ── C2: Codebase Navigability ────────────────────────────────────────────────
echo ""; echo -e "${BOLD}  Codebase Navigability — Large Subdirectory Coverage${RESET}"
LARGE_TOTAL=0; LARGE_COVERED=0

while IFS= read -r -d '' subdir; do
    relpath="${subdir#$TARGET/}"
    dirname=$(basename "$subdir")

    # Skip dirs that are clearly not domain logic
    echo "$relpath" | grep -qE '(mock|mocks|gen|generated|proto|pb|grpc|testdata|test/dummy|migrations?|\.github)' && continue

    src_count=$(find "$subdir" -maxdepth 1 -type f \
        \( -name "*.go" -o -name "*.py" -o -name "*.ts" -o -name "*.tsx" \
           -o -name "*.js" -o -name "*.jsx" -o -name "*.java" \
           -o -name "*.rb" -o -name "*.rs" -o -name "*.php" \) 2>/dev/null | wc -l | tr -d ' ')

    [ "$src_count" -lt 15 ] && continue

    # Skip if majority of files are configs, not source
    total_count=$(find "$subdir" -maxdepth 1 -type f 2>/dev/null | wc -l | tr -d ' ')
    [ "$total_count" -gt 0 ] && src_pct=$(( src_count * 100 / total_count )) || src_pct=0
    [ "$src_pct" -lt 50 ] && continue

    LARGE_TOTAL=$((LARGE_TOTAL+1))
    has_agents=false; has_claude=false
    [ -f "$subdir/AGENTS.md" ] && has_agents=true
    [ -f "$subdir/CLAUDE.md" ] && has_claude=true

    if $has_agents || $has_claude; then
        LARGE_COVERED=$((LARGE_COVERED+1))
        signal=""
        $has_agents && signal="AGENTS.md"
        $has_claude && signal="${signal:+$signal, }CLAUDE.md"
        ok "$relpath/ ($src_count src files) — $signal"
    else
        warn "$relpath/ ($src_count src files) — no AGENTS.md or CLAUDE.md"
    fi
done < <(find "$TARGET" -mindepth 2 -maxdepth 7 -type d \
    -not -path "*/.git/*" -not -path "*/.agents/*" \
    -not -path "*/vendor/*" -not -path "*/node_modules/*" \
    -not -path "*/.claude/*" -print0 2>/dev/null)

if [ "$LARGE_TOTAL" -eq 0 ]; then
    ok "No qualifying subdirectories (no source-heavy dirs with ≥15 files)"
    echo "Navigability Coverage: 100%"
else
    COVERAGE_PCT=$(( (LARGE_COVERED * 100) / LARGE_TOTAL ))
    [ "$COVERAGE_PCT" -ge 75 ] && ok "Navigability: $LARGE_COVERED/$LARGE_TOTAL dirs covered ($COVERAGE_PCT%)" \
        || { [ "$COVERAGE_PCT" -ge 40 ] && warn "Navigability: $LARGE_COVERED/$LARGE_TOTAL dirs covered ($COVERAGE_PCT%)" \
             || fail "Navigability: $LARGE_COVERED/$LARGE_TOTAL dirs covered ($COVERAGE_PCT%)"; }
    echo "Navigability Coverage: ${COVERAGE_PCT}%"
fi

# ── C3: Context Freshness ────────────────────────────────────────────────────
echo ""; echo -e "${BOLD}  Context Freshness${RESET}"
FRESHNESS_SCORE=0
SKILL_DIR="$TARGET/.agents/skills/repo-skill"
STALE_COUNT=0
NO_FRONTMATTER_COUNT=0
TOTAL_DOCS=0

echo -e "  ${BOLD}Signal 1 & 2: Doc staleness via source frontmatter${RESET}"

# Walk all .md docs in repo-skill (excluding SKILL.md itself)
while IFS= read -r doc; do
    rel="${doc#$TARGET/}"
    [ "$(basename "$doc")" = "SKILL.md" ] && continue
    TOTAL_DOCS=$((TOTAL_DOCS+1))

    # Extract sources: frontmatter list under "sources:" key
    sources=$(awk '/^---/{f=!f;next} f && /^sources:/{p=1;next} p && /^  - /{print substr($0,5);next} p && /^[^ ]/{p=0}' "$doc" 2>/dev/null)
    extracted_at=$(awk '/^---/{f=!f;next} f && /^extracted_at:/{print $2;exit}' "$doc" 2>/dev/null)

    if [ -z "$sources" ] || [ -z "$extracted_at" ]; then
        NO_FRONTMATTER_COUNT=$((NO_FRONTMATTER_COUNT+1))
        warn "NO FRONTMATTER: $rel — sources/extracted_at missing (re-extract with agent-ready plugin)"
        continue
    fi

    # Convert extracted_at (YYYY-MM-DD) to epoch
    doc_ts=$(date -j -f "%Y-%m-%d" "$extracted_at" "+%s" 2>/dev/null \
          || date -d "$extracted_at" "+%s" 2>/dev/null || echo 0)

    # Check if any declared source file has commits after extracted_at
    stale=false
    stale_detail=""
    while IFS= read -r src_file; do
        [ -z "$src_file" ] && continue
        last_change=$(git -C "$TARGET" log -1 --format="%ct" -- "$src_file" 2>/dev/null)
        [ -z "$last_change" ] && continue
        if [ "$last_change" -gt "$doc_ts" ]; then
            stale=true
            stale_detail="$src_file changed $(git -C "$TARGET" log -1 --format="%cr" -- "$src_file" 2>/dev/null)"
            break
        fi
    done <<< "$sources"

    if $stale; then
        STALE_COUNT=$((STALE_COUNT+1))
        warn "STALE: $rel — $stale_detail"
    else
        ok "Fresh: $rel"
    fi
done < <(find "$SKILL_DIR" -name "*.md" 2>/dev/null)

if [ "$TOTAL_DOCS" -eq 0 ]; then
    warn "C3: no docs found in repo-skill"
elif [ "$NO_FRONTMATTER_COUNT" -eq "$TOTAL_DOCS" ]; then
    warn "C3: no docs have source frontmatter — agent-ready plugin needs updating (see Specs/agentic_sdlc_scoring/AGENT_READY_FRESHNESS_SPEC.md)"
    FRESHNESS_SCORE=0
elif [ "$STALE_COUNT" -eq 0 ]; then
    ok "Staleness: all $TOTAL_DOCS docs current"
    FRESHNESS_SCORE=$((FRESHNESS_SCORE+4))
elif [ "$STALE_COUNT" -le 2 ]; then
    warn "Staleness: $STALE_COUNT doc(s) stale"
    FRESHNESS_SCORE=$((FRESHNESS_SCORE+2))
else
    fail "Staleness: $STALE_COUNT docs stale"
    FRESHNESS_SCORE=$((FRESHNESS_SCORE+0))
fi

# ── Summary ──────────────────────────────────────────────────────────────────
echo ""
echo -e "  ${BOLD}Freshness Score: $FRESHNESS_SCORE/4${RESET}"
echo ""
echo -e "${BOLD}═══════════════════════════════════════════${RESET}"
TOTAL=$((PASS+FAIL+WARN))
echo -e "${BOLD}  Result: ${GREEN}$PASS passed${RESET}  ${RED}$FAIL failed${RESET}  ${YELLOW}$WARN warnings${RESET}  (of $TOTAL checks)"
echo -e "${BOLD}  Freshness Score: $FRESHNESS_SCORE/4${RESET}"
echo -e "${BOLD}═══════════════════════════════════════════${RESET}"
echo ""

[ "$FAIL" -gt 0 ] && exit 1 || exit 0
