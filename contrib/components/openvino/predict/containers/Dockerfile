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
RUN curl -L -o 2018_R3.tar.gz https://github.com/opencv/dldt/archive/2018_R3.tar.gz && \
    tar -zxf 2018_R3.tar.gz && \
    rm 2018_R3.tar.gz && \
    rm -Rf dldt-2018_R3/model-optimizer
WORKDIR dldt-2018_R3/inference-engine
RUN mkdir build && cd build && cmake -DGEMM=MKL -DENABLE_MKL_DNN=ON  -DCMAKE_BUILD_TYPE=Release .. 
RUN cd build && make -j4
RUN pip3 install cython numpy && mkdir ie_bridges/python/build && cd ie_bridges/python/build && \
    cmake -DInferenceEngine_DIR=/dldt-2018_R3/inference-engine/build -DPYTHON_EXECUTABLE=`which python3` -DPYTHON_LIBRARY=/usr/lib/x86_64-linux-gnu/libpython3.5m.so -DPYTHON_INCLUDE_DIR=/usr/include/python3.5m .. && \
    make -j4


FROM ubuntu:16.04 as PROD
WORKDIR /predict

RUN apt-get update && apt-get install -y \
            curl \
            ca-certificates \
            python3-pip \
            python3-dev \
            vim \
            virtualenv

COPY --from=DEV /dldt-2018_R3/inference-engine/bin/intel64/Release/lib/*.so /usr/local/lib/
COPY --from=DEV /dldt-2018_R3/inference-engine/ie_bridges/python/build/ /usr/local/lib/
COPY --from=DEV /dldt-2018_R3/inference-engine/temp/mkltiny_lnx_20180511/lib/libiomp5.so /usr/local/lib/
ENV LD_LIBRARY_PATH=/usr/local/lib
ENV PYTHONPATH=/usr/local/lib
COPY requirements.txt .
RUN pip3 install -r requirements.txt
COPY predict.py .






