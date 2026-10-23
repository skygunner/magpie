#!/usr/bin/env python3
"""Generate UPSTREAM_STATUS.md from upstream-status.json and check it.

Read-only by default: writes only the .md next to the .json, and only
with --write. Checks the ledger for the mistakes a hand-edited one makes:
duplicate unit ids, missing SHAs, an issue named by two units without a
shared-patch note, references to branches that do not exist, partial
units listed as complete, mixed batches marked archivable.

Branch inventory comes from git (origin remotes), so it is always current.

    python3 docs/fork/generate.py            # check only
    python3 docs/fork/generate.py --write    # also write the markdown
"""
import json, os, subprocess, sys

HERE = os.path.dirname(os.path.abspath(__file__))
JSON_PATH = os.path.join(HERE, "upstream-status.json")
MD_PATH = os.path.join(HERE, "UPSTREAM_STATUS.md")

def git(*args):
    return subprocess.run(["git"] + list(args), capture_output=True, text=True).stdout.strip()

def branch_inventory():
    out, seen = [], set()
    for line in git("for-each-ref", "refs/remotes/origin",
                    "--format=%(refname:short)|%(objectname)").splitlines():
        full, sha = line.split("|")
        name = full[len("origin/"):]
        seen.add(name)
        out.append(branch_entry(name, sha, "fix"))
    for line in git("for-each-ref", "refs/heads",
                    "--format=%(refname:short)|%(objectname)").splitlines():
        name, sha = line.split("|")
        if name not in seen:
            out.append(branch_entry(name, sha, "local-only"))
    return out

def branch_entry(name, sha, default_kind):
    kind = default_kind
    if name == "origin":
        kind = "mirror-of-upstream-main"
    elif name == "dev":
        kind = "maintenance"
    base = git("merge-base", sha, "upstream/main")[:8]
    tail = [c for c in git("rev-list", "--reverse", "--no-merges",
                           f"{base}..{sha}").splitlines() if c]
    return {"branch": name, "head": sha[:8], "kind": kind, "base": base, "commits": tail}

def check(data, branches):
    errs = []
    units = data.get("units", [])
    ids = [u["id"] for u in units]
    if len(ids) != len(set(ids)):
        errs.append("duplicate unit ids")
    seen_issues = {}
    for u in units:
        for c in u.get("commits", []) + u.get("commits_v2", []):
            for sha in c.replace("->", " ").split():
                if len(sha) == 40 and git("cat-file", "-t", sha) != "commit":
                    errs.append(f"{u['id']}: unknown commit {sha[:8]}")
        for i in u.get("issues", []):
            seen_issues.setdefault(i, []).append(u["id"])
        if u.get("upstream", {}).get("status") in ("partial",) and \
           "NOT implemented" not in u.get("coverage", "") and "remain" not in u.get("coverage", ""):
            errs.append(f"{u['id']}: partial but coverage names no remaining scope")
        if u.get("archive", {}).get("candidate") and len(u.get("issues", [])) > 3:
            errs.append(f"{u['id']}: mixed batch marked archivable")
        if u.get("verify", {}).get("state") == "rework_needed" and u.get("archive", {}).get("candidate"):
            errs.append(f"{u['id']}: rework needed but archivable")
    for i, us in seen_issues.items():
        if len(us) > 1:
            shared = [u for u in units if u["id"] in us
                      and ("shared" in u.get("coverage", "").lower()
                           or "layer" in u.get("coverage", "").lower()
                           or len(u.get("issues", [])) > 1)]
            if not shared:
                errs.append(f"issue {i} claimed by {us} with no shared-patch note")
    known = {b["branch"] for b in branches}
    text = json.dumps(data)
    for u in units:
        for key in ("branch", "branch_v2"):
            b = u.get(key)
            if b and not b.startswith(("fix/issue-batch-20261023*", "various", "chain")):
                base = b.split(" ")[0]
                if base not in known:
                    errs.append(f"{u['id']}: branch {base} not on the fork")
    # every issue in the 72 must appear somewhere
    n_units_issues = len(seen_issues) + len(data.get("pointed_elsewhere", []))
    if n_units_issues < 72:
        errs.append(f"only {n_units_issues} issues accounted for, want >= 72")
    return errs

def write_md(data, branches):
    L = []
    w = L.append
    m = data["meta"]
    w(f"# Fork delivery status — {data['meta']['fork']} → {data['meta']['upstream']}\n")
    w(f"Checked {m['checked_at']} against upstream/main `{m['last_checked_upstream_sha'][:8]}`. "
      "Generated from `upstream-status.json` by `docs/fork/generate.py` — edit the JSON, not this file.\n")
    w("## Fix units\n")
    for u in data["units"]:
        w(f"### {u['id']} — {u['title']}\n")
        w(f"- Issues: {', '.join('#'+i for i in u['issues'])} ({u['category']})")
        if u.get("branch"):
            w(f"- Branch: `{u['branch']}` base `{u.get('base','?')}` commits {' '.join('`'+c[:8]+'`' for c in u.get('commits', []))}")
        if u.get("branch_v2"):
            w(f"- Rebased tree: `{u['branch_v2']}` base `{u['base_v2'][:8]}` commits {' '.join('`'+c[:8]+'`' for c in u.get('commits_v2', []))}")
        w(f"- Coverage: {u.get('coverage','(none recorded)')}")
        ups = u.get("upstream", {})
        w(f"- Upstream: {ups.get('status','unconfirmed')}"
          + (f" at `{ups['commit'][:8]}` ({ups.get('release','')})" if ups.get("commit") else ""))
        w(f"- Verification: {u.get('verify',{}).get('state','unverified')} — {u.get('verify',{}).get('evidence','')}")
        arch = u.get("archive", {})
        if arch.get("candidate"):
            w(f"- **Archive candidate**: {arch.get('reason','')} earliest review {arch.get('earliest_review','?')}")
        w("")
    w("## Pointed elsewhere / no fix attempted\n")
    for p in data.get("pointed_elsewhere", []):
        w(f"- {', '.join('#'+i for i in p['issues'])}: {p['note']}")
    w("\n## Branch inventory (from git, this read)\n")
    w(f"{len(branches)} remote branches. `batch` chains carry many units; `single` carry one; "
      "`maintenance`/`mirror` never archive.\n")
    for b in branches:
        n = len(b["commits"])
        w(f"- `{b['branch']}` {b['kind']} head `{b['head']}` base `{b['base']}` {n} commit(s)")
    w("\n## Archive\n")
    ap = data["archive_policy"]
    w(ap["candidate_rule"] + "\n")
    for c in ap["candidates_now"]:
        w(f"- Candidate now: {c}")
    for c in ap["never"]:
        w(f"- Never: {c}")
    return "\n".join(L) + "\n"

def main():
    data = json.load(open(JSON_PATH))
    branches = branch_inventory()
    errs = check(data, branches)
    for e in errs:
        print("CHECK FAIL:", e)
    if errs:
        return 1
    print(f"checks ok: {len(data['units'])} units, {len(branches)} branches")
    if "--write" in sys.argv:
        open(MD_PATH, "w").write(write_md(data, branches))
        print("wrote", MD_PATH)
    return 0

if __name__ == "__main__":
    sys.exit(main())
