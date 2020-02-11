# six>=1.12.0 is required by virtualenv
pip install -U six>=1.12.0 tox virtualenv
tox "$@"
