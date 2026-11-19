@joei4cm 移植收到，我把我这边能补的一块验证补上了——你如实标注的「两个新测试的改前红是推演」：

在本机对 `fix/2878-credit-pools`（`d174c5e41`）反向应用 `9e84c1ac5` 的 `routing.go` 产品改动（保留四个测试）重跑，**四个测试改前全红**：

- `TestACreditsPoolUsedUpIsNoShareUsed`（provider）
- `TestAUsedUpDedicatedPackageIsNoShareUsed`（provider）
- `TestGroupTriesAQoderAccountOnItsAddOnCredits`（gateway）
- `TestGroupTriesAQoderAccountOnItsTotalCredits`（gateway）

恢复产品改动后四个测试全绿（`d174c5e41` 头上）；`go build -tags nogui ./...` 干净。改前红/改后绿两条腿现在都是实跑见证，与移植记录里的推演一致。

我原分支的两个测试在你 base 上的改前红见证仍以原帖为准。分支状态无需再动，等 @yetone 收取即可。
