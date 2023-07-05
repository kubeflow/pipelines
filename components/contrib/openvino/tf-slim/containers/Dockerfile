FROM intelpython/intelpython3_core as BUILD

RUN apt-get update && apt-get install -y --no-install-recommends \
        openjdk-8-jdk \
        openjdk-8-jre-headless \
        build-essential \
        curl \
        git \
        libcurl3-dev \
        libfreetype6-dev \
        libhdf5-serial-dev \
        libpng-dev \
        libzmq3-dev \
        pkg-config \
        rsync \
        software-properties-common \
        unzip \
        zip \
        zlib1g-dev && \
        apt-get clean

RUN git clone --depth 1 https://github.com/tensorflow/tensorflow


RUN conda create --name myenv -y
ENV PATH /opt/conda/envs/myenv/bin:$PATH

# Set up Bazel.


# Running bazel inside a `docker build` command causes trouble, cf:
#   https://github.com/bazelbuild/bazel/issues/134
# The easiest solution is to set up a bazelrc file forcing --batch.
RUN echo "startup --batch" >>/etc/bazel.bazelrc
# Similarly, we need to workaround sandboxing issues:
#   https://github.com/bazelbuild/bazel/issues/418
RUN echo "build --spawn_strategy=standalone --genrule_strategy=standalone" \
    >>/etc/bazel.bazelrc
# Install the most recent bazel release.
ENV BAZEL_VERSION 0.19.2
WORKDIR /
RUN mkdir /bazel && \
    cd /bazel && \
    curl -H "User-Agent: Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/57.0.2987.133 Safari/537.36" -fSsL -O https://github.com/bazelbuild/bazel/releases/download/$BAZEL_VERSION/bazel-$BAZEL_VERSION-installer-linux-x86_64.sh && \
    curl -H "User-Agent: Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/57.0.2987.133 Safari/537.36" -fSsL -o /bazel/LICENSE.txt https://raw.githubusercontent.com/bazelbuild/bazel/master/LICENSE && \
    chmod +x bazel-*.sh && \
    ./bazel-$BAZEL_VERSION-installer-linux-x86_64.sh && \
    cd / && \
    rm -f /bazel/bazel-$BAZEL_VERSION-installer-linux-x86_64.sh

RUN cd tensorflow && bazel build tensorflow/tools/graph_transforms:summarize_graph

FROM intelpython/intelpython3_core as PROD
RUN apt-get update && apt-get install -y --no-install-recommends \
        git && \
        apt-get clean

WORKDIR /slim

RUN git clone --depth 1 https://github.com/tensorflow/models && rm -Rf models/.git && \
    git clone --depth 1 https://github.com/tensorflow/tensorflow && rm -Rf tensorflow/.git

RUN conda create --name myenv -y
ENV PATH /opt/conda/envs/myenv/bin:$PATH

RUN pip install --no-cache-dir tensorflow validators google-cloud-storage
ENV PYTHONPATH=models/research/slim:tensorflow/python/tools

COPY --from=BUILD /tensorflow/bazel-bin/tensorflow/tools/graph_transforms/summarize_graph summarize_graph
COPY --from=BUILD /root/.cache/bazel/_bazel_root/*/execroot/org_tensorflow/bazel-out/k8-opt/bin/_solib_k8/_U_S_Stensorflow_Stools_Sgraph_Utransforms_Csummarize_Ugraph___Utensorflow/libtensorflow_framework.so libtensorflow_framework.so
COPY slim_model.py .



