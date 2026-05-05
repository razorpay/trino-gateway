#!/usr/bin/env python3
"""
Render baseline alerting YAML from Jinja2 templates and an inputs config.

Usage:
    python scripts/render.py --inputs path/to/inputs.yaml --output path/to/output.yaml
"""

import argparse
import copy
import re
import sys
from pathlib import Path

import jinja2
import yaml


# ---------------------------------------------------------------------------
# Defaults for optional fields -- merged under user-supplied values
# ---------------------------------------------------------------------------
DEFAULTS = {
    "application": {
        "language": "",
        "interface": {
            "rest": False,
            "grpc": False,
        },
        "utils": {
            "hasOutboxWorkers": False,
        },
    },
    "infra": {
        "traefik": {
            "services": [],
        },
        "edgeService": "",
        "kafka": {
            "topics": [],
        },
        "aws": {
            "rds": [],
            "elasticCache": [],
            "sqs": [],
            "asg": [],
        },
        "kubernetes": {
            "clusters": [],
        },
    },
    "metrics": {
        "statusCake": {
            "testId": "",
        },
        "anomalyOffsets": [],
    },
}

# Fields that must be present and non-empty
REQUIRED_FIELDS = [
    ("team", "bu"),
    ("team", "pod"),
    ("team", "slack", "channel"),
    ("team", "slack", "handle"),
    ("team", "runbook"),
    ("team", "dashboardLink"),
    ("cmd", "appName"),
    ("metrics", "prefix"),
    ("infra", "kubernetes", "namespace"),
    ("infra", "kubernetes", "containers"),
]


# ---------------------------------------------------------------------------
# Helpers
# ---------------------------------------------------------------------------

def deep_merge(base: dict, override: dict) -> dict:
    """Recursively merge *override* into a copy of *base*.

    Values in *override* take precedence. Dicts are merged recursively;
    all other types are replaced outright.
    """
    result = copy.deepcopy(base)
    for key, value in override.items():
        if key in result and isinstance(result[key], dict) and isinstance(value, dict):
            result[key] = deep_merge(result[key], value)
        else:
            result[key] = copy.deepcopy(value)
    return result


def get_nested(data: dict, *keys):
    """Traverse *data* by *keys*, returning None if any key is missing."""
    current = data
    for k in keys:
        if not isinstance(current, dict) or k not in current:
            return None
        current = current[k]
    return current


def validate_required(data: dict) -> list[str]:
    """Return a list of human-readable paths for every missing required field."""
    missing = []
    for keys in REQUIRED_FIELDS:
        value = get_nested(data, *keys)
        # Treat None, empty string, and empty list as missing
        if value is None or value == "" or value == []:
            missing.append(".".join(keys))
    return missing


def add_view_panel(dashboard_link: str, panel_id: str, *args: str) -> str:
    """Build a Grafana panel link.

    2 args  -> {dashboard_link}&viewPanel={panel_id}
    4 args  -> ...&var-{args[0]}={args[1]}
    """
    url = f"{dashboard_link}&viewPanel={panel_id}"
    if len(args) == 2:
        url += f"&var-{args[0]}={args[1]}"
    return url


# ---------------------------------------------------------------------------
# Core pipeline
# ---------------------------------------------------------------------------

def load_inputs(path: Path) -> dict:
    """Load and return the inputs YAML file."""
    if not path.exists():
        print(f"Error: input file not found: {path}", file=sys.stderr)
        sys.exit(1)

    with open(path, "r") as fh:
        try:
            data = yaml.safe_load(fh)
        except yaml.YAMLError as exc:
            print(f"Error: failed to parse input YAML: {exc}", file=sys.stderr)
            sys.exit(1)

    if not isinstance(data, dict):
        print("Error: input YAML must be a mapping at the top level", file=sys.stderr)
        sys.exit(1)

    return data


def apply_defaults(data: dict) -> dict:
    """Deep-merge DEFAULTS under *data* so optional keys always exist."""
    return deep_merge(DEFAULTS, data)


def build_jinja_env(template_dir: Path) -> jinja2.Environment:
    """Create a Jinja2 environment with custom globals/filters."""
    if not template_dir.is_dir():
        print(f"Error: template directory not found: {template_dir}", file=sys.stderr)
        sys.exit(1)

    env = jinja2.Environment(
        loader=jinja2.FileSystemLoader(str(template_dir)),
        undefined=jinja2.StrictUndefined,
        trim_blocks=True,
        lstrip_blocks=True,
    )

    # Register custom helpers
    env.globals["add_view_panel"] = add_view_panel

    return env


def render(env: jinja2.Environment, data: dict) -> str:
    """Render the master alerts template and return the string output."""
    try:
        template = env.get_template("alerts.yaml.j2")
    except jinja2.TemplateNotFound as exc:
        print(f"Error: master template not found: {exc}", file=sys.stderr)
        sys.exit(1)

    try:
        return template.render(**data)
    except jinja2.TemplateError as exc:
        print(f"Error: template rendering failed: {exc}", file=sys.stderr)
        sys.exit(1)


def post_process(rendered: str) -> str:
    """Clean up whitespace artefacts left by Jinja2 block trimming.

    Jinja2's {%- / -%} operators aggressively strip whitespace and newlines.
    When one module's closing -%} eats its trailing newline and the next
    module's {%- eats the preceding newline, the last content line of one
    group merges with the # comment of the next.

    Symptoms:
      vajra_link: https://...&viewPanel=9001#        (stray # glued to URL)
      vajra_link: https://...  - name: Foo           (- name: on same line)

    This function:
    1. Splits lines where a # comment or - name: block got merged onto the
       end of a content line.
    2. Ensures every - name: block is preceded by a blank line.
    3. Strips trailing whitespace from each line.
    4. Ensures the file ends with exactly one newline.
    """
    lines = [line.rstrip() for line in rendered.splitlines()]

    # --- Pass 1: split merged content ---
    split: list[str] = []
    for line in lines:
        # Case A: "  - name:" embedded after other content on the same line
        m = re.match(r'^(.+?)(  - name:.*)$', line)
        if m and not line.lstrip().startswith("- name:"):
            split.append(m.group(1).rstrip())
            split.append("")
            split.append(m.group(2))
            continue

        # Case B: a stray "#" (with optional trailing text) glued directly
        # to the end of a deep-indented YAML value line.  The merged "#" is
        # the first character of the next module's comment block.
        #
        # Pattern: 6+ spaces of indent, then YAML content, ending with "#"
        # possibly followed by comment text.  We only fire when the content
        # before "#" is NOT itself a comment line and the "#" is not part of
        # a quoted string value.
        m2 = re.match(r'^( {6,}\S.*?)(#(.*))$', line)
        if m2:
            content = m2.group(1).rstrip()
            comment_text = m2.group(2)  # e.g. "#" or "# Node level alerts"
            # Avoid splitting legitimate YAML -- only split when content
            # does not start with # and the trailing # is not inside quotes.
            if not content.lstrip().startswith("#"):
                # Check the # is not inside a quoted string value
                # Simple heuristic: if there's an odd number of " before
                # the #, it's inside a string -- skip.
                pre = m2.group(1)
                if pre.count('"') % 2 == 0:
                    split.append(content)
                    split.append("")
                    split.append("  " + comment_text)
                    continue

        split.append(line)

    # --- Pass 2: ensure blank line before every "  - name:" block ---
    result: list[str] = []
    for line in split:
        if re.match(r'^  - name:', line) and result and result[-1] != "":
            result.append("")
        result.append(line)

    text = "\n".join(result)
    # Collapse runs of 3+ blank lines down to one blank line
    text = re.sub(r"\n{3,}", "\n\n", text)
    # Ensure exactly one trailing newline
    text = text.rstrip("\n") + "\n"
    return text


def validate_output(rendered: str) -> None:
    """Ensure the rendered string is valid YAML."""
    try:
        yaml.safe_load(rendered)
    except yaml.YAMLError as exc:
        print(f"Error: rendered output is not valid YAML:\n{exc}", file=sys.stderr)
        sys.exit(1)


def write_output(path: Path, rendered: str) -> None:
    """Write rendered YAML to *path*, creating parent dirs if needed."""
    path.parent.mkdir(parents=True, exist_ok=True)
    path.write_text(rendered)


# ---------------------------------------------------------------------------
# CLI
# ---------------------------------------------------------------------------

def main() -> None:
    parser = argparse.ArgumentParser(
        description="Render baseline alerting YAML from Jinja2 templates.",
    )
    parser.add_argument(
        "--inputs", required=True, type=Path,
        help="Path to the service inputs YAML file",
    )
    parser.add_argument(
        "--output", required=True, type=Path,
        help="Path to write the rendered alerts YAML",
    )
    args = parser.parse_args()

    # 1. Load inputs
    data = load_inputs(args.inputs)

    # 2. Apply defaults for optional fields
    data = apply_defaults(data)

    # 3. Validate required fields
    missing = validate_required(data)
    if missing:
        print(
            "Error: the following required fields are missing or empty:\n  - "
            + "\n  - ".join(missing),
            file=sys.stderr,
        )
        sys.exit(1)

    # 4. Load templates
    template_dir = Path(__file__).resolve().parent.parent / "templates" / "jinja2"
    env = build_jinja_env(template_dir)

    # 5. Render
    rendered = render(env, data)

    # 6. Post-process: fix whitespace artefacts from Jinja2 block trimming
    rendered = post_process(rendered)

    # 7. Validate output YAML
    validate_output(rendered)

    # 8. Write
    write_output(args.output, rendered)
    print(f"Rendered alerts written to {args.output}")


if __name__ == "__main__":
    main()
