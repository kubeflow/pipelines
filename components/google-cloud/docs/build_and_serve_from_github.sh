# run from google3/third_party/py/google_cloud_pipeline_components/docs
pushd ..
python setup.py bdist_wheel
WHEEL_FILE=$(find "dist/" -name "google_cloud_pipeline_components*.whl")
pip3 install --upgrade $WHEEL_FILE
popd
pip install -r requirements.txt
builddir=$(mktemp -d)
sphinx-build -M html "source" $builddir
pushd $builddir/html
python3 -m http.server
popd