#!/usr/bin/env bash

pushd ../..

coverage run -m pytest tests/unit_tests/tests --junitxml ./unit_tests.log
coverage report -m --omit "*/usr/*,tests/*,*__init__*,*/Python/*"

popd
