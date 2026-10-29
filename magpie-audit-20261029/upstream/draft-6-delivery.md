补丁已推送：[fix/4141-dav-put-version](https://github.com/skygunner/magpie/tree/fix/4141-dav-put-version)。完整 inline diff 在 issue 正文。

提交历史：https://github.com/skygunner/magpie/commits/fix/4141-dav-put-version/
最终提交：https://github.com/skygunner/magpie/commit/3452982b8ad99aa3c74f86c326a65f5f53ad1844

首次盲写后的不明确竞争会停止同步并保留两侧，需手工协调；尚未在真实 WebDAV 服务测试。分支在 skygunner/magpie，欢迎直接 cherry-pick 吸收。
