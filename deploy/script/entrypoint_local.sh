#!/bin/bash

: "${AIRFLOW__CORE__EXECUTOR:=${EXECUTOR:-Local}Executor}"
export AIRFLOW__CORE__EXECUTOR

: "${AIRFLOW__CORE__FERNET_KEY:="hXIXeOXGoR8xowEKylmY6MnE7EkRgj6VikXjhxwwNUo="}"
export AIRFLOW__CORE__FERNET_KEY

: "${POSTGRES_USER:="airflow"}"
export POSTGRES_USER

: "${POSTGRES_PASSWORD:="airflow"}"
export POSTGRES_PASSWORD

: "${POSTGRES_DB:="airflow"}"
export POSTGRES_DB

: "${POSTGRES_PORT:=5432}"
export POSTGRES_PORT

: "${POSTGRES_HOST:="localhost"}"
export POSTGRES_HOST

: "${AIRFLOW__CORE__SQL_ALCHEMY_CONN:="postgresql+psycopg2://$POSTGRES_USER:$POSTGRES_PASSWORD@$POSTGRES_HOST:$POSTGRES_PORT/$POSTGRES_DB"}"
export AIRFLOW__CORE__SQL_ALCHEMY_CONN

case "$1" in
  local)
    echo "$(date) - starting webserver..."
    airflow initdb
    airflow scheduler &
    exec airflow webserver
    ;;
  version)
    exec airflow "$@"
    ;;
  *)
    # Run commands with the correct environment.
    echo "$(date) - executing arbitrary command..."
    exec "$@"
    ;;
esac
