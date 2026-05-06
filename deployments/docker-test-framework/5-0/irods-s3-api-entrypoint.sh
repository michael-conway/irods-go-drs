#!/usr/bin/env sh
set -eu

bucket_plugin="$(find /usr -name 'libirods_s3_api_plugin-bucket_mapping-local_file.so' | head -n 1)"
user_plugin="$(find /usr -name 'libirods_s3_api_plugin-user_mapping-local_file.so' | head -n 1)"

if [ -z "$bucket_plugin" ] || [ -z "$user_plugin" ]; then
  echo "Could not locate iRODS S3 API mapping plugins" >&2
  exit 1
fi

for mapping_file in \
  /shared-s3-config/irods-s3-bucket-mapping.json \
  /shared-s3-config/irods-s3-user-mapping.json
do
  if [ ! -f "$mapping_file" ]; then
    echo "Missing iRODS S3 API mapping file: $mapping_file" >&2
    exit 1
  fi
done

sed \
  -e "s#__BUCKET_MAPPING_PLUGIN__#$bucket_plugin#g" \
  -e "s#__USER_MAPPING_PLUGIN__#$user_plugin#g" \
  /config-template.json > /tmp/irods-s3-api-config.json

exec irods_s3_api /tmp/irods-s3-api-config.json
