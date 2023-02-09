rm -rf .venv
python -m venv .venv
source .venv/bin/activate
pip install cloudpickle
pip install numpy
python do_pickle.py
# # pip uninstall -y numpy
# python unpickle.py
# deactivate
# rm -rf .venv