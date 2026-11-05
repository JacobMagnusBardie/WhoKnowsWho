#!/usr/bin/env bash
# Hardens SSH on the production VM (issue #85):
#   - turns off password and keyboard-interactive login, so only SSH keys work
#   - turns off root login
#   - installs fail2ban and bans an IP for 1 hour after 5 failed logins in 10 minutes
#
# Run on the VM as the admin user, from a session you keep open until the check at the end
# passes:
#   scp scripts/harden-vm.sh azure:/tmp/ && ssh azure 'sudo bash /tmp/harden-vm.sh'
# Safe to run again: it rewrites its own config files and restarts the services.
set -euo pipefail

if [[ $EUID -ne 0 ]]; then
    echo "Run with sudo" >&2
    exit 1
fi

# Refuse to turn off passwords unless the admin user can already log in with a key,
# otherwise this script would lock everyone out of the VM.
admin_user="${SUDO_USER:-azureuser}"
admin_home=$(getent passwd "$admin_user" | cut -d: -f6)
if [[ ! -s "$admin_home/.ssh/authorized_keys" ]]; then
    echo "No SSH key in $admin_home/.ssh/authorized_keys – not disabling password login" >&2
    exit 1
fi

# --- SSH -------------------------------------------------------------------------------
# sshd uses the FIRST value it reads for each setting, and cloud images ship their own files
# in sshd_config.d (e.g. 50-cloud-init.conf with PasswordAuthentication yes). The 00- prefix
# makes this file win.
if ! grep -qE '^\s*Include\s+/etc/ssh/sshd_config\.d/\*\.conf' /etc/ssh/sshd_config; then
    echo "/etc/ssh/sshd_config does not include sshd_config.d – add the settings there by hand" >&2
    exit 1
fi
cat > /etc/ssh/sshd_config.d/00-hardening.conf <<'EOF'
# Managed by scripts/harden-vm.sh in the WhoKnowsWho repo (issue #85)
PasswordAuthentication no
KbdInteractiveAuthentication no
PermitRootLogin no
EOF
chmod 600 /etc/ssh/sshd_config.d/00-hardening.conf

# Check the full config before reloading. If it is invalid, remove the file again: left in
# place, it would stop sshd from starting after the next reboot.
if ! sshd -t; then
    rm -f /etc/ssh/sshd_config.d/00-hardening.conf
    echo "sshd rejected the new settings – removed them, nothing changed" >&2
    exit 1
fi
# The service is called sshd on RHEL-family systems and ssh on Debian/Ubuntu
if systemctl list-unit-files sshd.service >/dev/null 2>&1; then
    systemctl reload sshd
else
    systemctl reload ssh
fi

# --- fail2ban --------------------------------------------------------------------------
. /etc/os-release
if command -v dnf >/dev/null; then
    # On RHEL, Alma and Rocky fail2ban comes from EPEL; Fedora has it in its own repos
    if [[ "${ID:-}" != "fedora" ]] && ! rpm -q epel-release >/dev/null 2>&1; then
        dnf install -y epel-release ||
            dnf install -y "https://dl.fedoraproject.org/pub/epel/epel-release-latest-$(rpm -E %rhel).noarch.rpm"
    fi
    dnf install -y fail2ban
else
    apt-get update
    # python3-systemd lets fail2ban read sshd's log from the journal (backend = systemd)
    apt-get install -y fail2ban python3-systemd
fi

# The RHEL package bans through firewalld by default. Without firewalld running, use nftables.
banaction=""
if [[ -f /etc/fail2ban/jail.d/00-firewalld.conf ]] && ! systemctl is-active --quiet firewalld; then
    banaction="banaction = nftables-multiport"
fi

cat > /etc/fail2ban/jail.d/whoknows.local <<EOF
# Managed by scripts/harden-vm.sh in the WhoKnowsWho repo (issue #85)
[DEFAULT]
bantime  = 1h
findtime = 10m
maxretry = 5
$banaction

[sshd]
enabled = true
backend = systemd
EOF

systemctl enable fail2ban
systemctl restart fail2ban

# --- Check -----------------------------------------------------------------------------
echo
echo "Effective sshd settings:"
sshd -T | grep -E '^(passwordauthentication|kbdinteractiveauthentication|permitrootlogin) '
echo
# fail2ban needs a moment to load its jails after a restart
for _ in 1 2 3 4 5; do
    fail2ban-client status sshd >/dev/null 2>&1 && break
    sleep 1
done
fail2ban-client status sshd
echo
echo "Done. Before closing this session, open a NEW terminal and check that 'ssh azure' still works."
