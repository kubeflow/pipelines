# Copyright 2021 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from kfp import compiler
from kfp import dsl


@dsl.container_component
def chicago_taxi_trips_dataset(
        Table: dsl.Output[dsl.Artifact],
        where:
    str = 'trip_start_timestamp>="1900-01-01" AND trip_start_timestamp<"2100-01-01"',
        limit: int = 1000,
        select:
    str = 'trip_id,taxi_id,trip_start_timestamp,trip_end_timestamp,trip_seconds,trip_miles,pickup_census_tract,dropoff_census_tract,pickup_community_area,dropoff_community_area,fare,tips,tolls,extras,trip_total,payment_type,company,pickup_centroid_latitude,pickup_centroid_longitude,pickup_centroid_location,dropoff_centroid_latitude,dropoff_centroid_longitude,dropoff_centroid_location',
        format: str = 'csv'):
    """City of Chicago Taxi Trips dataset: https://data.cityofchicago.org/Transportation/Taxi-Trips/wrvz-psew

    The input parameters configure the SQL query to the database.
    The dataset is pretty big, so limit the number of results using the `limit` or `where` parameters.
    Read [Socrata dev](https://dev.socrata.com/docs/queries/) for the advanced query syntax

    Args:
        limit: Number of rows to return. The rows are randomly sampled.
        format: Output data format. Suports csv,tsv,cml,rdf,json
        Table: Result type depends on format. CSV and TSV have header."""
    return dsl.ContainerSpec(
        image='byrnedo/alpine-curl@sha256:548379d0a4a0c08b9e55d9d87a592b7d35d9ab3037f4936f5ccd09d0b625a342',
        command=[
            'sh', '-c',
            'set -e -x -o pipefail\noutput_path="$0"\nselect="$1"\nwhere="$2"\nlimit="$3"\nformat="$4"\nmkdir -p "$(dirname "$output_path")"\ncurl --get \'https://data.cityofchicago.org/resource/wrvz-psew.\'"${format}" \\\n    --data-urlencode \'$limit=\'"${limit}" \\\n    --data-urlencode \'$where=\'"${where}" \\\n    --data-urlencode \'$select=\'"${select}" \\\n    | tr -d \'"\' > "$output_path"  # Removing unneeded quotes around all numbers\n',
            Table.path, select, where, limit, format
        ],
    )


chicago_taxi_dataset_op = chicago_taxi_trips_dataset


@dsl.container_component
def convert_csv_to_apache_parquet(data: dsl.Input[dsl.Artifact],
                                  output_data: dsl.Output[dsl.Artifact]):
    """Converts CSV table to Apache Parquet.

        [Apache Parquet](https://parquet.apache.org/)

        Annotations:
            author: Alexey Volkov <alexey.volkov@ark-kun.com>"""
    return dsl.ContainerSpec(
        image='python:3.7',
        command=[
            'sh', '-c',
            '(PIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet --no-warn-script-location \'pyarrow==0.17.1\' || PIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet --no-warn-script-location \'pyarrow==0.17.1\' --user) && "$0" "$@"',
            'python3', '-u', '-c',
            'def _make_parent_dirs_and_return_path(file_path: str):\n    import os\n    os.makedirs(os.path.dirname(file_path), exist_ok=True)\n    return file_path\n\ndef convert_csv_to_apache_parquet(\n    data_path,\n    output_data_path,\n):\n    \'\'\'Converts CSV table to Apache Parquet.\n\n    [Apache Parquet](https://parquet.apache.org/)\n\n    Annotations:\n        author: Alexey Volkov <alexey.volkov@ark-kun.com>\n    \'\'\'\n    from pyarrow import csv, parquet\n\n    table = csv.read_csv(data_path)\n    parquet.write_table(table, output_data_path)\n\nimport argparse\n_parser = argparse.ArgumentParser(prog=\'Convert csv to apache parquet\', description=\'Converts CSV table to Apache Parquet.\\n\\n    [Apache Parquet](https://parquet.apache.org/)\\n\\n    Annotations:\\n        author: Alexey Volkov <alexey.volkov@ark-kun.com>\')\n_parser.add_argument("--data", dest="data_path", type=str, required=True, default=argparse.SUPPRESS)\n_parser.add_argument("--output-data", dest="output_data_path", type=_make_parent_dirs_and_return_path, required=True, default=argparse.SUPPRESS)\n_parsed_args = vars(_parser.parse_args())\n_output_files = _parsed_args.pop("_output_paths", [])\n\n_outputs = convert_csv_to_apache_parquet(**_parsed_args)\n\n_output_serializers = [\n\n]\n\nimport os\nfor idx, output_file in enumerate(_output_files):\n    try:\n        os.makedirs(os.path.dirname(output_file))\n    except OSError:\n        pass\n    with open(output_file, \'w\') as f:\n        f.write(_output_serializers[idx](_outputs[idx]))\n'
        ],
        args=['--data', data.path, '--output-data', output_data.path],
    )


convert_csv_to_apache_parquet_op = convert_csv_to_apache_parquet


@dsl.container_component
def xgboost_train_csv(training_data: dsl.Input[dsl.Artifact],
                      model: dsl.Output[dsl.Artifact],
                      model_config: dsl.Output[dsl.Artifact],
                      starting_model: dsl.Input[dsl.Artifact] = None,
                      label_column: int = 0,
                      num_iterations: int = 10,
                      booster_params: dict = None,
                      objective: str = 'reg:squarederror',
                      booster: str = 'gbtree',
                      learning_rate: float = 0.3,
                      min_split_loss: float = 0,
                      max_depth: int = 6):
    """Train an XGBoost model.

        Args:
            training_data_path: Path for the training data in CSV format.
            model_path: Output path for the trained model in binary XGBoost format.
            model_config_path: Output path for the internal parameter configuration of Booster as a JSON string.
            starting_model_path: Path for the existing trained model to start from.
            label_column: Column containing the label data.
            num_boost_rounds: Number of boosting iterations.
            booster_params: Parameters for the booster. See https://xgboost.readthedocs.io/en/latest/parameter.html
            objective: The learning task and the corresponding learning objective.
                See https://xgboost.readthedocs.io/en/latest/parameter.html#learning-task-parameters
                The most common values are:
                "reg:squarederror" - Regression with squared loss (default).
                "reg:logistic" - Logistic regression.
                "binary:logistic" - Logistic regression for binary classification, output probability.
                "binary:logitraw" - Logistic regression for binary classification, output score before logistic transformation
                "rank:pairwise" - Use LambdaMART to perform pairwise ranking where the pairwise loss is minimized
                "rank:ndcg" - Use LambdaMART to perform list-wise ranking where Normalized Discounted Cumulative Gain (NDCG) is maximized

        Annotations:
            author: Alexey Volkov <alexey.volkov@ark-kun.com>"""
    return dsl.ContainerSpec(
        image='python:3.7',
        command=[
            'sh', '-c',
            '(PIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet --no-warn-script-location \'xgboost==1.1.1\' \'pandas==1.0.5\' || PIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet --no-warn-script-location \'xgboost==1.1.1\' \'pandas==1.0.5\' --user) && "$0" "$@"',
            'python3', '-u', '-c',
            'def _make_parent_dirs_and_return_path(file_path: str):\n    import os\n    os.makedirs(os.path.dirname(file_path), exist_ok=True)\n    return file_path\n\ndef xgboost_train(\n    training_data_path,  # Also supports LibSVM\n    model_path,\n    model_config_path,\n    starting_model_path = None,\n\n    label_column = 0,\n    num_iterations = 10,\n    booster_params = None,\n\n    # Booster parameters\n    objective = \'reg:squarederror\',\n    booster = \'gbtree\',\n    learning_rate = 0.3,\n    min_split_loss = 0,\n    max_depth = 6,\n):\n    \'\'\'Train an XGBoost model.\n\n    Args:\n        training_data_path: Path for the training data in CSV format.\n        model_path: Output path for the trained model in binary XGBoost format.\n        model_config_path: Output path for the internal parameter configuration of Booster as a JSON string.\n        starting_model_path: Path for the existing trained model to start from.\n        label_column: Column containing the label data.\n        num_boost_rounds: Number of boosting iterations.\n        booster_params: Parameters for the booster. See https://xgboost.readthedocs.io/en/latest/parameter.html\n        objective: The learning task and the corresponding learning objective.\n            See https://xgboost.readthedocs.io/en/latest/parameter.html#learning-task-parameters\n            The most common values are:\n            "reg:squarederror" - Regression with squared loss (default).\n            "reg:logistic" - Logistic regression.\n            "binary:logistic" - Logistic regression for binary classification, output probability.\n            "binary:logitraw" - Logistic regression for binary classification, output score before logistic transformation\n            "rank:pairwise" - Use LambdaMART to perform pairwise ranking where the pairwise loss is minimized\n            "rank:ndcg" - Use LambdaMART to perform list-wise ranking where Normalized Discounted Cumulative Gain (NDCG) is maximized\n\n    Annotations:\n        author: Alexey Volkov <alexey.volkov@ark-kun.com>\n    \'\'\'\n    import pandas\n    import xgboost\n\n    df = pandas.read_csv(\n        training_data_path,\n    )\n\n    training_data = xgboost.DMatrix(\n        data=df.drop(columns=[df.columns[label_column]]),\n        label=df[df.columns[label_column]],\n    )\n\n    booster_params = booster_params or {}\n    booster_params.setdefault(\'objective\', objective)\n    booster_params.setdefault(\'booster\', booster)\n    booster_params.setdefault(\'learning_rate\', learning_rate)\n    booster_params.setdefault(\'min_split_loss\', min_split_loss)\n    booster_params.setdefault(\'max_depth\', max_depth)\n\n    starting_model = None\n    if starting_model_path:\n        starting_model = xgboost.Booster(model_file=starting_model_path)\n\n    model = xgboost.train(\n        params=booster_params,\n        dtrain=training_data,\n        num_boost_round=num_iterations,\n        xgb_model=starting_model\n    )\n\n    # Saving the model in binary format\n    model.save_model(model_path)\n\n    model_config_str = model.save_config()\n    with open(model_config_path, \'w\') as model_config_file:\n        model_config_file.write(model_config_str)\n\nimport json\nimport argparse\n_parser = argparse.ArgumentParser(prog=\'Xgboost train\', description=\'Train an XGBoost model.\\n\\n    Args:\\n        training_data_path: Path for the training data in CSV format.\\n        model_path: Output path for the trained model in binary XGBoost format.\\n        model_config_path: Output path for the internal parameter configuration of Booster as a JSON string.\\n        starting_model_path: Path for the existing trained model to start from.\\n        label_column: Column containing the label data.\\n        num_boost_rounds: Number of boosting iterations.\\n        booster_params: Parameters for the booster. See https://xgboost.readthedocs.io/en/latest/parameter.html\\n        objective: The learning task and the corresponding learning objective.\\n            See https://xgboost.readthedocs.io/en/latest/parameter.html#learning-task-parameters\\n            The most common values are:\\n            "reg:squarederror" - Regression with squared loss (default).\\n            "reg:logistic" - Logistic regression.\\n            "binary:logistic" - Logistic regression for binary classification, output probability.\\n            "binary:logitraw" - Logistic regression for binary classification, output score before logistic transformation\\n            "rank:pairwise" - Use LambdaMART to perform pairwise ranking where the pairwise loss is minimized\\n            "rank:ndcg" - Use LambdaMART to perform list-wise ranking where Normalized Discounted Cumulative Gain (NDCG) is maximized\\n\\n    Annotations:\\n        author: Alexey Volkov <alexey.volkov@ark-kun.com>\')\n_parser.add_argument("--training-data", dest="training_data_path", type=str, required=True, default=argparse.SUPPRESS)\n_parser.add_argument("--starting-model", dest="starting_model_path", type=str, required=False, default=argparse.SUPPRESS)\n_parser.add_argument("--label-column", dest="label_column", type=int, required=False, default=argparse.SUPPRESS)\n_parser.add_argument("--num-iterations", dest="num_iterations", type=int, required=False, default=argparse.SUPPRESS)\n_parser.add_argument("--booster-params", dest="booster_params", type=json.loads, required=False, default=argparse.SUPPRESS)\n_parser.add_argument("--objective", dest="objective", type=str, required=False, default=argparse.SUPPRESS)\n_parser.add_argument("--booster", dest="booster", type=str, required=False, default=argparse.SUPPRESS)\n_parser.add_argument("--learning-rate", dest="learning_rate", type=float, required=False, default=argparse.SUPPRESS)\n_parser.add_argument("--min-split-loss", dest="min_split_loss", type=float, required=False, default=argparse.SUPPRESS)\n_parser.add_argument("--max-depth", dest="max_depth", type=int, required=False, default=argparse.SUPPRESS)\n_parser.add_argument("--model", dest="model_path", type=_make_parent_dirs_and_return_path, required=True, default=argparse.SUPPRESS)\n_parser.add_argument("--model-config", dest="model_config_path", type=_make_parent_dirs_and_return_path, required=True, default=argparse.SUPPRESS)\n_parsed_args = vars(_parser.parse_args())\n\n_outputs = xgboost_train(**_parsed_args)\n'
        ],
        args=[
            '--training-data', training_data.path,
            dsl.IfPresentPlaceholder(
                input_name='starting_model',
                then=['--starting-model', starting_model.path]),
            dsl.IfPresentPlaceholder(
                input_name='label_column',
                then=['--label-column', label_column]),
            dsl.IfPresentPlaceholder(
                input_name='num_iterations',
                then=['--num-iterations', num_iterations]),
            dsl.IfPresentPlaceholder(
                input_name='booster_params',
                then=['--booster-params', booster_params]),
            dsl.IfPresentPlaceholder(
                input_name='objective', then=['--objective', objective]),
            dsl.IfPresentPlaceholder(
                input_name='booster', then=['--booster', booster]),
            dsl.IfPresentPlaceholder(
                input_name='learning_rate',
                then=['--learning-rate', learning_rate]),
            dsl.IfPresentPlaceholder(
                input_name='min_split_loss',
                then=['--min-split-loss', min_split_loss]),
            dsl.IfPresentPlaceholder(
                input_name='max_depth', then=['--max-depth', max_depth]),
            '--model', model.path, '--model-config', model_config.path
        ],
    )


xgboost_train_on_csv_op = xgboost_train_csv


@dsl.container_component
def xgboost_predict_csv(data: dsl.Input[dsl.Artifact],
                        model: dsl.Input[dsl.Artifact],
                        predictions: dsl.Output[dsl.Artifact],
                        label_column: int = None):
    """Make predictions using a trained XGBoost model.

        Args:
            data_path: Path for the feature data in CSV format.
            model_path: Path for the trained model in binary XGBoost format.
            predictions_path: Output path for the predictions.
            label_column: Column containing the label data.

        Annotations:
            author: Alexey Volkov <alexey.volkov@ark-kun.com>"""
    return dsl.ContainerSpec(
        image='python:3.7',
        command=[
            'sh', '-c',
            '(PIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet --no-warn-script-location \'xgboost==1.1.1\' \'pandas==1.0.5\' || PIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet --no-warn-script-location \'xgboost==1.1.1\' \'pandas==1.0.5\' --user) && "$0" "$@"',
            'python3', '-u', '-c',
            'def _make_parent_dirs_and_return_path(file_path: str):\n    import os\n    os.makedirs(os.path.dirname(file_path), exist_ok=True)\n    return file_path\n\ndef xgboost_predict(\n    data_path,  # Also supports LibSVM\n    model_path,\n    predictions_path,\n    label_column = None,\n):\n    \'\'\'Make predictions using a trained XGBoost model.\n\n    Args:\n        data_path: Path for the feature data in CSV format.\n        model_path: Path for the trained model in binary XGBoost format.\n        predictions_path: Output path for the predictions.\n        label_column: Column containing the label data.\n\n    Annotations:\n        author: Alexey Volkov <alexey.volkov@ark-kun.com>\n    \'\'\'\n    from pathlib import Path\n\n    import numpy\n    import pandas\n    import xgboost\n\n    df = pandas.read_csv(\n        data_path,\n    )\n\n    if label_column is not None:\n        df = df.drop(columns=[df.columns[label_column]])\n\n    testing_data = xgboost.DMatrix(\n        data=df,\n    )\n\n    model = xgboost.Booster(model_file=model_path)\n\n    predictions = model.predict(testing_data)\n\n    Path(predictions_path).parent.mkdir(parents=True, exist_ok=True)\n    numpy.savetxt(predictions_path, predictions)\n\nimport argparse\n_parser = argparse.ArgumentParser(prog=\'Xgboost predict\', description=\'Make predictions using a trained XGBoost model.\\n\\n    Args:\\n        data_path: Path for the feature data in CSV format.\\n        model_path: Path for the trained model in binary XGBoost format.\\n        predictions_path: Output path for the predictions.\\n        label_column: Column containing the label data.\\n\\n    Annotations:\\n        author: Alexey Volkov <alexey.volkov@ark-kun.com>\')\n_parser.add_argument("--data", dest="data_path", type=str, required=True, default=argparse.SUPPRESS)\n_parser.add_argument("--model", dest="model_path", type=str, required=True, default=argparse.SUPPRESS)\n_parser.add_argument("--label-column", dest="label_column", type=int, required=False, default=argparse.SUPPRESS)\n_parser.add_argument("--predictions", dest="predictions_path", type=_make_parent_dirs_and_return_path, required=True, default=argparse.SUPPRESS)\n_parsed_args = vars(_parser.parse_args())\n\n_outputs = xgboost_predict(**_parsed_args)\n'
        ],
        args=[
            '--data', data.path, '--model', model.path,
            dsl.IfPresentPlaceholder(
                input_name='label_column',
                then=['--label-column', label_column]), '--predictions',
            predictions.path
        ],
    )


xgboost_predict_on_csv_op = xgboost_predict_csv


@dsl.container_component
def xgboost_train(training_data: dsl.Input[dsl.Artifact],
                  label_column_name: str,
                  model: dsl.Output[dsl.Artifact],
                  model_config: dsl.Output[dsl.Artifact],
                  starting_model: dsl.Input[dsl.Artifact] = None,
                  num_iterations: int = 10,
                  booster_params: dict = None,
                  objective: str = 'reg:squarederror',
                  booster: str = 'gbtree',
                  learning_rate: float = 0.3,
                  min_split_loss: float = 0,
                  max_depth: int = 6):
    """Train an XGBoost model.

        Args:
            training_data_path: Path for the training data in Apache Parquet format.
            model_path: Output path for the trained model in binary XGBoost format.
            model_config_path: Output path for the internal parameter configuration of Booster as a JSON string.
            starting_model_path: Path for the existing trained model to start from.
            label_column_name: Name of the column containing the label data.
            num_boost_rounds: Number of boosting iterations.
            booster_params: Parameters for the booster. See https://xgboost.readthedocs.io/en/latest/parameter.html
            objective: The learning task and the corresponding learning objective.
                See https://xgboost.readthedocs.io/en/latest/parameter.html#learning-task-parameters
                The most common values are:
                "reg:squarederror" - Regression with squared loss (default).
                "reg:logistic" - Logistic regression.
                "binary:logistic" - Logistic regression for binary classification, output probability.
                "binary:logitraw" - Logistic regression for binary classification, output score before logistic transformation
                "rank:pairwise" - Use LambdaMART to perform pairwise ranking where the pairwise loss is minimized
                "rank:ndcg" - Use LambdaMART to perform list-wise ranking where Normalized Discounted Cumulative Gain (NDCG) is maximized

        Annotations:
            author: Alexey Volkov <alexey.volkov@ark-kun.com>"""
    return dsl.ContainerSpec(
        image='python:3.7',
        command=[
            'sh', '-c',
            '(PIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet --no-warn-script-location \'xgboost==1.1.1\' \'pandas==1.0.5\' \'pyarrow==0.17.1\' || PIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet --no-warn-script-location \'xgboost==1.1.1\' \'pandas==1.0.5\' \'pyarrow==0.17.1\' --user) && "$0" "$@"',
            'python3', '-u', '-c',
            'def _make_parent_dirs_and_return_path(file_path: str):\n    import os\n    os.makedirs(os.path.dirname(file_path), exist_ok=True)\n    return file_path\n\ndef xgboost_train(\n    training_data_path,\n    model_path,\n    model_config_path,\n    label_column_name,\n\n    starting_model_path = None,\n\n    num_iterations = 10,\n    booster_params = None,\n\n    # Booster parameters\n    objective = \'reg:squarederror\',\n    booster = \'gbtree\',\n    learning_rate = 0.3,\n    min_split_loss = 0,\n    max_depth = 6,\n):\n    \'\'\'Train an XGBoost model.\n\n    Args:\n        training_data_path: Path for the training data in Apache Parquet format.\n        model_path: Output path for the trained model in binary XGBoost format.\n        model_config_path: Output path for the internal parameter configuration of Booster as a JSON string.\n        starting_model_path: Path for the existing trained model to start from.\n        label_column_name: Name of the column containing the label data.\n        num_boost_rounds: Number of boosting iterations.\n        booster_params: Parameters for the booster. See https://xgboost.readthedocs.io/en/latest/parameter.html\n        objective: The learning task and the corresponding learning objective.\n            See https://xgboost.readthedocs.io/en/latest/parameter.html#learning-task-parameters\n            The most common values are:\n            "reg:squarederror" - Regression with squared loss (default).\n            "reg:logistic" - Logistic regression.\n            "binary:logistic" - Logistic regression for binary classification, output probability.\n            "binary:logitraw" - Logistic regression for binary classification, output score before logistic transformation\n            "rank:pairwise" - Use LambdaMART to perform pairwise ranking where the pairwise loss is minimized\n            "rank:ndcg" - Use LambdaMART to perform list-wise ranking where Normalized Discounted Cumulative Gain (NDCG) is maximized\n\n    Annotations:\n        author: Alexey Volkov <alexey.volkov@ark-kun.com>\n    \'\'\'\n    import pandas\n    import xgboost\n\n    # Loading data\n    df = pandas.read_parquet(training_data_path)\n    training_data = xgboost.DMatrix(\n        data=df.drop(columns=[label_column_name]),\n        label=df[[label_column_name]],\n    )\n    # Training\n    booster_params = booster_params or {}\n    booster_params.setdefault(\'objective\', objective)\n    booster_params.setdefault(\'booster\', booster)\n    booster_params.setdefault(\'learning_rate\', learning_rate)\n    booster_params.setdefault(\'min_split_loss\', min_split_loss)\n    booster_params.setdefault(\'max_depth\', max_depth)\n\n    starting_model = None\n    if starting_model_path:\n        starting_model = xgboost.Booster(model_file=starting_model_path)\n\n    model = xgboost.train(\n        params=booster_params,\n        dtrain=training_data,\n        num_boost_round=num_iterations,\n        xgb_model=starting_model\n    )\n\n    # Saving the model in binary format\n    model.save_model(model_path)\n\n    model_config_str = model.save_config()\n    with open(model_config_path, \'w\') as model_config_file:\n        model_config_file.write(model_config_str)\n\nimport json\nimport argparse\n_parser = argparse.ArgumentParser(prog=\'Xgboost train\', description=\'Train an XGBoost model.\\n\\n    Args:\\n        training_data_path: Path for the training data in Apache Parquet format.\\n        model_path: Output path for the trained model in binary XGBoost format.\\n        model_config_path: Output path for the internal parameter configuration of Booster as a JSON string.\\n        starting_model_path: Path for the existing trained model to start from.\\n        label_column_name: Name of the column containing the label data.\\n        num_boost_rounds: Number of boosting iterations.\\n        booster_params: Parameters for the booster. See https://xgboost.readthedocs.io/en/latest/parameter.html\\n        objective: The learning task and the corresponding learning objective.\\n            See https://xgboost.readthedocs.io/en/latest/parameter.html#learning-task-parameters\\n            The most common values are:\\n            "reg:squarederror" - Regression with squared loss (default).\\n            "reg:logistic" - Logistic regression.\\n            "binary:logistic" - Logistic regression for binary classification, output probability.\\n            "binary:logitraw" - Logistic regression for binary classification, output score before logistic transformation\\n            "rank:pairwise" - Use LambdaMART to perform pairwise ranking where the pairwise loss is minimized\\n            "rank:ndcg" - Use LambdaMART to perform list-wise ranking where Normalized Discounted Cumulative Gain (NDCG) is maximized\\n\\n    Annotations:\\n        author: Alexey Volkov <alexey.volkov@ark-kun.com>\')\n_parser.add_argument("--training-data", dest="training_data_path", type=str, required=True, default=argparse.SUPPRESS)\n_parser.add_argument("--label-column-name", dest="label_column_name", type=str, required=True, default=argparse.SUPPRESS)\n_parser.add_argument("--starting-model", dest="starting_model_path", type=str, required=False, default=argparse.SUPPRESS)\n_parser.add_argument("--num-iterations", dest="num_iterations", type=int, required=False, default=argparse.SUPPRESS)\n_parser.add_argument("--booster-params", dest="booster_params", type=json.loads, required=False, default=argparse.SUPPRESS)\n_parser.add_argument("--objective", dest="objective", type=str, required=False, default=argparse.SUPPRESS)\n_parser.add_argument("--booster", dest="booster", type=str, required=False, default=argparse.SUPPRESS)\n_parser.add_argument("--learning-rate", dest="learning_rate", type=float, required=False, default=argparse.SUPPRESS)\n_parser.add_argument("--min-split-loss", dest="min_split_loss", type=float, required=False, default=argparse.SUPPRESS)\n_parser.add_argument("--max-depth", dest="max_depth", type=int, required=False, default=argparse.SUPPRESS)\n_parser.add_argument("--model", dest="model_path", type=_make_parent_dirs_and_return_path, required=True, default=argparse.SUPPRESS)\n_parser.add_argument("--model-config", dest="model_config_path", type=_make_parent_dirs_and_return_path, required=True, default=argparse.SUPPRESS)\n_parsed_args = vars(_parser.parse_args())\n\n_outputs = xgboost_train(**_parsed_args)\n'
        ],
        args=[
            '--training-data', training_data.path, '--label-column-name',
            label_column_name,
            dsl.IfPresentPlaceholder(
                input_name='starting_model',
                then=['--starting-model', starting_model.path]),
            dsl.IfPresentPlaceholder(
                input_name='num_iterations',
                then=['--num-iterations', num_iterations]),
            dsl.IfPresentPlaceholder(
                input_name='booster_params',
                then=['--booster-params', booster_params]),
            dsl.IfPresentPlaceholder(
                input_name='objective', then=['--objective', objective]),
            dsl.IfPresentPlaceholder(
                input_name='booster', then=['--booster', booster]),
            dsl.IfPresentPlaceholder(
                input_name='learning_rate',
                then=['--learning-rate', learning_rate]),
            dsl.IfPresentPlaceholder(
                input_name='min_split_loss',
                then=['--min-split-loss', min_split_loss]),
            dsl.IfPresentPlaceholder(
                input_name='max_depth', then=['--max-depth', max_depth]),
            '--model', model.path, '--model-config', model_config.path
        ],
    )


xgboost_train_on_parquet_op = xgboost_train


@dsl.container_component
def xgboost_predict(data: dsl.Input[dsl.Artifact],
                    model: dsl.Input[dsl.Artifact],
                    predictions: dsl.Output[dsl.Artifact],
                    label_column_name: str = None):
    """Make predictions using a trained XGBoost model.

        Args:
            data_path: Path for the feature data in Apache Parquet format.
            model_path: Path for the trained model in binary XGBoost format.
            predictions_path: Output path for the predictions.
            label_column_name: Optional. Name of the column containing the label data that is excluded during the prediction.

        Annotations:
            author: Alexey Volkov <alexey.volkov@ark-kun.com>"""
    return dsl.ContainerSpec(
        image='python:3.7',
        command=[
            'sh', '-c',
            '(PIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet --no-warn-script-location \'xgboost==1.1.1\' \'pandas==1.0.5\' \'pyarrow==0.17.1\' || PIP_DISABLE_PIP_VERSION_CHECK=1 python3 -m pip install --quiet --no-warn-script-location \'xgboost==1.1.1\' \'pandas==1.0.5\' \'pyarrow==0.17.1\' --user) && "$0" "$@"',
            'python3', '-u', '-c',
            'def _make_parent_dirs_and_return_path(file_path: str):\n    import os\n    os.makedirs(os.path.dirname(file_path), exist_ok=True)\n    return file_path\n\ndef xgboost_predict(\n    data_path,\n    model_path,\n    predictions_path,\n    label_column_name = None,\n):\n    \'\'\'Make predictions using a trained XGBoost model.\n\n    Args:\n        data_path: Path for the feature data in Apache Parquet format.\n        model_path: Path for the trained model in binary XGBoost format.\n        predictions_path: Output path for the predictions.\n        label_column_name: Optional. Name of the column containing the label data that is excluded during the prediction.\n\n    Annotations:\n        author: Alexey Volkov <alexey.volkov@ark-kun.com>\n    \'\'\'\n    from pathlib import Path\n\n    import numpy\n    import pandas\n    import xgboost\n\n    # Loading data\n    df = pandas.read_parquet(data_path)\n    if label_column_name:\n        df = df.drop(columns=[label_column_name])\n\n    evaluation_data = xgboost.DMatrix(\n        data=df,\n    )\n\n    # Training\n    model = xgboost.Booster(model_file=model_path)\n\n    predictions = model.predict(evaluation_data)\n\n    Path(predictions_path).parent.mkdir(parents=True, exist_ok=True)\n    numpy.savetxt(predictions_path, predictions)\n\nimport argparse\n_parser = argparse.ArgumentParser(prog=\'Xgboost predict\', description=\'Make predictions using a trained XGBoost model.\\n\\n    Args:\\n        data_path: Path for the feature data in Apache Parquet format.\\n        model_path: Path for the trained model in binary XGBoost format.\\n        predictions_path: Output path for the predictions.\\n        label_column_name: Optional. Name of the column containing the label data that is excluded during the prediction.\\n\\n    Annotations:\\n        author: Alexey Volkov <alexey.volkov@ark-kun.com>\')\n_parser.add_argument("--data", dest="data_path", type=str, required=True, default=argparse.SUPPRESS)\n_parser.add_argument("--model", dest="model_path", type=str, required=True, default=argparse.SUPPRESS)\n_parser.add_argument("--label-column-name", dest="label_column_name", type=str, required=False, default=argparse.SUPPRESS)\n_parser.add_argument("--predictions", dest="predictions_path", type=_make_parent_dirs_and_return_path, required=True, default=argparse.SUPPRESS)\n_parsed_args = vars(_parser.parse_args())\n\n_outputs = xgboost_predict(**_parsed_args)\n'
        ],
        args=[
            '--data', data.path, '--model', model.path,
            dsl.IfPresentPlaceholder(
                input_name='label_column_name',
                then=['--label-column-name', label_column_name]),
            '--predictions', predictions.path
        ],
    )


xgboost_predict_on_parquet_op = xgboost_predict


@dsl.pipeline(name='xgboost-sample-pipeline')
def xgboost_pipeline():
    training_data_csv = chicago_taxi_dataset_op(
        where='trip_start_timestamp >= "2019-01-01" AND trip_start_timestamp < "2019-02-01"',
        select='tips,trip_seconds,trip_miles,pickup_community_area,dropoff_community_area,fare,tolls,extras,trip_total',
        limit=10000,
    ).output

    # Training and prediction on dataset in CSV format
    model_trained_on_csv = xgboost_train_on_csv_op(
        training_data=training_data_csv,
        label_column=0,
        objective='reg:squarederror',
        num_iterations=200,
    ).outputs['model']

    xgboost_predict_on_csv_op(
        data=training_data_csv,
        model=model_trained_on_csv,
        label_column=0,
    )

    # Training and prediction on dataset in Apache Parquet format
    training_data_parquet = convert_csv_to_apache_parquet_op(
        data=training_data_csv).output

    model_trained_on_parquet = xgboost_train_on_parquet_op(
        training_data=training_data_parquet,
        label_column_name='tips',
        objective='reg:squarederror',
        num_iterations=200,
    ).outputs['model']

    xgboost_predict_on_parquet_op(
        data=training_data_parquet,
        model=model_trained_on_parquet,
        label_column_name='tips',
    )

    # Checking cross-format predictions
    xgboost_predict_on_parquet_op(
        data=training_data_parquet,
        model=model_trained_on_csv,
        label_column_name='tips',
    )

    xgboost_predict_on_csv_op(
        data=training_data_csv,
        model=model_trained_on_parquet,
        label_column=0,
    )


if __name__ == '__main__':
    compiler.Compiler().compile(
        pipeline_func=xgboost_pipeline,
        package_path=__file__.replace('.py', '.yaml'))
