接着做了，分支 [`fix/r25-autostart-exec-stable`](https://github.com/skygunner/magpie/tree/fix/r25-autostart-exec-stable)（基于 main `19253f67`，提交 `0aac5452b2310d9a658b6787fdf6d57238de0102`）。

**根因**：`internal/autostart/autostart.go` 的 `self()` 用 `os.Executable()` 加 `filepath.EvalSymlinks` 取运行中的文件，把解析后的带版本号真实路径（`/nix/store/<hash>-magpie-<版本>/bin/magpie`、`.../Cellar/magpie/<版本>/bin/magpie` 这类）原样交给 `enable()` 写进 `~/.config/autostart/magpie.desktop` 的 `Exec`。升级回收旧版本后这个文件没了，登录时什么也不启动；而 Linux 的 `enabled()` 只看有没有 `Type` 和 `Exec`，`refresh()` 又是空的，开关就一直显示开。

**改动**（`internal/autostart`，与 `daemon.pick` 同一套判定，因为 daemon 导入 autostart，反向导入会成环，只能在这边重写这两步）：

1. PATH 上的 `magpie` 经 `EvalSymlinks` 后与运行中的文件是同一个文件（`os.SameFile`）时，写 PATH 上的那个路径——Nix profile、Homebrew 的链接升级后仍指向新版本；
2. 路径含 `<分隔符>go-build` 的 `go run` 临时构建直接报错拒绝，与 `magpie daemon install` 一致。

**RED→GREEN**：新增 `internal/autostart/self_test.go` 两个测试，在干净 base（`19253f67`，只加测试文件）上先跑出失败原文：

```
--- FAIL: TestSelfNamesThePathLink (0.00s)
    self_test.go:34: self() = "/tmp/go-build3392625092/b001/autostart.test", want the link on PATH "/tmp/TestSelfNamesThePathLink3166114821/001/magpie"
--- FAIL: TestSelfRefusesAGoRunBuild (0.00s)
    self_test.go:47: self() took "/tmp/go-build3392625092/b001/autostart.test", a build `go run` deletes when it ends
```

修复后同两个测试通过：`TestSelfNamesThePathLink`（PATH 上的 magpie 链接指向运行中的同一文件时，`self()` 返回链接本身）、`TestSelfRefusesAGoRunBuild`（`go test` 的测试二进制本身就是 go-build 临时构建，`self()` 拒绝它）。

**测试**：`go test ./internal/autostart/ -count=1` 全绿；`go vet ./...`、`gofmt -l`（空）、`go build ./...`、`GOOS=linux GOARCH=amd64 go build -tags nogui ./...`、`GOOS=windows GOARCH=amd64 go build ./...` 都通过。改动顺带让三个直接调 `Set(true)` 的 darwin 测试补上了 `APPIMAGE`（测试二进制本身就是 go-build 构建会被拒绝），darwin 侧只做了 `GOOS=darwin go build/vet` 编译级验证。

**没试的**：没有在真实的 Nix 或 Homebrew 环境验证升级回收路径；已经写坏的旧记录不会自动修复（Linux 的 `refresh()` 仍是空的），需要把开关关一次再开，重新写入的是 PATH 路径。
