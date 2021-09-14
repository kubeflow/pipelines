#!/bin/bash -e

CONDA_ENV="${1:-naas-tests}"

is_conda_env() {
  check_env=$(conda env list | grep "${CONDA_ENV}\s")
  if [[ -n "$check_env" ]] ; then
    echo 1
  fi
}

is_conda_env_active() {
  check_env_active=$(conda env list | grep "${CONDA_ENV}\s*\*")
  if [[ -n "$check_env_active" ]] ; then
    echo 1
  fi
}

if [[ ! $(is_conda_env) ]]; then
  conda create -y -n "${CONDA_ENV}" --no-default-packages
  conda activate "${CONDA_ENV}"
  conda install -y pip
fi

if [[ ! $(is_conda_env_active) ]]; then
  conda activate "${CONDA_ENV}"
fi

pip install -r requirements-tests.txt
python -m pytest -vvv