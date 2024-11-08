#!/bin/bash

default_env_file=".env"

env_file="${1:-$default_env_file}"

if [ -f "$env_file" ]; then
    # we expect argument splitting here
    # shellcheck disable=SC2046
    export $(<"$env_file" xargs)
else
    echo "Warning: .env file not found at $env_file" >&2
fi

get_env_with_default() {
    local var_name=$1
    local default_value=$2
    local value
    value=$(printenv "$var_name")
    if [ -z "$value" ]; then
        echo "$default_value"
    else
        echo "$value"
    fi
}

logstash_url=$(get_env_with_default "LOGSTASH_URL" "http://localhost:9600")
port=$(get_env_with_default "PORT" "9198")
host=$(get_env_with_default "HOST" "")
log_level=$(get_env_with_default "LOG_LEVEL" "info")

cat << EOF
logstash:
  instances:
    - url: "$logstash_url"
server:
  host: "${host:-0.0.0.0}"
  port: $port
logging:
  level: "$log_level"
EOF
