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
        matches = re.findall(r'(GO-\d{4}-\d+)', content)
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

def ensure_label_exists(label_name, color="d73a4a"):
    code, _, _ = run_cmd(["gh", "label", "create", label_name, "--color", color])

def create_issue_and_pr(group, base_commit):
    rule_id = group["rule_id"]
    group_id = group["id"]
    severity = group["severity"]
    title = f"Security [{severity}]: {group['title']}"

    # Ensure labels exist so gh issue create does not fail
    ensure_label_exists("security", "d73a4a")
    ensure_label_exists(f"{severity.lower()}-priority", "b60205" if severity == "HIGH" else "fbca04")

    # Check existing issues
    existing_issues = get_existing_issues()
    issue_exists = any(group_id in iss.get("title", "") or group["title"] in iss.get("title", "") for iss in existing_issues)

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

        body_lines.extend([
            "",
            "---",
            "Attention: @jules please investigate and resolve this security issue."
        ])

        body = "\n".join(body_lines)
        cmd = ["gh", "issue", "create", "--title", title, "--body", body, "--label", f"security,{severity.lower()}-priority"]
        print(f"Creating issue for {group_id}...")
        code, out, err = run_cmd(cmd)
        if code == 0:
            print(f"Issue created successfully: {out}")
        else:
            print(f"Failed to create issue for {group_id}: {err}")

    # Check existing PRs
    slug = group_id.lower().replace("-", "_")
    branch_name = f"security/fix-{slug}"
    existing_prs = get_existing_prs()
    pr_exists = any(branch_name in pr.get("headRefName", "") or group_id in pr.get("title", "") for pr in existing_prs)

    if pr_exists:
        print(f"PR/Branch already exists for {branch_name}, skipping PR creation.")
    else:
        # Create branch & PR from the base_commit (ensuring independent branches)
        print(f"Creating dedicated branch and PR for {group_id} from {base_commit}...")
        run_cmd(["git", "config", "user.name", "github-actions[bot]"])
        run_cmd(["git", "config", "user.email", "github-actions[bot]@users.noreply.github.com"])

        # Reset to base_commit before creating new branch
        run_cmd(["git", "checkout", "-f", base_commit])
        run_cmd(["git", "checkout", "-b", branch_name])
        run_cmd(["git", "commit", "--allow-empty", "-m", f"security: setup tracking branch for {group_id}"])
        code, out, err = run_cmd(["git", "push", "origin", branch_name])

        if code == 0:
            pr_title = f"Fix {group_id}: {group['title']}"
            pr_body = (
                f"Automated PR created for security alert **{group_id}**.\n\n"
                f"**Severity:** {severity}\n"
                f"**Details:** {group['details']}\n\n"
                f"Tagging @jules to inspect and commit the fix for this issue."
            )
            pr_cmd = ["gh", "pr", "create", "--title", pr_title, "--body", pr_body, "--base", "main", "--head", branch_name]
            code_pr, out_pr, err_pr = run_cmd(pr_cmd)
            if code_pr == 0:
                print(f"PR created successfully: {out_pr}")
            else:
                print(f"Failed to create PR for {group_id}: {err_pr}")
        else:
            print(f"Failed to push branch {branch_name}: {err}")

        # Switch back to base_commit
        run_cmd(["git", "checkout", "-f", base_commit])

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

    # Capture original base commit hash or commit ref to prevent branch chaining
    _, base_commit, _ = run_cmd(["git", "rev-parse", "HEAD"])
    if not base_commit:
        base_commit = os.environ.get("GITHUB_SHA", "main")

    print(f"Found {len(sorted_groups)} security finding groups to process.")
    for group in sorted_groups:
        print(f"Processing group: {group['id']} - Severity: {group['severity']} - Occurrences: {len(group['occurrences'])}")
        create_issue_and_pr(group, base_commit)

if __name__ == "__main__":
    main()
