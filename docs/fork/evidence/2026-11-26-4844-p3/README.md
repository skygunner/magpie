# Magpie 0.1.4844-p3

Built on docker1 from `5752c2bbc66a5ae217d3abb63212654686366447`, based on upstream v0.1.4844. This image adds [#7867](https://github.com/yetone/magpie/issues/7867) to the production integration containing [#7864](https://github.com/yetone/magpie/issues/7864).

The OpenAI wire regression reproduced the custom temperature on unpatched reasoning attempts. Targeted gateway, update and console checks, Linux gateway vet, targeted gateway race checks and 83 deployment transaction tests passed. A container using an isolated temporary configuration reported the expected version and passed its health check on docker1.

The additional browser run reported 174 passes and 6 failures (four narrow Usage wheel cases and two parent suites). Re-running the four cases against p2 assets reproduced the two Chromium failures; the two WebKit cases passed on that rerun. The GUI code and browser fixtures are identical in p2 and p3.

The fixed deployment entry verified the production configuration snapshot and real inference. Two additional Chat Completions probes used `temperature=0.1` with medium effort, directed separately to OpenAI GPT-6 Luna and GPT-6.1 Terra. Both returned HTTP 200 with assistant content.

The transaction completed at 2026-11-26T14:19:49.661244+00:00 after 2760.991 seconds of observation, with its final snapshot and health checks passing. The versioned image and latest tag were promoted from the accepted image. Public registry readback confirmed both resolve to `sha256:b5a9441b256a4d173dac398e29a7d69ab07f30b881a77dea07b212e847641903`, matching the immutable manifest running on docker1. Docker Hub returned the updated hotfix overview on readback.
