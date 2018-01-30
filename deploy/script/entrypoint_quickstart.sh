#!/bin/bash

: "${AIRFLOW__CORE__EXECUTOR:=${EXECUTOR:-Sequential}Executor}"
export AIRFLOW__CORE__EXECUTOR

: "${AIRFLOW__CORE__SQL_ALCHEMY_CONN:="sqlite:////usr/local/airflow/airflow.db"}"
export AIRFLOW__CORE__SQL_ALCHEMY_CONN

: "${AIRFLOW__CORE__FERNET_KEY:="hXIXeOXGoR8xowEKylmY6MnE7EkRgj6VikXjhxwwNUo="}"
export AIRFLOW__CORE__FERNET_KEY

case "$1" in
  quickstart)
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
