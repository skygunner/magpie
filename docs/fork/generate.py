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

def gitrc(*args):
    """git's exit status: 0 when the assertion holds (is-ancestor & co)."""
    return subprocess.run(["git"] + list(args), capture_output=True, text=True).returncode

PROTECTED_BRANCHES = {"main", "master", "dev", "origin"}

def branch_inventory():
    """Live refs, with the JSON's classifications merged back in.

    Symrefs are detected by %(symref) pointing at another ref, not by
    objecttype. Protected names (main, dev, origin) never classify as
    fix/unknown. fix_ids, category and notes come from the JSON's branch
    records where it has them, so the markdown keeps the ledger's
    attributions; heads and existence come from git.
    """
    out, seen = [], set()
    for line in git("for-each-ref", "refs/remotes/origin",
                    "--format=%(refname)|%(objectname)|%(symref)").splitlines():
        full, sha, symref = line.split("|")
        name = full[len("refs/remotes/origin/"):]
        if symref:
            seen.add(name)
            out.append({"branch": name, "head": "symref->" + symref.split("origin/")[-1],
                        "kind": "symref", "where": "remote", "category": "n/a",
                        "fix_ids": [], "base": "",
                        "note": "symbolic ref, not a branch; excluded from head counts"})
            continue
        if name == "HEAD":
            continue
        seen.add(name)
        out.append(branch_entry(name, sha, "remote"))
    for line in git("for-each-ref", "refs/heads",
                    "--format=%(refname:short)|%(objectname)").splitlines():
        name, sha = line.split("|")
        if name not in seen:
            out.append(branch_entry(name, sha, "local-only"))
    return out

def branch_entry(name, sha, where):
    if name in ("main", "master"):
        kind, cat = "protected", "stale-protected"
    elif name == "dev":
        kind, cat = "maintenance", "maintenance"
    elif name == "origin":
        kind, cat = "fix", "stray"
    else:
        kind, cat = "fix", "unknown"
    base = git("merge-base", sha, "upstream/main")[:40] if kind == "fix" else ""
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
        if b.get("archive", {}).get("candidate"):
            if b.get("head") in (None, "", "0" * 40):
                errs.append(f"branch {b['branch']}: candidate with a null head")
            if str(b.get("branch")) in PROTECTED_BRANCHES:
                errs.append(f"branch {b['branch']}: a protected branch is never a "
                            f"deletion candidate, whatever its units")
            # the candidate head must be what the branch actually is: a
            # real but different commit means the record drifted and the
            # candidacy is unproven
            live = git("rev-parse", "--verify", "--quiet",
                       "origin/" + b["branch"] + "^{commit}") or \
                  git("rev-parse", "--verify", "--quiet",
                      "refs/heads/" + b["branch"] + "^{commit}")
            if not live or live != b.get("head"):
                errs.append(f"branch {b['branch']}: candidate head {str(b.get('head'))[:8]} "
                            f"is not the branch's actual ref ({live[:8] if live else 'unresolvable'}) "
                            f"— candidacy unproven")
            for uid in b.get("fix_ids", []):
                u = next((x for x in units if x["id"] == uid), None)
                if u is None:
                    errs.append(f"branch {b['branch']}: candidate cites unknown unit {uid}")
                elif not u.get("archive", {}).get("candidate"):
                    errs.append(f"branch {b['branch']}: candidate while its unit {uid} is not")
    # every code unit must carry its delivery comments; pointed-elsewhere
    # comments must belong to their own issue and the verified evidence set
    for u in units:
        if u.get("category") in ("product", "tests") and not u.get("comments"):
            errs.append(f"{u['id']}: a code unit without its delivery comments")
    for pnt in data.get("pointed_elsewhere", []):
        for i in pnt["issues"]:
            for url in pnt.get("comments", []):
                if f"/issues/{i}" not in url and f"/pull/{i}" not in url:
                    errs.append(f"pointed entry #{i}: comment url names another topic: {url}")
                elif str(i) in prior and url not in prior[str(i)].get("urls", []):
                    errs.append(f"pointed entry #{i}: comment not in the verified evidence set: {url}")
    if data.get("archive_policy", {}).get("deletions_this_round") != 0:
        errs.append("deletions_this_round must be 0 in a read-only round")
    # totals are computed, not believed
    want = compute_totals(data)
    got = data.get("meta", {}).get("totals")
    if got and any(got.get(k) != v for k, v in want.items() if k in got):
        diffs = {k: (got.get(k), v) for k, v in want.items() if k in got and got.get(k) != v}
        errs.append(f"stored totals disagree with the ledger's own data: {diffs}")
    if got and "unknown_entries" in want:
        errs.append(f"the ledger carries entries outside the required 72: {want['unknown_entries']}")
    # candidates come from one graph: the stored list, the per-unit flags
    # and the per-branch flags must agree with what the relations imply
    ap = data.get("archive_policy", {})
    listed = ap.get("candidates", [])
    for c in listed:
        if str(c) in PROTECTED_BRANCHES:
            errs.append(f"archive candidate list names a protected branch: {c}")
    unit_candidates = {u["id"] for u in units if u.get("archive", {}).get("candidate")}
    branch_candidate_ids = set()
    for b in data.get("branches", []):
        if b.get("archive", {}).get("candidate"):
            if not b.get("fix_ids"):
                errs.append(f"branch {b['branch']}: candidate without attributed fix units")
            else:
                branch_candidate_ids.update(b["fix_ids"])
    for uid in unit_candidates:
        u = next(x for x in units if x["id"] == uid)
        up = u.get("upstream", {})
        if str(up.get("status", "")).startswith("adopted") and not up.get("commit"):
            errs.append(f"candidate unit {uid}: adopted upstream without its commit recorded")
        if str(up.get("status", "")).startswith("adopted"):
            c = up.get("commit", "")
            if not SHA_RE.match(str(c)) or git("cat-file", "-t", c) != "commit":
                errs.append(f"candidate unit {uid}: adoption commit {c!r} is not a full, "
                            f"resolvable commit SHA")
            if not up.get("release"):
                errs.append(f"candidate unit {uid}: an adopted product fix records no source "
                            f"release for its release-verifiable scope")
    if unit_candidates != set(listed):
        errs.append(f"archive_policy.candidates {sorted(listed)} != the units flagged candidate {sorted(unit_candidates)}")
    derived = {u["id"] for u in units
               if u.get("archive", {}).get("candidate")
               and u.get("verify", {}).get("state") in ("pass", "pass_upstream")
               and u.get("upstream", {}).get("status") in (
                   "adopted_with_changes", "adopted_verbatim", "patch_equivalent",
                   "independent_alternative", "superseded")}
    if not unit_candidates <= derived:
        errs.append(f"candidate units whose verify/upstream do not support candidacy: {sorted(unit_candidates - derived)}")
    for uid in unit_candidates - branch_candidate_ids:
        errs.append(f"candidate unit {uid} has no candidate branch carrying it")
    # a declared chain is an order: each commit must have the previous as
    # an ancestor, and the last must be (or contain) the delivery head
    for dname, drec in data.get("deliveries", {}).items():
        if drec.get("kind") == "maintenance":
            continue
        head, commits = drec.get("head", ""), drec.get("commits", [])
        if head:
            if not SHA_RE.match(str(head)) or git("cat-file", "-t", head) != "commit":
                errs.append(f"delivery {dname}: head is not a resolvable full SHA")
            if head not in {b["head"] for b in branches if b["kind"] != "symref"} and dname != "dev":
                errs.append(f"delivery {dname}: head is neither a live remote nor local head")
        prev = None
        for c in commits:
            if not SHA_RE.match(str(c)) or git("cat-file", "-t", c) != "commit":
                errs.append(f"delivery {dname}: chain commit {str(c)[:8]} is not a resolvable full SHA")
                prev = None
                continue
            if prev and gitrc("merge-base", "--is-ancestor", prev, c) != 0:
                errs.append(f"delivery {dname}: {prev[:8]} is not an ancestor of {c[:8]} — chain order is wrong")
            prev = c
        if commits and head and gitrc("merge-base", "--is-ancestor", commits[-1], head) != 0:
            errs.append(f"delivery {dname}: last chain commit {commits[-1][:8]} is not an ancestor of the head")
        # completeness: a delivery promising the current sequence must carry
        # every non-merge commit of base..head — a truncated list passes the
        # ancestry checks while dropping the newest fixes
        if commits and head:
            base = drec.get("base") or ""
            if SHA_RE.match(str(base)) and git("cat-file", "-t", base) == "commit":
                want = [c for c in git("rev-list", "--reverse", "--no-merges",
                                       f"{base}..{head}").splitlines() if c]
                missing = [c for c in want if c not in commits]
                if missing:
                    errs.append(f"delivery {dname}: commits omit {len(missing)} commit(s) of "
                                f"base..head, first {missing[0][:8]}")
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

    u0 = next(u for u in data["units"] if u.get("category") in ("product", "tests"))
    # the nine review negatives, each against an otherwise-good copy
    v = copy.deepcopy(data)
    v["units"][0]["comments"] = {}
    variants["code unit's comments deleted"] = v

    v = copy.deepcopy(data)
    cand = data["archive_policy"]["candidates"][0] if data["archive_policy"].get("candidates") else None
    if cand:
        bu = next(u for u in v["units"] if u["id"] == cand)
        bb = next(b for b in v["branches"] if b["branch"] == bu.get("branch") or
                  b.get("fix_ids") and cand in b["fix_ids"])
        bb["archive"] = {"candidate": True}
        bb["head"] = "0" * 40
        variants["candidate branch head is null sha"] = v

    v = copy.deepcopy(data)
    if cand:
        bu = next(u for u in v["units"] if u["id"] == cand)
        bu["upstream"].pop("commit", None)
        bu["upstream"].pop("release", None)
        variants["adopted candidate lost its upstream commit/release"] = v

    v = copy.deepcopy(data)
    v["archive_policy"]["candidates"] = ["main"]
    variants["candidates list names main"] = v

    v = copy.deepcopy(data)
    for b in v["branches"]:
        if b.get("fix_ids") and b["branch"] != "r3-batch":
            b["archive"] = {"candidate": True}
            break
    variants["branch candidate while its units are not"] = v

    v = copy.deepcopy(data)
    v["branches"] = []
    variants["empty branch map"] = v

    v = copy.deepcopy(data)
    v["meta"]["totals"]["entries"] = 999
    variants["totals = 999"] = v

    v = copy.deepcopy(data)
    drec = v["deliveries"]["r3-union"]
    drec["commits"] = list(reversed(drec["commits"]))
    variants["union chain reversed"] = v

    v = copy.deepcopy(data)
    for pnt in v["pointed_elsewhere"]:
        if pnt.get("comments"):
            pnt["comments"] = [u.replace(f"/issues/{pnt['issues'][0]}", "/issues/3076")
                               for u in pnt["comments"]]
            break
    variants["pointed comment names another topic"] = v

    # the four r4-review archive negatives, one changed fact each
    v = copy.deepcopy(data)
    vu = next(u for u in v["units"] if u["id"] == "u-volcengine-resets")
    vu["upstream"].pop("release", None)
    variants["adopted candidate keeps commit but loses its release"] = v

    v = copy.deepcopy(data)
    vb = next(b for b in v["branches"] if b["branch"] == "fix/volcengine-resets")
    vb["archive"] = {"candidate": True}
    vb["head"] = "275b7a2ea558fcc25c24b71ec077bfe000e83f04"
    variants["candidate branch head swapped for another real commit"] = v

    v = copy.deepcopy(data)
    vu = next(u for u in v["units"] if u["id"] == "u-volcengine-resets")
    vu["upstream"]["commit"] = "not-a-sha"
    variants["adoption commit is not a sha"] = v

    v = copy.deepcopy(data)
    mb = next(b for b in v["branches"] if b["branch"] == "main")
    mb["archive"] = {"candidate": True}
    mb["fix_ids"] = ["u-volcengine-resets"]
    variants["main itself flagged candidate with a qualified unit"] = v

    v = copy.deepcopy(data)
    v["deliveries"]["r3-batch"]["commits"] = v["deliveries"]["r3-batch"]["commits"][:-3]
    variants["delivery chain truncated before the newest fixes"] = v

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
    remote = [b for b in branches if is_live_remote_head(b)]
    w(f"{len(remote)} remote heads (this run's snapshot), "
      f"{sum(1 for b in branches if b.get('where')=='local-only')} local-only, "
      f"{sum(1 for b in branches if b['kind']=='symref')} symref(s) excluded.\n")
    for b in branches:
        if b["kind"] == "symref":
            continue
        if b.get("where") == "local-only":
            w(f"- `{b['branch']}` local-only (historical snapshot, unavailable on this machine) "
              f"head `{b['head'][:8]}`"
              + (f" → fix_ids: {', '.join(b['fix_ids'])}" if b.get("fix_ids") else ""))
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

def normalize_branch_record(rec):
    """One record contract for every consumer: where says machine-local
    vs remote. A stored record that omits where is judged by its kind —
    a local-only record is another machine's history, never a live
    remote."""
    if not rec.get("where"):
        rec["where"] = "local-only" if rec.get("kind") == "local-only" else "remote"
    rec.setdefault("kind", "fix")
    rec.setdefault("fix_ids", [])
    rec.setdefault("category", "unknown")
    if rec["where"] == "local-only":
        rec.setdefault("note", "")
        if "unavailable" not in rec["note"]:
            rec["note"] = (rec["note"] + "; " if rec["note"] else "") + \
                "machine-local ref, unavailable here (historical snapshot)"
    return rec

def is_live_remote_head(b):
    """A head this run actually sees: a remote, non-symref ref."""
    return b.get("where") == "remote" and b.get("kind") != "symref"

def merge_branch_records(branches, data):
    """Carry the JSON's branch attributions onto the live rows, and flag
    drift: a JSON head that no longer matches git is kept as
    head_in_ledger with a note, never silently overwritten."""
    by_name = {b["branch"]: b for b in branches}
    for rec in data.get("branches", []):
        normalize_branch_record(rec)
        b = by_name.get(rec["branch"])
        if b is None:
            where = rec["where"]
            if where == "remote":
                # a remote branch the ledger classifies but git no longer
                # has: the record may be stale — said, not hidden
                rec.setdefault("note", "")
                rec["note"] = (rec["note"] + "; " if rec["note"] else "") + \
                    "not on the remote at this run (stale record?)"
                branches.append(rec)
            else:
                rec.setdefault("note", "")
                rec["note"] = (rec["note"] + "; " if rec["note"] else "") + \
                    "machine-local ref, unavailable here (historical snapshot)"
                branches.append(rec)
            continue
        for k in ("fix_ids", "category", "note", "kind"):
            if rec.get(k) and (b.get(k) in (None, "", [], "unknown", "n/a") or k == "fix_ids"):
                b[k] = rec[k]
        if b["head"] != rec.get("head") and not str(rec.get("head", "")).startswith("symref"):
            b["head_in_ledger"] = rec.get("head")
            b["note"] = (b.get("note", "") + "; " if b.get("note") else "") + \
                f"ledger head was {str(rec.get('head'))[:8]}"
    return branches

def compute_totals(data):
    units = data["units"]
    covered = {}
    for u in units:
        for i in u["issues"]:
            covered.setdefault(i, True)
    for pnt in data.get("pointed_elsewhere", []):
        for i in pnt["issues"]:
            covered.setdefault(i, True)
    prs = {i for i in covered if str(i) in prior and prior[str(i)].get("type") == "PR"}
    cats = {}
    unknown = [n for n in covered if str(n) not in prior]
    for n in covered:
        if str(n) in prior:
            cats[prior[str(n)]["category"]] = cats.get(prior[str(n)]["category"], 0) + 1
    t = {"entries": len(covered), "issues": len(covered) - len(prs), "prs": len(prs),
         "product": cats.get("product", 0), "tests_ci": cats.get("tests", 0),
         "upstream_existing": cats.get("upstream_fixed", 0), "other": cats.get("other", 0),
         "code_entries": cats.get("product", 0) + cats.get("tests", 0),
         "units": len(units)}
    if unknown:
        t["unknown_entries"] = unknown
    return t

def main():
    global prior
    data = json.load(open(JSON_PATH))
    prior = json.load(open(PRIOR_PATH))
    branches = merge_branch_records(branch_inventory(), data)
    errs = check(data, branches, prior)
    if "--selftest" in sys.argv:
        errs += selftest(data, branches, prior)
    if errs:
        for e in errs:
            print("CHECK FAIL:", e)
        return 1
    units = data["units"]
    heads = [b for b in branches if is_live_remote_head(b)]
    print(f"checks ok: {len(units)} units, {sum(len(u['issues']) for u in units)} unit-covered issues, "
          f"{len(heads)} remote heads, {sum(1 for b in branches if b.get('where')=='local-only')} local-only")
    if "--write" in sys.argv:
        open(MD_PATH, "w").write(write_md(data, branches))
        print("wrote", MD_PATH)
    return 0

if __name__ == "__main__":
    sys.exit(main())
