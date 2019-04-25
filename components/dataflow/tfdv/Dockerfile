# Copyright 2018 The Kubeflow Authors
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
# http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

FROM ml-pipeline-dataflow-base

RUN apt-get update -y && \
    apt-get install --no-install-recommends -y -q build-essential && \
    pip install absl-py==0.3.0 \
                apache-beam==2.9.0 \
                astor==0.7.1 \
                avro==1.8.2 \
                backports.shutil-get-terminal-size==1.0.0 \
                backports.weakref==1.0.post1 \
                bleach==1.5.0 \
                cachetools==2.1.0 \
                certifi==2018.4.16 \
                chardet==3.0.4 \
                crcmod==1.7 \
                decorator==4.3.2 \
                dill==0.2.6 \
                docopt==0.6.2 \
                enum34==1.1.6 \
                fastavro==0.21.17 \
                fasteners==0.14.1 \
                funcsigs==1.0.2 \
                future==0.16.0 \
                futures==3.2.0 \
                gapic-google-cloud-pubsub-v1==0.15.4 \
                gast==0.2.0 \
                google-api-core==1.7.0 \
                google-apitools==0.5.24 \
                google-auth==1.5.1 \
                google-auth-httplib2==0.0.3 \
                google-cloud-bigquery==1.6.1 \
                google-cloud-core==0.29.1 \
                google-cloud-pubsub==0.35.4 \
                google-gax==0.15.16 \
                google-resumable-media==0.3.2 \
                googleapis-common-protos==1.5.3 \
                googledatastore==7.0.1 \
                grpc-google-iam-v1==0.11.4 \
                grpcio==1.14.0 \
                hdfs==2.1.0 \
                html5lib==0.9999999 \
                httplib2==0.9.2 \
                idna==2.7 \
                ipython==5.8.0 \
                ipython-genutils==0.2.0 \
                Markdown==2.6.11 \
                mock==2.0.0 \
                monotonic==1.5 \
                numpy==1.15.0 \
                oauth2client==3.0.0 \
                pandas==0.24.0 \
                pathlib2==2.3.3 \
                pbr==4.2.0 \
                pexpect==4.6.0 \
                pickleshare==0.7.5 \
                ply==3.8 \
                prompt-toolkit==1.0.15 \
                proto-google-cloud-datastore-v1==0.90.4 \
                proto-google-cloud-pubsub-v1==0.15.4 \
                protobuf==3.6.0 \
                ptyprocess==0.6.0 \
                pyasn1==0.4.4 \
                pyasn1-modules==0.2.2 \
                pydot==1.2.4 \
                Pygments==2.3.1 \
                pyparsing==2.3.1 \
                python-dateutil==2.7.5 \
                pytz==2018.4 \
                PyVCF==0.6.8 \
                PyYAML==3.12 \
                requests==2.19.1 \
                rsa==3.4.2 \
                scandir==1.9.0 \
                simplegeneric==0.8.1 \
                six==1.11.0 \
                tensorboard==1.9.0 \
                tensorflow==1.9.0 \
                tensorflow-data-validation==0.9.0 \
                tensorflow-metadata==0.9.0 \
                tensorflow-transform==0.9.0 \
                termcolor==1.1.0 \
                traitlets==4.3.2 \
                typing==3.6.4 \
                urllib3==1.23 \
                wcwidth==0.1.7 \
                Werkzeug==0.14.1 && \
    apt-get --purge autoremove -y build-essential

WORKDIR /ml

RUN mkdir /usr/licenses && \
    /ml/license.sh /ml/third_party_licenses.csv /usr/licenses

ENTRYPOINT ["python", "/ml/validate.py"]
