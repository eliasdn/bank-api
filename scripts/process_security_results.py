#!/usr/bin/env python3
import json
import os
import re
import shutil
import subprocess
import sys

SEVERITY_ORDER = {
    "HIGH": 0,
    "MEDIUM": 1,
    "LOW": 2,
    "WARNING": 3,
    "INFORMATIONAL": 4
}

def parse_gosec(gosec_file):
    if not os.path.exists(gosec_file):
        print(f"Gosec file {gosec_file} not found.")
        return {}

    with open(gosec_file, 'r') as f:
        try:
            data = json.load(f)
        except Exception as e:
            print(f"Error reading gosec file: {e}")
            return {}

    issues = data.get("Issues", [])
    grouped = {}

    for issue in issues:
        rule_id = issue.get("rule_id", "UNKNOWN")
        details = issue.get("details", "Security issue detected")
        severity = issue.get("severity", "MEDIUM").upper()
        confidence = issue.get("confidence", "MEDIUM")
        file_path = issue.get("file", "")
        # Normalize relative path if needed
        file_path = file_path.replace("/app/", "")
        line = issue.get("line", "")
        code = issue.get("code", "").strip()

        group_key = f"GOSEC-{rule_id}"
        if group_key not in grouped:
            grouped[group_key] = {
                "id": group_key,
                "type": "gosec",
                "rule_id": rule_id,
                "title": f"Gosec {rule_id}: {details}",
                "severity": severity,
                "confidence": confidence,
                "details": details,
                "occurrences": []
            }

        grouped[group_key]["occurrences"].append({
            "file": file_path,
            "line": line,
            "code": code
        })

    return grouped

def parse_govulncheck(govulncheck_file):
    if not os.path.exists(govulncheck_file):
        print(f"Govulncheck file {govulncheck_file} not found.")
        return {}

    grouped = {}
    with open(govulncheck_file, 'r') as f:
        content = f.read()

    lines = content.splitlines()
    vuln_entries = []

    for line in lines:
        if not line.strip():
            continue
        try:
            entry = json.loads(line)
            if "finding" in entry:
                finding = entry["finding"]
                osv_id = finding.get("osv", "UNKNOWN")
                trace = finding.get("trace", [])
                pkg = trace[0].get("module", "unknown") if trace else "unknown"
                vuln_entries.append({
                    "id": osv_id,
                    "pkg": pkg,
                    "details": f"Vulnerability {osv_id} found in package {pkg}"
                })
        except json.JSONDecodeError:
            pass

    if not vuln_entries and content.strip():
        # Do not extract vulnerability IDs if govulncheck indicates 0 vulnerabilities affect the code
        if "Your code is affected by 0 vulnerabilities" not in content and "No vulnerabilities found." not in content:
            matches = re.findall(r'Vulnerability\s+#\d+:\s+(GO-\d{4}-\d+)', content) or re.findall(r'(GO-\d{4}-\d+)', content)
            for osv_id in set(matches):
                vuln_entries.append({
                    "id": osv_id,
                    "pkg": "dependencies",
                    "details": f"Govulncheck detected vulnerability {osv_id}"
                })

    for item in vuln_entries:
        group_key = f"GOVULN-{item['id']}"
        if group_key not in grouped:
            grouped[group_key] = {
                "id": group_key,
                "type": "govulncheck",
                "rule_id": item['id'],
                "title": f"Govulncheck {item['id']}: Vulnerability in {item['pkg']}",
                "severity": "HIGH",
                "confidence": "HIGH",
                "details": item['details'],
                "occurrences": [{"file": "go.mod / dependencies", "line": "-", "code": item['pkg']}]
            }

    return grouped

MAX_JULES_TASKS = int(os.environ.get("MAX_JULES_TASKS", 3))

def get_default_branch():
    code, out, _ = run_cmd(["git", "symbolic-ref", "--short", "refs/remotes/origin/HEAD"])
    if code == 0 and out:
        return out.replace("origin/", "")
    for b in ["master", "main"]:
        code, _, _ = run_cmd(["git", "rev-parse", "--verify", b])
        if code == 0:
            return b
    return "master"

def get_active_jules_count(existing_issues):
    count = 0
    for iss in existing_issues:
        for lbl in iss.get("labels", []):
            name = lbl.get("name", "") if isinstance(lbl, dict) else str(lbl)
            if name.lower() == "jules":
                count += 1
                break
    return count

def run_cmd(cmd_args):
    if not shutil.which(cmd_args[0]):
        return -1, "", f"Command {cmd_args[0]} not found in PATH"
    try:
        result = subprocess.run(cmd_args, capture_output=True, text=True, timeout=30)
        return result.returncode, result.stdout.strip(), result.stderr.strip()
    except Exception as e:
        return -1, "", str(e)

def get_existing_issues():
    code, out, _ = run_cmd(["gh", "issue", "list", "--state", "open", "--json", "title,number,labels"])
    if code != 0 or not out:
        return []
    try:
        return json.loads(out)
    except Exception:
        return []

def get_existing_prs():
    code, out, _ = run_cmd(["gh", "pr", "list", "--state", "open", "--json", "title,headRefName"])
    if code != 0 or not out:
        return []
    try:
        return json.loads(out)
    except Exception:
        return []

def ensure_label_exists(label_name, color="d73a4a", description=""):
    cmd = ["gh", "label", "create", label_name, "--color", color, "--force"]
    if description:
        cmd.extend(["--description", description])
    run_cmd(cmd)

def create_security_issue(group, existing_issues, jules_tracker):
    rule_id = group["rule_id"]
    group_id = group["id"]
    severity = group["severity"]
    title = f"Security [{severity}]: {group['title']}"

    # Check existing issues
    issue_exists = any(group_id in iss.get("title", "") or group["title"] in iss.get("title", "") for iss in existing_issues)

    # Determine if this issue should be assigned to Jules
    assign_to_jules = False
    if not issue_exists and jules_tracker["count"] < MAX_JULES_TASKS:
        # Prioritize assigning High and Medium severity to Jules
        if severity in ["HIGH", "CRITICAL", "MEDIUM"]:
            assign_to_jules = True
            jules_tracker["count"] += 1

    labels = ["security", f"{severity.lower()}-priority"]
    if assign_to_jules:
        labels.append("jules")

    if issue_exists:
        print(f"Issue already exists for {group_id}, skipping issue creation.")
    else:
        # Construct issue body
        body_lines = [
            f"## Security Finding: {group['title']}",
            f"**Severity:** {severity}",
            f"**Confidence:** {group.get('confidence', 'N/A')}",
            f"**Type:** {group['type']}",
            "",
            "### Description",
            group['details'],
            "",
            "### Occurrences",
            "| File | Line | Snippet |",
            "| --- | --- | --- |"
        ]

        for occ in group["occurrences"]:
            code_snippet = f"`{occ['code']}`" if occ['code'] else "N/A"
            body_lines.append(f"| `{occ['file']}` | {occ['line']} | {code_snippet} |")

        if assign_to_jules:
            body_lines.extend([
                "",
                "---",
                "🤖 Assigned to @google-labs-jules[bot] with the `jules` label for autonomous remediation."
            ])
        else:
            body_lines.extend([
                "",
                "---",
                "💡 *To assign this task to Jules, add the `jules` label to this issue.*"
            ])

        body = "\n".join(body_lines)
        cmd = ["gh", "issue", "create", "--title", title, "--body", body, "--label", ",".join(labels)]
        print(f"Creating issue for {group_id} (Jules: {assign_to_jules})...")
        code, out, err = run_cmd(cmd)
        if code == 0:
            print(f"Issue created successfully: {out}")
        else:
            print(f"Failed to create issue for {group_id}: {err}")



def main():
    gosec_file = sys.argv[1] if len(sys.argv) > 1 else "gosec-results.json"
    govulncheck_file = sys.argv[2] if len(sys.argv) > 2 else "govulncheck-results.json"

    gosec_groups = parse_gosec(gosec_file)
    govulncheck_groups = parse_govulncheck(govulncheck_file)

    all_groups = {**gosec_groups, **govulncheck_groups}

    # Sort groups by severity (HIGH > MEDIUM > LOW > WARNING)
    sorted_groups = sorted(
        all_groups.values(),
        key=lambda g: SEVERITY_ORDER.get(g.get("severity", "MEDIUM").upper(), 99)
    )

    # Ensure standard labels exist once
    ensure_label_exists("security", "d73a4a", "Security finding")
    ensure_label_exists("high-priority", "b60205", "High priority finding")
    ensure_label_exists("medium-priority", "fbca04", "Medium priority finding")
    ensure_label_exists("low-priority", "0e8a16", "Low priority finding")
    ensure_label_exists("jules", "1d76db", "Autonomous remediation by Google Labs Jules")

    # Fetch existing issues once
    existing_issues = get_existing_issues()
    active_jules_count = get_active_jules_count(existing_issues)
    jules_tracker = {"count": active_jules_count}

    print(f"Found {len(sorted_groups)} security finding groups to process.")
    print(f"Active Jules tasks: {active_jules_count}/{MAX_JULES_TASKS}.")

    for group in sorted_groups:
        print(f"Processing group: {group['id']} - Severity: {group['severity']} - Occurrences: {len(group['occurrences'])}")
        create_security_issue(group, existing_issues, jules_tracker)


if __name__ == "__main__":
    main()

