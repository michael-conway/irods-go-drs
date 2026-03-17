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

# Function to start iRODS services
start_irods() {
  echo "Attempting to start iRODS services..."
  if command -v irods_control >/dev/null 2>&1; then
      echo "Using irods_control"
      irods_control start || true
  elif [ -f /var/lib/irods/scripts/irods_control.py ]; then
      echo "Using irods_control.py"
      python3 /var/lib/irods/scripts/irods_control.py start || true
  elif [ -f /usr/sbin/irodsctl ]; then
      echo "Using irodsctl"
      /usr/sbin/irodsctl start || true
  else
      echo "WARNING: No iRODS control script found!"
  fi
}

# Function to tail iRODS logs
tail_logs() {
  # Common log paths for 4.x and 5.x
  LOG_PATHS=(
    "/var/lib/irods/log/rodsLog"
    "/var/lib/irods/iRODS/server/log/rodsLog"
  )

  RODSLOG_FILE=""
  for p in "${LOG_PATHS[@]}"; do
    if [ -f "$p" ]; then
      RODSLOG_FILE="$p"
      break
    fi
  done

  if [ -z "$RODSLOG_FILE" ]; then
    echo "rodsLog not found, creating dummy at /var/lib/irods/log/rodsLog"
    mkdir -p /var/lib/irods/log
    touch /var/lib/irods/log/rodsLog
    RODSLOG_FILE="/var/lib/irods/log/rodsLog"
  fi

  echo "Tailing $RODSLOG_FILE..."
  # Use tail -F to be robust against file rotation/recreation
  tail -F "$RODSLOG_FILE" || sleep infinity
}

# If /etc/irods/server_config.json exists we assume iRODS is already configured
if [ -f /etc/irods/server_config.json ]; then
  echo "iRODS already initialized — starting services..."
  start_irods
  tail_logs
  exit 0
fi

echo "First-run initialization of iRODS 5.x..."

wait_for_db

# Create answer file for unattended setup (iRODS 5.0 unattended installation schema)
cat > /tmp/irods_setup_answers.json <<EOF
{
    "admin_password": "$IRODS_ADMIN_PASSWORD",
    "default_resource_directory": "$IRODS_VAULT_DIR",
    "default_resource_name": "demoResc",
    "host_system_information": {
        "service_account_user_name": "irods",
        "service_account_group_name": "irods"
    },
    "server_config": {
        "catalog_service_role": "provider",
        "zone_name": "$IRODS_ZONE",
        "zone_key": "TEMPORARY_ZONE_KEY",
        "negotiation_key": "TEMPORARY_NEGOTIATION_KEY_32CHARS",
        "control_plane_key": "TEMPORARY_CONTROL_PLANE_KEY_32",
        "zone_user": "$IRODS_ADMIN_USER",
        "zone_port": 1247,
        "server_control_plane_port": 1248,
        "first_port_ephemeral_range": 20000,
        "last_port_ephemeral_range": 20199
    },
    "service_account_environment": {
        "irods_host": "$IRODS_HOSTNAME",
        "irods_port": 1247,
        "irods_user_name": "$IRODS_ADMIN_USER",
        "irods_zone_name": "$IRODS_ZONE",
        "irods_default_resource": "demoResc",
        "irods_database_type": "postgres",
        "irods_database_server_hostname": "$IRODS_DB_HOST",
        "irods_database_server_port": $IRODS_DB_PORT,
        "irods_database_name": "$IRODS_DB_NAME",
        "irods_database_user_name": "$IRODS_DB_USER",
        "irods_database_password": "$IRODS_DB_PASSWORD"
    }
}
EOF

# Use Python-based setup script if available (Standard for 5.x)
if [ -f /var/lib/irods/scripts/setup_irods.py ]; then
    echo "Running setup_irods.py..."
    # The --json_configuration_file flag is the correct way to perform an unattended install in iRODS 5.x
    if python3 /var/lib/irods/scripts/setup_irods.py --json_configuration_file /tmp/irods_setup_answers.json > /tmp/setup_irods.log 2>&1; then
        echo "setup_irods.py succeeded with --json_configuration_file"
    else
        echo "setup_irods.py failed with --json_configuration_file. Logs:"
        cat /tmp/setup_irods.log
        exit 1
    fi
else
    echo "setup_irods.py not found, attempting legacy-style setup..."
    # Fallback to legacy if necessary, but 5.x should have setup_irods.py
    /var/lib/irods/packaging/setup_irods.sh < /tmp/irods_setup_answers.json || true
fi

# Start iRODS (setup_irods.py might already start it)
start_irods

# Post-setup configurations
echo "Running testsetup-consortium.sh..."
chmod +x /var/lib/irods/testsetup-consortium.sh
sudo -u irods -E /var/lib/irods/testsetup-consortium.sh

echo "iRODS 5.x initialized. Tailing logs..."
tail_logs
