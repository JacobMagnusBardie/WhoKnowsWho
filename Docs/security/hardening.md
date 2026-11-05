# Server and pipeline hardening

What is hardened, where it lives, and how to check it (issue #85).

## VM

| What | Where | Check |
|---|---|---|
| SSH keys only: no password or keyboard-interactive login, no root login | `/etc/ssh/sshd_config.d/00-hardening.conf`, written by `scripts/harden-vm.sh` | `sudo sshd -T \| grep -E '^(passwordauthentication\|permitrootlogin)'` prints `no` for both |
| fail2ban bans an IP for 1 hour after 5 failed SSH logins in 10 minutes | `/etc/fail2ban/jail.d/whoknows.local`, written by `scripts/harden-vm.sh` | `sudo fail2ban-client status sshd` |
| The app is reachable only through nginx (port 80); port 8080 is bound to localhost | `compose.yaml` | `curl http://whoknows.swag.dk:8080/` from outside fails |

Run the script from a session you keep open, then check that a new `ssh azure` still works
before closing it:

```
scp scripts/harden-vm.sh azure:/tmp/ && ssh azure 'sudo bash /tmp/harden-vm.sh'
```

It refuses to run if the admin user has no SSH key in `authorized_keys`. If `sshd -t`
rejects the new settings, it removes them again.

Docker publishes ports with its own firewall rules, which go around firewalld/ufw. That is why
8080 is closed by binding it to `127.0.0.1` in `compose.yaml` rather than by a firewall rule.

## Pipeline

Based on GitHub's [security hardening for GitHub Actions](https://docs.github.com/en/actions/security-for-github-actions/security-guides/security-hardening-for-github-actions).

- **Least-privilege tokens.** Every workflow has a top-level `permissions:` block. `deploy.yaml`
  starts from `permissions: {}`, and each job asks only for what it needs.
- **Third-party actions are pinned to a commit SHA**, with the version as a comment. A tag
  can be moved to other code; a SHA cannot. Dependabot (`github-actions`) updates the SHA and
  the comment together. The `appleboy/*` actions receive the VM's SSH key, so they matter most.
- **No untrusted input in `run:`.** The PR title in `release-check.yaml` goes through `env:`,
  never `${{ }}` inside the script.
- **Trivy scans the app image on every PR** (`trivy` job in `ci.yaml`) and fails on HIGH or
  CRITICAL vulnerabilities that have a fix. `trivy-action` is pinned to the v0.36.0 SHA:
  tags 0.0.1–0.34.2 were hijacked in March 2026 (CVE-2026-33634).
