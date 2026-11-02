# Delivered branch limitations

- B1 #4118: synthetic interleaving test, not proof of all observed 503s.
- B2 #4137: three commits; no real WebDAV/S3 server tested.
- B3 #4136: six commits; Windows build/vet passed, Windows tests and actual WSL not run.
- B5 #4144: two commits; conservative mitigation only, later records require source repair and detectable metadata change.
- B6 #4141: three commits; ambiguous initial blind-write conflicts stop for manual reconciliation, no real DAV server test.

Apply the whole issue-body diff or all branch commits, not the final commit alone. B4 remains incomplete.
