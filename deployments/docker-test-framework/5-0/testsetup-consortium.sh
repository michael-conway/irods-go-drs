#!/usr/bin/env bash
# Test setup script for iRODS 5.x
# Should be run inside the container as the irods user or using sudo -u irods

# Wait for server
for i in {1..10}; do
    if iadmin lr > /dev/null 2>&1; then
        break
    fi
    echo "Waiting for iRODS..."
    sleep 2
done

iadmin mkuser test1 rodsadmin
iadmin moduser test1 password test
iadmin aua test1 test1DN

iadmin mkuser test2 rodsuser
iadmin moduser test2 password test

iadmin mkuser test3 rodsuser
iadmin moduser test3 password test

# resources
mkdir -p /var/lib/irods/iRODS/Vault1 /var/lib/irods/iRODS/Vault2 /var/lib/irods/iRODS/Vault3
iadmin mkresc test1-resc "unixfilesystem"  $(hostname):/var/lib/irods/iRODS/Vault1
iadmin mkresc test1-resc2 "unixfilesystem"  $(hostname):/var/lib/irods/iRODS/Vault2
iadmin mkresc test1-resc3 "unixfilesystem"  $(hostname):/var/lib/irods/iRODS/Vault3

iadmin mkuser anonymous rodsuser
iadmin atg public anonymous

iadmin mkgroup jargonTestUg
iadmin atg jargonTestUg test1
iadmin atg jargonTestUg test3

echo "Test setup complete."
