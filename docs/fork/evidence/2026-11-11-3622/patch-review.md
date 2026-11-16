# v0.1.3622 integration review

Pinned upstream: 025123431c140a4988efc3d5c44546e8d73d16d3 (v0.1.3622).
Previous accepted source: b7b5539de03005a071a57eceaa06fde3f7faecfa (0.1.3336-p1).
411 upstream commits. Candidate status is separate from production.

## Patch decisions

Retain #3035's explicit grammar-format downgrade; #3635's quoted parameter-list
repair and proven-equivalent reasoning aliases; #3636's narrow unsupported-list
group fallback; #3624's explicit Usage-to-Routing reveal; #3948's bounded
nonstream protocol diagnostics with committed HTTP status; #3794's full-tools
count-limit fallback; #3623's ordered attempt history, including 499; and #4385's
complete interrupted pixel-wheel step per view. Build-pN normalization remains.

Upstream converted package-level expressions to `lazyre.New` in this range; the
retained patterns were carried into that form (gateway, format, update) with no
behavior change. Upstream's Usage request page replaces the in-table expanding
row and moves the one Routing link into the page. #3624's reveal and #3623's
attempt history are carried into the page, and the row's time cell no longer
links to Routing (upstream's own change). #3948's error-source labels and
client-processing note follow the page. The attempt history reads the route's
recorded tries when the page opens and on the row's own route only; another
computer's row keeps the gateway's own limits.

#3487 remainder, #3788, #3791 and the layered cooldown editor/API stay
suspended: their competing quota/cooldown semantics still conflict with upstream.
#3417 and #3430 remain absorbed upstream. The Cloudflare Workers AI plugin
remains 0.0.5 separately.

## Issue follow-up

All eight retained issues remain open with no ready upstream fix.
#4385 is reproduced again on origin/main a6a99717e (yetone, 2026-11-09);
the fix remains ours. The other issues' discussions are unchanged since the
previous round. No duplicate issue comment or upstream PR is posted.

## Proportional validation

Freeze one candidate source, then run Go once, Linux nogui build/vet, one
selected race run, 80 deployment tests and the eight browser fixtures:
click-scroll, view-open-wheel, usage-ledger-detail, usage-routing-visible,
usage-speed-every-row, gateway-key-concurrency, model-limits and
sessions-context-line. Playwright covers both engines with the fixtures' own
language cases. No complete GUI matrix, unrelated fixture repair or
cross-platform desktop release testing is planned. Isolated production-config
replay precedes the production switch; actual results are recorded separately.
