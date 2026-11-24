@takomitkm thanks for the nudge. Two notes:

1. The refusal matcher is generic, not SenseNova-specific: any 400/422 whose text names a structured-output field (`response_format`, `json_schema`, `guided_grammar`, `compile_grammar`, …) counts as a refusal and falls into the existing retry cascade. So another vendor rejecting structured output with different 400 wording is covered by the same matcher without another patch.
2. The patch is carried on our fork's production integration line (`internal/gateway/format.go`, `formatRefusal` with `guided_grammar|compile_grammar`). The standalone branch gets rebased onto current upstream/main when the maintainer wants to look at it; upstream still takes no outside PRs, so for now it waits there.
