# Production 0.1.4984-p1

[Upstream v0.1.4984](https://github.com/yetone/magpie/releases/tag/v0.1.4984) pins `ee3fe2a683422c135cceb97b3667b19c0c856437`. This round integrates 246 upstream commits since v0.1.4844. Production image source is `ced88b9d5567db1cd7580cee677aac5eb0a03377`.

[Upstream issue #8028](https://github.com/yetone/magpie/issues/8028) and [fork PR #4](https://github.com/skygunner/magpie/pull/4) track Raccoon GLM Flash group routing. Chat and Responses requests with JSON schema output skip that member before affinity. JSON object and direct requests retain their existing behavior.

The regression passed on the prior integration and the pinned upgrade; disabling the filter failed the candidate-selection assertion. Reviewed the upstream delta in the retained source paths. The GUI routing reference merge preserves both upstream slow-frame input handling and the fork's interrupted pixel-wheel behavior. Existing runtime patch deltas are retained.

Targeted gateway, update and GUI Go tests, gateway race checks, Linux headless vet, six additional short-test package suites and 83 deployment transaction tests passed. Chromium and WebKit covered clicks and wheel recovery, request details, routing navigation and slow-frame scrolling (tests 64, pass 64, fail 0). An isolated docker1 container passed its health and binary version checks.

Production probes:

```
group/smart-split json_schema 200 verified structured output
served by cline nvidia/nemotron-3-ultra-550b-a55b:free
raccoon/sn-glm-5-3-flash-p json_object 200 verified structured output
served by raccoon sn-glm-5-3-flash-p
```

The production transaction returned PASSED after 2760.952 seconds, including final API snapshot and health verification. Docker Hub manifest bytes for the version and latest tags match the running image (`sha256:c59db60ac91c5a07a49c57322739ca5cdd1d07577580fd982dfc37e4dfb1cf6b`). The public repository overview was updated and verified.
