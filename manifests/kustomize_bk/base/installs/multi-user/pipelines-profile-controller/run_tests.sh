# Build venv with required packages
VENV=".venv"
PYTHON_VENV="${VENV}/bin/python"
python -m venv $VENV
$PYTHON_VENV -m pip install -U pip
$PYTHON_VENV -m pip install -r requirements-dev.txt

# Run tests
$PYTHON_VENV -m pytest ./test_sync.py
