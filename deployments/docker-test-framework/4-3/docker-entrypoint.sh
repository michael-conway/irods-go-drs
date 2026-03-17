#!/usr/bin/env bash
set -euo pipefail
IFS=$'\n\t'

# Defaults (override with environment variables)
: "${IRODS_ZONE:=tempZone}"
: "${IRODS_ADMIN_USER:=rods}"
: "${IRODS_ADMIN_PASSWORD:=rods}"
: "${IRODS_DB_HOST:=postgres}"
: "${IRODS_DB_PORT:=5432}"
: "${IRODS_DB_NAME:=ICAT}"
: "${IRODS_DB_USER:=irods}"
: "${IRODS_DB_PASSWORD:=irods}"
: "${IRODS_VAULT_DIR:=/var/lib/irods/iRODS/Vault}"
: "${IRODS_HOSTNAME:=$(hostname -f)}"

# Helper: wait for DB to be reachable
wait_for_db() {
  for i in $(seq 1 60); do
    if nc -z "$IRODS_DB_HOST" "$IRODS_DB_PORT"; then
      echo "Postgres reachable at $IRODS_DB_HOST:$IRODS_DB_PORT"
      return 0
    fi
    echo "Waiting for Postgres ($i/60)..."
    sleep 2
  done
  echo "ERROR: Postgres did not become reachable" >&2
  return 1
}

# If /etc/irods/server_config.json exists we assume iRODS is already configured
if [ -f /etc/irods/server_config.json ]; then
  echo "iRODS already initialized — starting services..."
  /usr/sbin/irodsctl start || true
  tail -f /var/lib/irods/log/rodsLog || sleep infinity
  exit 0
fi

echo "First-run initialization of iRODS..."

wait_for_db

# Create answer file for unattended setup (legacy-style)
cat > /tmp/irods_setup_answers <<EOF
$IRODS_ZONE
$IRODS_DB_HOST
$IRODS_DB_PORT
$IRODS_DB_NAME
$IRODS_DB_USER
$IRODS_DB_PASSWORD
$IRODS_ADMIN_USER
$IRODS_ADMIN_PASSWORD
$IRODS_VAULT_DIR
EOF

# Try the package-provided installer helpers (names can vary between package builds)
if command -v /var/lib/irods/packaging/install_irods.sh >/dev/null 2>&1; then
  echo "Running /var/lib/irods/packaging/install_irods.sh (non-interactive)..."
  /var/lib/irods/packaging/install_irods.sh < /tmp/irods_setup_answers
elif command -v irodssetup >/dev/null 2>&1; then
  echo "Running irodssetup..."
  irodssetup < /tmp/irods_setup_answers
elif command -v /var/lib/irods/irodssetup >/dev/null 2>&1; then
  /var/lib/irods/irodssetup < /tmp/irods_setup_answers
else
  echo "No installer helper found — attempting 'irods_setup' package script..."
  if [ -x /usr/sbin/irodsctl ]; then
    echo "irodsctl exists; attempting manual config steps may be required."
    # fallthrough: admin may need to run interactive config
    echo "ERROR: automated installer helper not found in image. Please run the interactive installer inside the container."
    exit 1
  fi
fi

# Start iRODS
/usr/sbin/irodsctl start || true

echo "iRODS initialization complete — tailing logs..."
tail -f /var/lib/irods/log/rodsLog || sleep infinity

#!/bin/sh
iadmin mkuser test1 rodsadmin

iadmin moduser test1 password test

iadmin aua test1 test1DN

iadmin mkuser test2 rodsuser

iadmin moduser test2 password test

iadmin mkuser test3 rodsuser

iadmin moduser test3 password test

iadmin mkuser anonymous rodsuser

iadmin atg public anonymous



echo "iRODS initialized. Tailing logs..."
tail -f /dev/null || sleep infinity