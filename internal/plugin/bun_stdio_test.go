package plugin

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// #5063: a plugin's real Bun children must not write onto the RPC wire.
func TestBunStandardFilesDoNotCorruptHostRPC(t *testing.T) {
	bun, err := exec.LookPath("bun")
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	hostPath := filepath.Join(dir, "host.js")
	if err := os.WriteFile(hostPath, hostJS, 0600); err != nil {
		t.Fatal(err)
	}
	pluginPath, err := filepath.Abs("testdata/bunstdio")
	if err != nil {
		t.Fatal(err)
	}
	frame, err := json.Marshal(map[string]any{"id": 1, "method": "init", "params": map[string]any{
		"authPath": filepath.Join(dir, "auth.json"), "directory": dir,
		"plugins": []any{map[string]any{"spec": "bunstdio", "target": pluginPath}},
	}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, bun, hostPath)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "HOME="+dir, "XDG_CONFIG_HOME="+dir)
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	errout, err := cmd.StderrPipe()
	if err != nil {
		t.Fatal(err)
	}
	var out, stderr bytes.Buffer
	cmd.Stdout = &out
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if _, err := in.Write(append(frame, '\n')); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		scan := bufio.NewScanner(errout)
		for scan.Scan() {
			line := scan.Text()
			stderr.WriteString(line + "\n")
			if line == "READY-TO-READ" {
				_, err := in.Write([]byte("{\"id\":2,\"method\":\"gatewayStates\"}\n"))
				if err != nil {
					done <- err
					return
				}
				if err := in.Close(); err != nil {
					done <- err
					return
				}
			}
		}
		done <- scan.Err()
	}()
	readErr := <-done
	err = cmd.Wait()
	if readErr != nil {
		t.Fatal(readErr)
	}
	if err != nil {
		t.Fatalf("host: %v; stdout %q; stderr %q", err, out.String(), stderr.String())
	}
	scan := bufio.NewScanner(&out)
	frames := 0
	seen := map[int]bool{}
	for scan.Scan() {
		var msg struct {
			ID     int `json:"id"`
			Result struct {
				Plugins []struct {
					Error string `json:"error"`
				} `json:"plugins"`
			} `json:"result"`
			Error any `json:"error"`
		}
		if err := json.Unmarshal(scan.Bytes(), &msg); err != nil {
			t.Errorf("non-RPC output %q: %v", scan.Text(), err)
			continue
		}
		frames++
		if seen[msg.ID] || msg.Error != nil || (msg.ID != 1 && msg.ID != 2) || (msg.ID == 1 && (len(msg.Result.Plugins) != 1 || msg.Result.Plugins[0].Error != "")) {
			t.Errorf("invalid response: %s", scan.Text())
		}
		seen[msg.ID] = true
	}
	if err := scan.Err(); err != nil {
		t.Fatal(err)
	}
	if frames != 2 {
		t.Errorf("got %d RPC frames, want 2", frames)
	}
	if !strings.Contains(stderr.String(), "READ-BYTES:0\n") {
		t.Errorf("child read host input: %q", stderr.String())
	}
	if n := strings.Count(stderr.String(), "CHILD-SENTINEL\n"); n != 8 {
		t.Errorf("stderr has %d child outputs, want 8: %q", n, stderr.String())
	}
}
