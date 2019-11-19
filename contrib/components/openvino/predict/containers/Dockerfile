FROM ubuntu:16.04 as DEV
RUN apt-get update && apt-get install -y \
            curl \
            ca-certificates \
            python3-pip \
            python-dev \
            libgfortran3 \
            vim \
            build-essential \
            cmake \
            curl \
            wget \
            libssl-dev \
            ca-certificates \
            git \
            libboost-regex-dev \
            gcc-multilib \
            g++-multilib \
            libgtk2.0-dev \
            pkg-config \
            unzip \
            automake \
            libtool \
            autoconf \
            libpng12-dev \
            libcairo2-dev \
            libpango1.0-dev \
            libglib2.0-dev \
            libgtk2.0-dev \
            libswscale-dev \
            libavcodec-dev \
            libavformat-dev \
            libgstreamer1.0-0 \
            gstreamer1.0-plugins-base \
            libusb-1.0-0-dev \
            libopenblas-dev
ARG DLDT_DIR=/dldt-2018_R5
RUN git clone --depth=1 -b 2018_R5 https://github.com/opencv/dldt.git ${DLDT_DIR} && \
    cd ${DLDT_DIR} && git submodule init && git submodule update --recursive && \
    rm -Rf .git && rm -Rf model-optimizer

WORKDIR ${DLDT_DIR}
RUN curl -L -o ${DLDT_DIR}/mklml_lnx_2019.0.1.20180928.tgz https://github.com/intel/mkl-dnn/releases/download/v0.17.2/mklml_lnx_2019.0.1.20180928.tgz && \
    tar -xzf ${DLDT_DIR}/mklml_lnx_2019.0.1.20180928.tgz && rm ${DLDT_DIR}/mklml_lnx_2019.0.1.20180928.tgz
WORKDIR ${DLDT_DIR}/inference-engine
RUN mkdir build && cd build && cmake -DGEMM=MKL  -DMKLROOT=${DLDT_DIR}/mklml_lnx_2019.0.1.20180928 -DENABLE_MKL_DNN=ON  -DCMAKE_BUILD_TYPE=Release ..
RUN cd build && make -j4
RUN pip3 install cython numpy && mkdir ie_bridges/python/build && cd ie_bridges/python/build && \
    cmake -DInferenceEngine_DIR=${DLDT_DIR}/inference-engine/build -DPYTHON_EXECUTABLE=`which python3` -DPYTHON_LIBRARY=/usr/lib/x86_64-linux-gnu/libpython3.5m.so -DPYTHON_INCLUDE_DIR=/usr/include/python3.5m .. && \
    make -j4

FROM ubuntu:16.04 as PROD

RUN apt-get update && apt-get install -y --no-install-recommends \
            curl \
            ca-certificates \
            python3-pip \
            python3-dev \
            virtualenv \
            libgomp1

COPY --from=DEV /dldt-2018_R5/inference-engine/bin/intel64/Release/lib/*.so /usr/local/lib/
COPY --from=DEV /dldt-2018_R5/inference-engine/ie_bridges/python/bin/intel64/Release/python_api/python3.5/openvino/ /usr/local/lib/openvino/
COPY --from=DEV /dldt-2018_R5/mklml_lnx_2019.0.1.20180928/lib/lib*.so /usr/local/lib/
ENV LD_LIBRARY_PATH=/usr/local/lib
ENV PYTHONPATH=/usr/local/lib
COPY requirements.txt .
RUN pip3 install setuptools wheel
RUN pip3 install -r requirements.txt
COPY predict.py classes.py ./






