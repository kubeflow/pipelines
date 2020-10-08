# Copyright 2019 kubeflow.org.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#    http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

from flask import Flask, request, abort
from flask_cors import CORS
import json
import os

from kfservingdeployer import deploy_model

app = Flask(__name__)
CORS(app)


@app.route("/deploy-model", methods=["POST"])
def deploy_model_post():
    if not request.json:
        abort(400)
    return json.dumps(
        deploy_model(
            action=request.json["action"],
            model_name=request.json["model_name"],
            default_model_uri=request.json["default_model_uri"],
            canary_model_uri=request.json["canary_model_uri"],
            canary_model_traffic=request.json["canary_model_traffic"],
            namespace=request.json["namespace"],
            framework=request.json["framework"],
            default_custom_model_spec=request.json["default_custom_model_spec"],
            canary_custom_model_spec=request.json["canary_custom_model_spec"],
            autoscaling_target=request.json["autoscaling_target"],
            service_account=request.json["service_account"],
            enable_istio_sidecar=request.json["enable_istio_sidecar"],
            inferenceservice_yaml=request.json["inferenceservice_yaml"]
        )
    )


@app.route("/", methods=["GET"])
def root_get():
    return 200


@app.route("/", methods=["OPTIONS"])
def root_options():
    return "200"


if __name__ == "__main__":
    app.run(debug=True, host="0.0.0.0", port=int(os.environ.get("PORT", 8080)))
