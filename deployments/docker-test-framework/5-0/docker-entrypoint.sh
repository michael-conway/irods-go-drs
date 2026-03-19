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
: "${IRODS_HOSTNAME:=irods-provider}"

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
  # Ensure PID directory exists and is writable
  mkdir -p /var/run/irods
  chown irods:irods /var/run/irods
  chmod 775 /var/run/irods

  echo "Container IP/Hostname info:"
  hostname -i || true
  hostname -f || true
  ip addr show || true

  # Before starting, check if database is actually initialized if it's a provider
  if [ -f /etc/irods/server_config.json ]; then
      # We could check if ICAT tables exist, but setup_irods.py should have done that
      :
  fi

  # Prioritize python module in 5.x
  if python3 -c "import irods_control" >/dev/null 2>&1; then
      echo "Using python3 -m irods_control"
      # We need to make sure irods_control is run with the right environment
      sudo -u irods -E python3 -m irods_control start > /tmp/irods_control.stdout 2> /tmp/irods_control.stderr || true
      if grep -qi "error" /tmp/irods_control.stderr; then
          echo "irods_control reported errors, checking stderr:"
          cat /tmp/irods_control.stderr
      fi
  # Common locations for irods_control.py in 5.x
  elif [ -f /var/lib/irods/irods_control.py ]; then
      echo "Using /var/lib/irods/irods_control.py"
      python3 /var/lib/irods/irods_control.py start || true
  elif [ -f /var/lib/irods/scripts/irods_control.py ]; then
      echo "Using /var/lib/irods/scripts/irods_control.py"
      python3 /var/lib/irods/scripts/irods_control.py start || true
  elif [ -f /var/lib/irods/scripts/irods/irods_control.py ]; then
      echo "Using /var/lib/irods/scripts/irods/irods_control.py"
      python3 /var/lib/irods/scripts/irods/irods_control.py start || true
  elif [ -f /usr/sbin/irods_control ]; then
      echo "Using /usr/sbin/irods_control"
      /usr/sbin/irods_control start || true
  elif [ -f /usr/bin/irods_control ]; then
      echo "Using /usr/bin/irods_control"
      /usr/bin/irods_control start || true
  elif [ -f /usr/sbin/irodsServer ]; then
      echo "Using /usr/sbin/irodsServer directly (fallback)"
      # Note: irodsServer might need to be run with certain arguments or environment
      # In 5.x, irodsServer expects certain environment variables normally set by irods_control
      export IRODS_SERVER_CONTROL_PLANE_KEY="32_byte_server_negotiation_key__"
      export IRODS_SERVER_CONTROL_PLANE_PORT=1248
      export IRODS_SERVER_NEGOTIATION_KEY="32_byte_server_negotiation_key__"
      sudo -u irods -E /usr/sbin/irodsServer > /tmp/irodsServer.stdout 2> /tmp/irodsServer.stderr &
      if [ -f /usr/sbin/irodsDelayServer ]; then
          echo "Also starting /usr/sbin/irodsDelayServer..."
          sudo -u irods -E /usr/sbin/irodsDelayServer > /tmp/irodsDelayServer.stdout 2> /tmp/irodsDelayServer.stderr &
      fi
  elif command -v irods_control >/dev/null 2>&1; then
      echo "Using irods_control"
      irods_control start || true
  elif [ -f /usr/sbin/irodsctl ]; then
      echo "Using irodsctl"
      /usr/sbin/irodsctl start || true
  elif [ -f /var/lib/irods/scripts/setup_irods.py ]; then
      echo "Using setup_irods.py for start (legacy-ish)"
      sudo -u irods python3 /var/lib/irods/scripts/setup_irods.py --start || true
  else
      echo "WARNING: No iRODS control script found!"
      echo "Checking common locations..."
      find /var/lib/irods -name "irods_control*" || true
      find /usr -name "irods_control*" || true
      echo "Checking for irods related files in /usr/bin and /usr/sbin:"
      ls /usr/bin/irods* || true
      ls /usr/sbin/irods* || true
      echo "Checking /var/lib/irods content:"
      ls -R /var/lib/irods | head -n 50
  fi

  # Wait for iRODS to be ready
  echo "Waiting for iRODS server to become responsive..."
  for i in $(seq 1 30); do
      if sudo -u irods iadmin lr >/dev/null 2>&1; then
          echo "iRODS server is ready!"
          return 0
      fi
      if [ -f /tmp/irodsServer.stderr ]; then
          echo "Recent irodsServer stderr:"
          tail -n 5 /tmp/irodsServer.stderr
      fi
      echo "Checking if port 1247 is listening..."
      ss -tuln | grep :1247 || netstat -tuln | grep :1247 || echo "Port 1247 not found in netstat/ss"
      sleep 2
      echo "Still waiting ($i/30)..."
  done
  echo "WARNING: iRODS server did not become responsive within 60 seconds."
}

# Function to tail iRODS logs
tail_logs() {
  # Always ensure irods user environment exists before tailing (and staying in container)
  # Overwrite with correct values to avoid USER_RODS_HOSTNAME_ERR
  echo "Ensuring irods user environment is correct..."
  mkdir -p /var/lib/irods/.irods
  cat > /var/lib/irods/.irods/irods_environment.json <<EOF
{
    "irods_host": "$IRODS_HOSTNAME",
    "irods_port": 1247,
    "irods_user_name": "$IRODS_ADMIN_USER",
    "irods_zone_name": "$IRODS_ZONE",
    "irods_default_resource": "demoResc",
    "irods_client_server_policy": "CS_NEG_REFUSE",
    "irods_client_server_negotiation_key": "32_byte_server_negotiation_key__",
    "irods_encryption_algorithm": "AES-256-CBC",
    "irods_encryption_key_size": 32,
    "irods_encryption_num_hash_rounds": 16,
    "irods_encryption_salt_size": 8
}
EOF
  chown -R irods:irods /var/lib/irods/.irods
  # Ensure interactive shells for irods pick up the right environment
  echo 'export IRODS_ENVIRONMENT_FILE=/var/lib/irods/.irods/irods_environment.json' > /var/lib/irods/.bashrc
  echo 'export HOME=/var/lib/irods' >> /var/lib/irods/.bashrc
  chown irods:irods /var/lib/irods/.bashrc
  
  # Initialize authentication for irods user
  echo "Initializing iRODS session for irods user..."
  su - irods -c "echo '$IRODS_ADMIN_PASSWORD' | iinit" || true

  # Ensure root can use icommands too, but irods user shouldn't try to read root's files
  mkdir -p /root/.irods
  cp /var/lib/irods/.irods/irods_environment.json /root/.irods/irods_environment.json
  chmod 600 /root/.irods/irods_environment.json
  chmod 700 /root/.irods

  # Common log paths for 4.x and 5.x
  LOG_PATHS=(
    "/var/lib/irods/log/irods.log"
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
    echo "rodsLog not found, creating dummy at /var/lib/irods/log/irods.log"
    mkdir -p /var/lib/irods/log
    touch /var/lib/irods/log/irods.log
    RODSLOG_FILE="/var/lib/irods/log/irods.log"
  fi

  echo "Tailing $RODSLOG_FILE..."
  # Use tail -F to be robust against file rotation/recreation
  tail -F "$RODSLOG_FILE" || sleep infinity
}

# If /etc/irods/server_config.json exists we assume iRODS is already configured
if [ -f /etc/irods/server_config.json ]; then
  echo "iRODS already initialized — starting services..."
  # Ensure irods user environment exists (always overwrite to ensure correct hostname)
  echo "Updating irods user environment..."
  mkdir -p /var/lib/irods/.irods
  cat > /var/lib/irods/.irods/irods_environment.json <<EOF
{
    "irods_host": "$IRODS_HOSTNAME",
    "irods_port": 1247,
    "irods_user_name": "$IRODS_ADMIN_USER",
    "irods_zone_name": "$IRODS_ZONE",
    "irods_default_resource": "demoResc",
    "irods_client_server_policy": "CS_NEG_REFUSE",
    "irods_client_server_negotiation_key": "32_byte_server_negotiation_key__",
    "irods_encryption_algorithm": "AES-256-CBC",
    "irods_encryption_key_size": 32,
    "irods_encryption_num_hash_rounds": 16,
    "irods_encryption_salt_size": 8
}
EOF
  chown -R irods:irods /var/lib/irods/.irods
  # Ensure interactive shells for irods pick up the right environment
  echo 'export IRODS_ENVIRONMENT_FILE=/var/lib/irods/.irods/irods_environment.json' > /var/lib/irods/.bashrc
  echo 'export HOME=/var/lib/irods' >> /var/lib/irods/.bashrc
  chown irods:irods /var/lib/irods/.bashrc
  
  # Initialize authentication for irods user
  echo "Initializing iRODS session for irods user..."
  su - irods -c "echo '$IRODS_ADMIN_PASSWORD' | iinit" || true

  # Ensure root can use icommands too
  mkdir -p /root/.irods
  cp /var/lib/irods/.irods/irods_environment.json /root/.irods/irods_environment.json
  chmod 600 /root/.irods/irods_environment.json
  chmod 700 /root/.irods
  
  start_irods

  # Still try to run testsetup if it hasn't run yet
  echo "Checking if testsetup-consortium.sh needs to run..."
  chmod +x /var/lib/irods/testsetup-consortium.sh
  if ! su - irods -c "iadmin lu test1" >/dev/null 2>&1; then
      echo "Running testsetup-consortium.sh..."
      su - irods -c "/var/lib/irods/testsetup-consortium.sh"
  else
      echo "testsetup-consortium.sh already run."
  fi

  tail_logs
  exit 0
fi

echo "First-run initialization of iRODS 5.x..."

wait_for_db

# Create answer file for unattended setup (iRODS 5.0 unattended installation schema)
# First, ensure irods environment is set up so setup_irods.py can use correct hostname
echo "Pre-setting irods user environment..."
mkdir -p /var/lib/irods/.irods
cat > /var/lib/irods/.irods/irods_environment.json <<EOF
{
    "irods_host": "$IRODS_HOSTNAME",
    "irods_port": 1247,
    "irods_user_name": "$IRODS_ADMIN_USER",
    "irods_zone_name": "$IRODS_ZONE",
    "irods_default_resource": "demoResc",
    "irods_client_server_policy": "CS_NEG_REFUSE",
    "irods_client_server_negotiation_key": "32_byte_server_negotiation_key__",
    "irods_encryption_algorithm": "AES-256-CBC",
    "irods_encryption_key_size": 32,
    "irods_encryption_num_hash_rounds": 16,
    "irods_encryption_salt_size": 8
}
EOF
chown -R irods:irods /var/lib/irods/.irods

echo "Generating /tmp/irods_setup_answers.json..."
cat > /tmp/irods_setup_answers.json <<EOF
{
    "admin_password": "$IRODS_ADMIN_PASSWORD",
    "default_resource_directory": "$IRODS_VAULT_DIR",
    "default_resource_name": "demoResc",
    "host_system_information": {
        "service_account_user_name": "irods",
        "service_account_group_name": "irods",
        "hostname_cache_shared_memory_name": "irods_hostname_cache",
        "dns_cache_shared_memory_name": "irods_dns_cache"
    },
    "service_account_environment": {
        "irods_client_server_policy": "CS_NEG_REFUSE",
        "irods_connection_pool_refresh_time_in_seconds": 300,
        "irods_cwd": "/$IRODS_ZONE/home/$IRODS_ADMIN_USER",
        "irods_default_hash_scheme": "SHA256",
        "irods_default_number_of_transfer_threads": 4,
        "irods_default_resource": "demoResc",
        "irods_encryption_algorithm": "AES-256-CBC",
        "irods_encryption_key_size": 32,
        "irods_encryption_num_hash_rounds": 16,
        "irods_encryption_salt_size": 8,
        "irods_home": "/$IRODS_ZONE/home/$IRODS_ADMIN_USER",
        "irods_host": "$IRODS_HOSTNAME",
        "irods_match_hash_policy": "compatible",
        "irods_maximum_size_for_single_buffer_in_megabytes": 32,
        "irods_port": 1247,
        "irods_transfer_buffer_size_for_parallel_transfer_in_megabytes": 4,
        "irods_user_name": "$IRODS_ADMIN_USER",
        "irods_zone_name": "$IRODS_ZONE",
        "schema_name": "service_account_environment",
        "schema_version": "v5"
    },
    "server_config": {
        "advanced_settings": {
            "checksum_read_buffer_size_in_bytes": 1048576,
            "default_number_of_transfer_threads": 4,
            "default_temporary_password_lifetime_in_seconds": 120,
            "delay_rule_executors": [],
            "delay_server_sleep_time_in_seconds": 30,
            "hostname_cache": {
                "eviction_age_in_seconds": 3600,
                "cache_clearer_sleep_time_in_seconds": 600,
                "shared_memory_size_in_bytes": 2500000,
                "shared_memory_instance": "irods_hostname_cache"
            },
            "dns_cache": {
                "eviction_age_in_seconds": 3600,
                "cache_clearer_sleep_time_in_seconds": 600,
                "shared_memory_size_in_bytes": 5000000,
                "shared_memory_instance": "irods_dns_cache"
            },
            "maximum_size_for_single_buffer_in_megabytes": 32,
            "maximum_size_of_delay_queue_in_bytes": 0,
            "maximum_temporary_password_lifetime_in_seconds": 1000,
            "migrate_delay_server_sleep_time_in_seconds": 5,
            "number_of_concurrent_delay_rule_executors": 4,
            "stacktrace_file_processor_sleep_time_in_seconds": 10,
            "transfer_buffer_size_for_parallel_transfer_in_megabytes": 4,
            "transfer_chunk_size_for_parallel_transfer_in_megabytes": 40
        },
        "catalog_provider_hosts": [
            "$IRODS_HOSTNAME"
        ],
        "catalog_service_role": "provider",
        "client_server_policy": "CS_NEG_REFUSE",
        "connection_pool_refresh_time_in_seconds": 300,
        "controlled_user_connection_list": {
            "control_type": "denylist",
            "users": []
        },
        "default_dir_mode": "0750",
        "default_file_mode": "0600",
        "default_hash_scheme": "SHA256",
        "default_resource_name": "demoResc",
        "encryption": {
            "algorithm": "AES-256-CBC",
            "key_size": 32,
            "num_hash_rounds": 16,
            "salt_size": 8
        },
        "environment_variables": {},
        "federation": [],
        "graceful_shutdown_timeout_in_seconds": 30,
        "host": "$IRODS_HOSTNAME",
        "host_access_control": {
            "access_entries": []
        },
        "host_resolution": {
            "host_entries": []
        },
        "log_level": {
            "agent": "info",
            "agent_factory": "info",
            "api": "info",
            "authentication": "info",
            "database": "info",
            "delay_server": "info",
            "genquery1": "info",
            "genquery2": "info",
            "legacy": "info",
            "microservice": "info",
            "network": "info",
            "resource": "info",
            "rule_engine": "info",
            "server": "info",
            "sql": "info"
        },
        "match_hash_policy": "compatible",
        "negotiation_key": "32_byte_server_negotiation_key__",
        "hostname_cache_shared_memory_name": "irods_hostname_cache",
        "dns_cache_shared_memory_name": "irods_dns_cache",
        "hostname_cache_shm_name": "irods_hostname_cache",
        "dns_cache_shm_name": "irods_dns_cache",
        "plugin_configuration": {
            "authentication": {},
            "database": {
                "technology": "postgres",
                "host": "$IRODS_DB_HOST",
                "name": "$IRODS_DB_NAME",
                "odbc_driver": "PostgreSQL ANSI",
                "password": "$IRODS_DB_PASSWORD",
                "port": $IRODS_DB_PORT,
                "username": "$IRODS_DB_USER"
            },
            "network": {},
            "resource": {},
            "rule_engines": [
                {
                    "instance_name": "irods_rule_engine_plugin-irods_rule_language-instance",
                    "plugin_name": "irods_rule_engine_plugin-irods_rule_language",
                    "plugin_specific_configuration": {
                        "re_data_variable_mapping_set": [
                            "core"
                        ],
                        "re_function_name_mapping_set": [
                            "core"
                        ],
                        "re_rulebase_set": [
                            "core"
                        ],
                        "regexes_for_supported_peps": [
                            "ac[^ ]*",
                            "msi[^ ]*",
                            "[^ ]*pep_[^ ]*_(pre|post|except|finally)"
                        ]
                    },
                    "shared_memory_instance": "irods_rule_language_rule_engine"
                },
                {
                    "instance_name": "irods_rule_engine_plugin-cpp_default_policy-instance",
                    "plugin_name": "irods_rule_engine_plugin-cpp_default_policy",
                    "plugin_specific_configuration": {}
                }
            ]
        },
        "rule_engine_namespaces": [
            ""
        ],
        "schema_name": "server_config",
        "schema_version": "v5",
        "server_port_range_end": 20199,
        "server_port_range_start": 20000,
        "zone_auth_scheme": "native",
        "zone_key": "TEMPORARY_ZONE_KEY",
        "zone_name": "$IRODS_ZONE",
        "zone_port": 1247,
        "zone_user": "$IRODS_ADMIN_USER"
    }
}
EOF

# Use Python-based setup script if available (Standard for 5.x)
if [ -f /var/lib/irods/irods_control.py ] || [ -f /var/lib/irods/scripts/setup_irods.py ] || [ -f /var/lib/irods/setup_irods.py ]; then
    echo "Running setup using iRODS scripts..."
    SETUP_PY=""
    if [ -f /var/lib/irods/scripts/setup_irods.py ]; then
        SETUP_PY="/var/lib/irods/scripts/setup_irods.py"
    elif [ -f /var/lib/irods/setup_irods.py ]; then
        SETUP_PY="/var/lib/irods/setup_irods.py"
    fi

    if [ -n "$SETUP_PY" ]; then
        echo "Using $SETUP_PY..."
        echo "Content of /tmp/irods_setup_answers.json:"
        cat /tmp/irods_setup_answers.json
        # The --json_configuration_file flag is the correct way to perform an unattended install in iRODS 5.x
        if python3 "$SETUP_PY" --json_configuration_file /tmp/irods_setup_answers.json > /tmp/setup_irods.log 2>&1; then
            echo "setup_irods.py succeeded"
        else
            echo "setup_irods.py failed. Logs:"
            cat /tmp/setup_irods.log
            # Attempt to start anyway if it was just the test that failed
            if grep -q "Post-install test failed" /tmp/setup_irods.log; then
                echo "Setup reported post-install test failure, but we will attempt to proceed..."
            else
                exit 1
            fi
        fi
    else
        echo "No setup_irods.py found, attempting legacy-style setup..."
        # Fallback to legacy if necessary, but 5.x should have setup_irods.py
        if [ -f /var/lib/irods/packaging/setup_irods.sh ]; then
            /var/lib/irods/packaging/setup_irods.sh < /tmp/irods_setup_answers.json || true
        else
            echo "ERROR: No setup script found!"
            exit 1
        fi
    fi
else
    echo "No iRODS scripts found, attempting legacy-style setup..."
    if [ -f /var/lib/irods/packaging/setup_irods.sh ]; then
        /var/lib/irods/packaging/setup_irods.sh < /tmp/irods_setup_answers.json || true
    else
        echo "ERROR: No setup script found!"
        exit 1
    fi
fi

  # setup_irods.py might have already started it, but we ensure it's up
  # Make sure environment is in sync before starting/checking
  mkdir -p /root/.irods
  cp /var/lib/irods/.irods/irods_environment.json /root/.irods/irods_environment.json
  chmod 600 /root/.irods/irods_environment.json
  chmod 700 /root/.irods
  
  # Ensure /etc/irods/server_config.json has the correct hostname if it was mis-set
  if [ -f /etc/irods/server_config.json ]; then
      sed -i "s/$(hostname)/$IRODS_HOSTNAME/g" /etc/irods/server_config.json || true
  fi

  start_irods

  # Post-setup configurations
  # Ensure the directory exists
  mkdir -p /var/lib/irods/.irods
  # We overwrite irods_environment.json to ensure it uses $IRODS_HOSTNAME,
  # as setup_irods.py might have created it with the dynamic container ID.
  echo "Setting up irods user environment (initial)..."
  cat > /var/lib/irods/.irods/irods_environment.json <<EOF
{
    "irods_host": "$IRODS_HOSTNAME",
    "irods_port": 1247,
    "irods_user_name": "$IRODS_ADMIN_USER",
    "irods_zone_name": "$IRODS_ZONE",
    "irods_default_resource": "demoResc",
    "irods_client_server_policy": "CS_NEG_REFUSE",
    "irods_client_server_negotiation_key": "32_byte_server_negotiation_key__",
    "irods_encryption_algorithm": "AES-256-CBC",
    "irods_encryption_key_size": 32,
    "irods_encryption_num_hash_rounds": 16,
    "irods_encryption_salt_size": 8
}
EOF
  chown -R irods:irods /var/lib/irods/.irods
  # Ensure interactive shells for irods pick up the right environment
  echo 'export IRODS_ENVIRONMENT_FILE=/var/lib/irods/.irods/irods_environment.json' > /var/lib/irods/.bashrc
  echo 'export HOME=/var/lib/irods' >> /var/lib/irods/.bashrc
  chown irods:irods /var/lib/irods/.bashrc
  
  # Initialize authentication for irods user
  echo "Initializing iRODS session for irods user..."
  su - irods -c "echo '$IRODS_ADMIN_PASSWORD' | iinit" || true

  # Ensure root can use icommands too
  mkdir -p /root/.irods
  cp /var/lib/irods/.irods/irods_environment.json /root/.irods/irods_environment.json
  chmod 600 /root/.irods/irods_environment.json
  chmod 700 /root/.irods

  echo "Running testsetup-consortium.sh..."
  chmod +x /var/lib/irods/testsetup-consortium.sh
  # Check if testsetup already ran (e.g. if test1 user exists)
  if ! su - irods -c "iadmin lu test1" >/dev/null 2>&1; then
      su - irods -c "/var/lib/irods/testsetup-consortium.sh"
  else
      echo "testsetup-consortium.sh seems to have already run, skipping."
  fi

echo "iRODS 5.x initialized. Tailing logs..."
tail_logs
