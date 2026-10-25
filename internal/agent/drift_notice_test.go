package agent

import (
	"slices"
	"testing"
)

// A drift detail is shown in the reader's language too (#2867): each one
// is a notice, so the GUI translates it. These are the sentences as they
// read, and what zh and ja show for the same slots.
func TestDriftDetailsAreTranslated(t *testing.T) {
	tables, _ := i18nTables(t)
	for _, c := range []struct {
		en, zh, ja string
		kv         []string
	}{
		{
			en: "{agent}'s config was changed outside magpie: {field} is {now}, not {want} as magpie set it",
			zh: "Codex 的配置在 magpie 之外被改动：model 现在是 gpt-5.6，而不是 magpie 设置的 magpie/gpt-5.6",
			ja: "Codex の設定が magpie の外で変更されました: model は gpt-5.6 で、magpie が設定した magpie/gpt-5.6 ではありません",
			kv: []string{"agent", "Codex", "field", "model", "now", "gpt-5.6", "want", "magpie/gpt-5.6"},
		},
		{
			en: "{agent}'s config was changed outside magpie: {field} is the agent's default, not {want} as magpie set it",
			zh: "Codex 的配置在 magpie 之外被改动：model 现在是该 agent 的默认值，而不是 magpie 设置的 magpie/gpt-5.6",
			ja: "Codex の設定が magpie の外で変更されました: model はエージェントの既定値で、magpie が設定した magpie/gpt-5.6 ではありません",
			kv: []string{"agent", "Codex", "field", "model", "want", "magpie/gpt-5.6"},
		},
		{
			en: "{agent}'s last request ({time}) went to {host}, not magpie — it was started before magpie set it up and still runs on its old config: quit and reopen it",
			zh: "Codex 上一次请求（15:04）发往了 relay.example.com，而不是 magpie——它是在 magpie 设置之前启动的，仍在用旧配置运行：退出并重新打开它",
			ja: "Codex の最後のリクエスト（15:04）は magpie ではなく relay.example.com に送られました。magpie が設定する前に起動したままで、古い設定のまま動いています: 終了して開き直してください",
			kv: []string{"agent", "Codex", "time", "15:04", "host", "relay.example.com"},
		},
		{
			en: "{agent} couldn't reach magpie at {time} — magpie wasn't running then, so it has none of magpie's models: quit and reopen it",
			zh: "Codex 在 15:04 无法连接 magpie——当时 magpie 没有运行，所以它拿不到 magpie 的任何模型：退出并重新打开它",
			ja: "Codex は 15:04 に magpie へ到達できませんでした。そのとき magpie が動いていなかったため、magpie のモデルを一つも持っていません: 終了して開き直してください",
			kv: []string{"agent", "Codex", "time", "15:04"},
		},
		{
			en: "{agent} was used at {time} but none of its requests reached magpie — one started before magpie set it up still runs on its old config: restart it",
			zh: "Codex 在 15:04 被使用过，但它的请求没有一个到达 magpie——有个在 magpie 设置之前启动的进程仍在用旧配置运行：重启它",
			ja: "Codex は 15:04 に使われましたが、そのリクエストはどれも magpie に届きませんでした。magpie が設定する前に起動したものが古い設定のまま動いています: 再起動してください",
			kv: []string{"agent", "Codex", "time", "15:04"},
		},
		{
			en: "{agent} is pointed at magpie at {addr}, where magpie listens for WSL, but nothing answers there from WSL {wsl}: Windows' firewall keeps WSL out. Allow magpie in when Windows asks, or, in PowerShell as administrator: {command}. On Windows 11, networkingMode=mirrored under [wsl2] in %UserProfile%\\.wslconfig lets WSL reach magpie at 127.0.0.1 instead.",
			zh: "Codex 指向的 magpie 地址是 http://172.20.0.1:3425，magpie 在此为 WSL 监听，但从 WSL Ubuntu 访问时没有任何响应：Windows 防火墙把 WSL 挡在外面。请在 Windows 询问时允许 magpie，或以管理员身份在 PowerShell 中运行：New-NetFirewallRule -DisplayName 'magpie (WSL)' -Program 'magpie.exe'。在 Windows 11 上，在 %UserProfile%\\.wslconfig 的 [wsl2] 下设置 networkingMode=mirrored，WSL 即可改用 127.0.0.1 访问 magpie。",
			ja: "Codex は http://172.20.0.1:3425 の magpie を指しています。そこは magpie が WSL 向けに待ち受けているアドレスですが、WSL Ubuntu からは応答がありません: Windows のファイアウォールが WSL を遮っています。Windows の確認で magpie を許可するか、管理者として PowerShell で次を実行してください: New-NetFirewallRule -DisplayName 'magpie (WSL)' -Program 'magpie.exe'。Windows 11 では、%UserProfile%\\.wslconfig の [wsl2] に networkingMode=mirrored を設定すると、WSL は 127.0.0.1 で magpie に到達できます。",
			kv: []string{"agent", "Codex", "addr", "http://172.20.0.1:3425", "wsl", "Ubuntu", "command", "New-NetFirewallRule -DisplayName 'magpie (WSL)' -Program 'magpie.exe'"},
		},
		{
			en: "{agent} is pointed at magpie at {addr}, where nothing answers from this computer, though magpie answers at {url}.",
			zh: "Codex 指向的 magpie 地址是 http://192.168.1.9:3425，从本机访问没有任何响应，但 magpie 在 http://127.0.0.1:3425 是有响应的。",
			ja: "Codex は http://192.168.1.9:3425 の magpie を指していますが、このコンピューターからは応答がありません。magpie は http://127.0.0.1:3425 では応答しています。",
			kv: []string{"agent", "Codex", "addr", "http://192.168.1.9:3425", "url", "http://127.0.0.1:3425"},
		},
		{
			en: "Whatever forwarded {host} to magpie isn't listening now; start it again, or point {agent} at {url}.",
			zh: "把 192.168.1.9:3425 转发到 magpie 的东西现在没有在监听；请重新启动它，或把 Codex 指向 http://127.0.0.1:3425。",
			ja: "192.168.1.9:3425 を magpie に転送していたものが今は待ち受けていません。もう一度起動するか、Codex を http://127.0.0.1:3425 に向けてください。",
			kv: []string{"host", "192.168.1.9:3425", "agent", "Codex", "url", "http://127.0.0.1:3425"},
		},
		{
			en: "Under WSL's NAT networking, Windows as WSL sees it reaches magpie only while something listens on {host}: magpie itself with Settings › Network and sharing › Share on local network on (it then listens on every address, and asks other computers for a gateway key), or a forward of your own (netsh interface portproxy, which may need setting up again after Windows restarts). Once it answers here and WSL still can't connect, Windows' firewall may be keeping WSL out: allow the port {port} for the WSL network. On Windows 11, networkingMode=mirrored under [wsl2] in %UserProfile%\\.wslconfig lets WSL reach magpie at 127.0.0.1 instead.",
			zh: "在 WSL 的 NAT 网络下，WSL 看到的 Windows 只有在 172.20.0.1:3425 上有人监听时才能访问 magpie：可以是 magpie 自身（在“设置 › 网络与共享”中打开“在局域网共享”，它随后会监听所有地址，并向其他电脑索取 gateway key），也可以是你自己的转发（netsh interface portproxy，Windows 重启后可能需要重新设置）。这里能通之后如果 WSL 仍连不上，可能是 Windows 防火墙把 WSL 挡在外面：请为 WSL 网络放行 3425 端口。在 Windows 11 上，在 %UserProfile%\\.wslconfig 的 [wsl2] 下设置 networkingMode=mirrored，WSL 即可改用 127.0.0.1 访问 magpie。",
			ja: "WSL の NAT ネットワークでは、WSL から見た Windows が magpie に届くのは 172.20.0.1:3425 で何かが待ち受けている間だけです: magpie 自身（設定 › ネットワークと共有 › ローカルネットワークで共有 をオンにすると、すべてのアドレスで待ち受け、他のコンピューターに gateway key を求めます）か、自分で用意した転送（netsh interface portproxy。Windows の再起動後に設定し直しが必要な場合があります）です。ここで応答するようになっても WSL からつながらない場合、Windows のファイアウォールが WSL を遮っている可能性があります: WSL のネットワークに対してポート 3425 を許可してください。Windows 11 では、%UserProfile%\\.wslconfig の [wsl2] に networkingMode=mirrored を設定すると、WSL は 127.0.0.1 で magpie に到達できます。",
			kv: []string{"host", "172.20.0.1:3425", "port", "3425"},
		},
		{
			en: "WSL {distro} reaches Windows at {to} now, not {addr} as before: Use {to} points {agent} there.",
			zh: "WSL Ubuntu 现在通过 http://172.20.0.1:3425 访问 Windows，而不是之前的 http://192.168.1.9:3425：使用 http://172.20.0.1:3425 会把 Codex 指向那里。",
			ja: "WSL Ubuntu は Windows に http://172.20.0.1:3425 で到達するようになりました（以前の http://192.168.1.9:3425 ではありません）: http://172.20.0.1:3425 を使用 を押すと Codex をそこに向けます。",
			kv: []string{"distro", "Ubuntu", "to", "http://172.20.0.1:3425", "addr", "http://192.168.1.9:3425", "agent", "Codex"},
		},
		{
			en: "{agent}'s {key} ({file}) is gone, so it no longer reaches magpie",
			zh: "OpenCode 的 baseURL（opencode.json）已不存在，因此不再走 magpie",
			ja: "OpenCode の baseURL（opencode.json）がなくなり、magpie に届かなくなりました",
			kv: []string{"agent", "OpenCode", "key", "baseURL", "file", "opencode.json"},
		},
		{
			en: "{agent}'s {key} ({file}) is {value}, not magpie's gateway at {want}",
			zh: "Cline 的 baseURL（settings.json）是 https://relay.example.com/v1，不是 magpie 的网关 http://127.0.0.1:3425/v1",
			ja: "Cline の baseURL（settings.json）は https://relay.example.com/v1 で、magpie のゲートウェイ http://127.0.0.1:3425/v1 ではありません",
			kv: []string{"agent", "Cline", "key", "baseURL", "file", "settings.json", "value", "https://relay.example.com/v1", "want", "http://127.0.0.1:3425/v1"},
		},
		{
			en: "{agent}'s {key} ({file}) was changed, so magpie's gateway won't take its requests",
			zh: "Cline 的 apiKey（settings.json）被改动了，magpie 的网关不会接受它的请求",
			ja: "Cline の apiKey（settings.json）が変更されたため、magpie のゲートウェイはそのリクエストを受け付けません",
			kv: []string{"agent", "Cline", "key", "apiKey", "file", "settings.json"},
		},
	} {
		if !slices.Contains(Notices(), c.en) {
			t.Errorf("drift detail isn't a notice, so it stays English: %q", c.en)
		}
		for _, l := range []struct{ name, want string }{{"zh", c.zh}, {"ja", c.ja}} {
			tr, ok := tables[l.name][c.en]
			if !ok {
				t.Fatalf("%s has no translation of %q", l.name, c.en)
			}
			if got := notice(tr).say(c.kv...); got != l.want {
				t.Errorf("%s: got %q, want %q", l.name, got, l.want)
			}
		}
	}
}
