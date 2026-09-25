# Build venv with required packages
VENV=".venv"
PYTHON_VENV="${VENV}/bin/python"
python -m venv $VENV
$PYTHON_VENV -m pip install -U pip
$PYTHON_VENV -m pip install pytest requests botocore

# Run tests
AWS_EC2_METADATA_DISABLED=true $PYTHON_VENV -m pytest ./test_sync.py
