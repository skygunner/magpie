// #5063: a plugin may pass Bun's own standard files to either spawn API.
export const BunStdioPlugin = async () => {
  if (Bun.stdout !== Bun.stdout || Bun.stdin !== Bun.stdin || !(Bun.stdout instanceof Blob) || !(Bun.stdin instanceof Blob)) throw Error("unexpected Bun standard file identity")
  const read = 'const fs=require("node:fs");process.stderr.write("READY-TO-READ\\n");const b=Buffer.alloc(256);const n=fs.readSync(0,b,0,b.length,null);process.stderr.write("READ-BYTES:"+n+"\\n")'
  const input = Bun.spawnSync([process.execPath, "-e", read], { stdin: Bun.stdin, stdout: "pipe", stderr: Bun.stderr })
  if (input.exitCode !== 0) throw Error(`stdin child exit ${input.exitCode}`)
  const script = 'process.stdout.write("CHILD-SENTINEL\\n")'
  for (const sync of [false, true]) {
    for (const array of [false, true]) {
      for (const stdio of [false, true]) {
        const cmd = [process.execPath, "-e", script]
        const opts = stdio ? { stdio: [Bun.stdin, Bun.stdout, Bun.stderr] } : { stdin: Bun.stdin, stdout: Bun.stdout, stderr: Bun.stderr }
        const child = array ? Bun[sync ? "spawnSync" : "spawn"](cmd, opts) : Bun[sync ? "spawnSync" : "spawn"]({ cmd, ...opts })
        const code = sync ? child.exitCode : await child.exited
        if (code !== 0) throw Error(`child exit ${code}`)
      }
    }
  }
  const child = Bun.spawn([process.execPath, "-e", 'process.stdout.write("pipe-ok");process.stderr.write("err-ok")'], { stdout: "pipe", stderr: "pipe" })
  const [out, err, code] = await Promise.all([new Response(child.stdout).text(), new Response(child.stderr).text(), child.exited])
  if (out !== "pipe-ok" || err !== "err-ok" || code !== 0) throw Error("normal pipes broken")
  const file = Bun.file("custom-output")
  const custom = Bun.spawnSync([process.execPath, "-e", 'process.stdout.write("file-ok")'], { stdout: file, stderr: "pipe" })
  if (custom.exitCode !== 0 || await Bun.file("plugin-work/custom-output").text() !== "file-ok") throw Error("custom file broken")
  return {}
}
