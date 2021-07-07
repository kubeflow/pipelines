set -ex
export KF_PIPELINES_COMPILER_MODE=V2_COMPATIBLE
dsl-compile --py ../core/visualization/html.py --output tmp.yaml && kfp pipeline upload tmp.yaml -p v2compat-visualization-html
dsl-compile --py lightweight_python_functions_v2_with_outputs.py --output tmp.yaml && kfp pipeline upload tmp.yaml -p v2compat-lightweight-py-functions-with-outputs
dsl-compile --py lightweight_python_functions_v2_pipeline.py --output tmp.yaml && kfp pipeline upload tmp.yaml -p v2compat-lightweight-py-functions
dsl-compile --py metrics_visualization_v2.py --output tmp.yaml && kfp pipeline upload tmp.yaml -p v2compat-metrics-visualization

