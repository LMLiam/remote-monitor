#!/usr/bin/env bash
set -euo pipefail

if [ -z "${AUTHORIZED_KEY:-}" ]; then
  echo "AUTHORIZED_KEY is required" >&2
  exit 64
fi

install -d -m 700 -o monitor -g monitor /home/monitor/.ssh
printf '%s\n' "${AUTHORIZED_KEY}" > /home/monitor/.ssh/authorized_keys
chown monitor:monitor /home/monitor/.ssh/authorized_keys
chmod 600 /home/monitor/.ssh/authorized_keys

ssh-keygen -A

exec /usr/sbin/sshd -D -e \
  -o PasswordAuthentication=no \
  -o KbdInteractiveAuthentication=no \
  -o PermitRootLogin=no \
  -o PubkeyAuthentication=yes \
  -o AuthorizedKeysFile=.ssh/authorized_keys \
  -o AllowUsers=monitor \
  -o UsePAM=no
