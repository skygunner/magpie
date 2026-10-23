#!/usr/bin/env python3
"""Fork ledger: validate upstream-status.json and regenerate UPSTREAM_STATUS.md.

Strict by default: exits non-zero on any of
  - duplicate unit ids or missing required fields
  - a commit that is not a full 40-hex SHA that resolves to a commit object
  - a branch or an issue referenced that the ledger does not itself carry
  - the 72-entry set not matching the required collection (55 product, 9
    test/CI, 6 pointed at upstream's own fixes, 2 other; #1072/#1073/#1258
    are PRs)
  - a comment permalink whose issue number does not match its key, or that
    prior-comments.json (the verified evidence set) does not carry
  - an archive candidate that is not verified+upstream-adopted (or
    superseded), or a mixed/unknown/superseded branch marked as candidate
  - a partial unit whose coverage names no remaining scope

Branch inventory (remote heads, local-only, symrefs) comes from git at run
time, so counts are never hand-written.

    python3 docs/fork/generate.py                 # validate only
    python3 docs/fork/generate.py --write         # validate, then rewrite the md
    python3 docs/fork/generate.py --selftest      # validate, then prove five
                                                  #   classes of bad data are
                                                  #   rejected (read-only)
"""
import json, os, re, subprocess, sys

HERE = os.path.dirname(os.path.abspath(__file__))
JSON_PATH = os.path.join(HERE, "upstream-status.json")
MD_PATH = os.path.join(HERE, "UPSTREAM_STATUS.md")
PRIOR_PATH = os.path.join(HERE, "prior-comments.json")

REQUIRED_SET = {n: "product" for n in
    (1072,1073,1258,2008,2259,2287,2323,2389,2460,2473,2530,2923,2953,3020,3027,
     3035,3062,3076,3097,3121,3139,3141,3148,3150,3152,3153,3165,3173,3176,3183,
     3193,3197,3205,3213,3224,3233,3234,3251,3258,3259,3260,3266,3276,3282,3285,
     3289,3302,3306,3339,3349,3352,3358,3365,3370,3374)}
REQUIRED_SET.update({n: "tests" for n in (2207,3095,3100,3174,3229,3277,3280,3284,3311)})
REQUIRED_SET.update({n: "upstream_fixed" for n in (2949,3143,3144,3151,3255,3345)})
REQUIRED_SET.update({n: "other" for n in (3131,3147)})
PRS = {1072, 1073, 1258}
SHA_RE = re.compile(r"^[0-9a-f]{40}$")
COMMENT_RE = re.compile(r"#issuecomment-([0-9]+)$")

def git(*args):
    return subprocess.run(["git"] + list(args), capture_output=True, text=True).stdout.strip()

def branch_inventory():
    out, seen = [], set()
    for line in git("for-each-ref", "refs/remotes/origin",
                    "--format=%(objectname)|%(refname:short)|%(objecttype)").splitlines():
        sha, full, typ = line.split("|")
        name = full[len("origin/"):]
        if typ == "symref":
            seen.add(name)
            out.append({"branch": name, "head": "symref", "kind": "symref", "where": "remote",
                        "category": "n/a", "fix_ids": [], "base": "",
                        "note": "symbolic ref, not a branch; excluded from head counts"})
            continue
        if name == "HEAD":
            continue
        seen.add(name)
        out.append(branch_entry(name, sha, "remote"))
    for line in git("for-each-ref", "refs/heads",
                    "--format=%(objectname)|%(refname:short)").splitlines():
        sha, name = line.split("|")
        if name not in seen:
            out.append(branch_entry(name, sha, "local-only"))
    return out

def branch_entry(name, sha, where):
    kind = "maintenance" if name == "dev" else "fix"
    base = ""
    if kind == "fix":
        base = git("merge-base", sha, "upstream/main")[:40]
    cat = "maintenance" if kind == "maintenance" else "unknown"
    return {"branch": name, "head": sha, "kind": kind, "where": where,
            "base": base, "fix_ids": [], "category": cat, "note": ""}

def check(data, branches, prior):
    errs = []
    units = data.get("units")
    if not isinstance(units, list) or not units:
        return ["no units"]
    ids = [u.get("id") for u in units]
    if any(not i for i in ids) or len(ids) != len(set(ids)):
        errs.append("duplicate or missing unit ids")
    prior_rows = prior
    sha_ok = {}
    for u in units:
        uid = u.get("id", "?")
        for f in ("issues", "commits", "comments", "verify", "upstream", "coverage"):
            if f not in u:
                errs.append(f"{uid}: missing field {f}")
        if not u.get("commits"):
            errs.append(f"{uid}: no commits recorded")
        for c in u.get("commits", []):
            if not SHA_RE.match(str(c)):
                errs.append(f"{uid}: commit {c!r} is not a full 40-hex SHA")
                continue
            if c not in sha_ok:
                sha_ok[c] = git("cat-file", "-t", c) == "commit"
            if not sha_ok[c]:
                errs.append(f"{uid}: commit {c[:8]} does not resolve to a commit object")
        if u.get("verify", {}).get("state") == "rework_needed" and u.get("archive", {}).get("candidate"):
            errs.append(f"{uid}: rework needed but marked an archive candidate")
        if u.get("upstream", {}).get("status") == "partial" and \
           not re.search(r"remain|not implemented|未实现", u.get("coverage", ""), re.I):
            errs.append(f"{uid}: upstream partial but coverage names no remaining scope")
        if u.get("archive", {}).get("candidate"):
            st = u.get("verify", {}).get("state")
            up = u.get("upstream", {}).get("status")
            if st not in ("pass", "pass_upstream") or up not in (
                    "adopted_with_changes", "adopted_verbatim", "independent_alternative",
                    "patch_equivalent", "superseded"):
                errs.append(f"{uid}: archive candidate without verification and adoption "
                            f"(verify={st}, upstream={up})")
            if len(u.get("issues", [])) > 3:
                errs.append(f"{uid}: mixed multi-issue unit marked an archive candidate")
        for issue_key, urls in u.get("comments", {}).items():
            row = prior_rows.get(str(issue_key))
            if row is None:
                errs.append(f"{uid}: issue {issue_key} has no verified prior evidence row")
                continue
            for url in urls:
                m = COMMENT_RE.search(url)
                if not m:
                    errs.append(f"{uid}: comment url for {issue_key} has no comment id: {url}")
                elif url not in row.get("urls", []):
                    errs.append(f"{uid}: comment for {issue_key} not in the verified evidence set: {url}")
    covered = {}
    for u in units:
        for i in u["issues"]:
            covered.setdefault(i, []).append(u["id"])
    for p in data.get("pointed_elsewhere", []):
        for i in p["issues"]:
            covered.setdefault(i, []).append("pointed")
    missing = set(REQUIRED_SET) - set(covered)
    extra = set(covered) - set(REQUIRED_SET)
    if missing:
        errs.append(f"required entries missing from the ledger: {sorted(missing)}")
    if extra:
        errs.append(f"entries outside the required set: {sorted(extra)}")
    for n, cat in REQUIRED_SET.items():
        row = prior_rows.get(str(n))
        if row is None:
            errs.append(f"#{n}: no verified prior row")
        elif row.get("category") != cat:
            errs.append(f"#{n}: category {row.get('category')} != required {cat}")
        if n in PRS and prior_rows.get(str(n), {}).get("type") != "PR":
            errs.append(f"#{n}: must be a PR")
    names = {b["branch"] for b in branches}
    for u in units:
        b = u.get("branch", "")
        if b and b.split(" ")[0] not in names and b not in names:
            errs.append(f"{u['id']}: branch {b} does not exist on the fork or locally")
    # branch bookkeeping is checked on the JSON's own branch records (the
    # single source of truth); `branches` (live git) cross-checks existence
    for b in data.get("branches", []):
        if b.get("kind") == "symref":
            continue
        if b["branch"] not in names:
            errs.append(f"branch {b['branch']} is in the ledger but not on the fork or local refs")
        for uid in b.get("fix_ids", []):
            if uid not in ids:
                errs.append(f"branch {b['branch']}: fix_id {uid} is not a unit")
        if b.get("archive", {}).get("candidate") and b.get("category") in (
                "unknown", "batch-chain", "superseded"):
            errs.append(f"branch {b['branch']}: {b['category']} marked an archive candidate")
    if data.get("archive_policy", {}).get("deletions_this_round") != 0:
        errs.append("deletions_this_round must be 0 in a read-only round")
    return errs

def selftest(data, branches, prior):
    """Five classes of bad data must be rejected; the delivered data must pass."""
    import copy
    fails = []
    good = check(data, branches, prior)
    if good:
        return [f"the delivered data itself does not validate: {good}"]
    variants = {}

    v = copy.deepcopy(data)
    v["units"][0]["commits"] = []
    variants["required commits deleted"] = v

    v = copy.deepcopy(data)
    v["units"][0]["commits"] = ["not-a-sha"]
    variants["commit is not a sha"] = v

    v = copy.deepcopy(data)
    for u in v["units"]:
        if 3374 in u["issues"]:
            u["issues"] = [999999 if i == 3374 else i for i in u["issues"]]
            urls = u["comments"].pop("3374", None)
            if urls:
                u["comments"]["999999"] = urls
    variants["3374 swapped for 999999"] = v

    v = copy.deepcopy(data)
    for u in v["units"]:
        if u["verify"]["state"] not in ("pass", "pass_upstream") or \
           u["upstream"].get("status") in ("unconfirmed", "none_yet"):
            u["archive"]["candidate"] = True
            break
    variants["unverified/unadopted unit made a candidate"] = v

    v = copy.deepcopy(data)
    for b in v["branches"]:
        if b["category"] in ("unknown", "batch-chain", "superseded"):
            b["archive"] = {"candidate": True}
            break
    variants["mixed/unknown branch made a candidate"] = v

    for name, v in variants.items():
        errs = check(v, branches, prior)
        if not errs:
            fails.append(f"variant NOT rejected: {name}")
        else:
            print(f"  selftest ok: rejected {name!r} -> {errs[0]}")
    return fails

def write_md(data, branches):
    L = []
    w = L.append
    m = data["meta"]
    w(f"# Fork delivery status — {m['fork']} → {m['upstream']}\n")
    w(f"Checked {m['checked_at']} against upstream/main `{m['last_checked_upstream_sha'][:8]}`. "
      "Generated by `docs/fork/generate.py` from `upstream-status.json` — edit the JSON, not this file.\n")
    r3 = m.get("r3", {})
    if r3:
        w(f"\nRound-3 trees (base `{r3.get('base','')[:8]}`): batch `{r3.get('batch_head','')[:8]}`, "
          f"union `{r3.get('union_head','')[:8]}`, rests `{r3.get('rests_head','')[:8]}` — {r3.get('pushed','')}\n")
    t = m["totals"]
    w(f"\n{t['entries']} entries = {t['issues']} issues + {t['prs']} PRs "
      f"({t['product']} product, {t['tests_ci']} test/CI, {t['upstream_existing']} pointed at upstream's own fixes, "
      f"{t['other']} other). {t['code_entries']} have delivered code, grouped into {t['units']} fix units by first "
      "primary commit; code entries ≠ independent fixes.\n")
    w("\n## Fix units\n")
    for u in data["units"]:
        w(f"### {u['id']}\n")
        w(f"- Issues: {', '.join('#'+str(i) for i in u['issues'])} ({u['category']})")
        w(f"- Commits: {', '.join('`'+c[:8]+'`' for c in u['commits'])}"
          + (f" — branch `{u['branch']}`" if u.get('branch') else ""))
        if u.get("branch_v2"):
            w(f"- Round-3 tree: `{u['branch_v2']}` (head `{u.get('verify',{}).get('head','')[:8]}`)")
        w(f"- Coverage: {u['coverage']}")
        up = u["upstream"]
        w(f"- Upstream: {up.get('status','unconfirmed')}"
          + (f" at `{up['commit'][:8]}` ({up.get('release','')})" if up.get("commit") else ""))
        src = u.get("sources", {})
        if src.get("credited") or src.get("coauthors"):
            w(f"- Sources: credited {src.get('credited') or '—'}; coauthors {', '.join(src.get('coauthors', [])) or '—'}")
        w(f"- Verification: {u['verify']['state']}"
          + (f" at `{u['verify']['head'][:8]}`" if u['verify'].get('head') else "")
          + f" — {u['verify']['evidence']}")
        for i, urls in sorted(u["comments"].items(), key=lambda kv: int(kv[0])):
            w(f"- Comment on #{i}: {urls[0]}")
        a = u["archive"]
        if a.get("candidate"):
            w(f"- **Archive candidate** (earliest review {a.get('earliest_review','?')}): {a.get('reason','')}")
            for bl in a.get("blockers", []):
                w(f"  - open check: {bl}")
        w("")
    w("## Pointed elsewhere / no code\n")
    for p in data["pointed_elsewhere"]:
        w(f"- {', '.join('#'+str(i) for i in p['issues'])} ({p['category']}): {p['note']}"
          + (f" — comment {p['comments'][0]}" if p.get("comments") else ""))
    w("\n## Branches (from git at this run)\n")
    remote = [b for b in branches if b["kind"] in ("fix", "maintenance") and b["where"] == "remote"]
    w(f"{len(remote)} remote heads, "
      f"{sum(1 for b in branches if b.get('where')=='local-only')} local-only, "
      f"{sum(1 for b in branches if b['kind']=='symref')} symref(s) excluded.\n")
    for b in branches:
        if b["kind"] == "symref":
            continue
        w(f"- `{b['branch']}` {b['kind']}/{b['category']} head `{b['head'][:8]}`"
          + (f" base `{b.get('base','')[:8]}`" if b.get("base") else "")
          + (f" → fix_ids: {', '.join(b['fix_ids'])}" if b.get("fix_ids") else "")
          + (f" — {b['note']}" if b.get("note") else ""))
    ap = data["archive_policy"]
    w("\n## Archive\n")
    w(ap["candidate_rule"] + "\n")
    for c in ap["candidates"]:
        w(f"- Candidate now: {c}")
    for c in ap["never"]:
        w(f"- Never: {c}")
    w(f"- Remote deletions this round: {ap['deletions_this_round']}")
    return "\n".join(L) + "\n"

def main():
    data = json.load(open(JSON_PATH))
    prior = json.load(open(PRIOR_PATH))
    branches = branch_inventory()
    errs = check(data, branches, prior)
    if "--selftest" in sys.argv:
        errs += selftest(data, branches, prior)
    if errs:
        for e in errs:
            print("CHECK FAIL:", e)
        return 1
    units = data["units"]
    heads = [b for b in branches if b["kind"] in ("fix", "maintenance") and b["where"] == "remote"]
    print(f"checks ok: {len(units)} units, {sum(len(u['issues']) for u in units)} unit-covered issues, "
          f"{len(heads)} remote heads, {sum(1 for b in branches if b.get('where')=='local-only')} local-only")
    if "--write" in sys.argv:
        open(MD_PATH, "w").write(write_md(data, branches))
        print("wrote", MD_PATH)
    return 0

if __name__ == "__main__":
    sys.exit(main())
