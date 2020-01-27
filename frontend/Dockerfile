FROM node:12.14.1 as build

ARG COMMIT_HASH
ARG DATE

WORKDIR ./src

COPY . .

WORKDIR ./frontend

# Workaround for ppc64le since phantomjs does not support ppc64le
RUN if [ "$(uname -m)" = "ppc64le" ]; then \
        wget --no-verbose -O /tmp/phantomjs-2.1.1-linux-ppc64.tar.bz2 https://github.com/ibmsoe/phantomjs/releases/download/2.1.1/phantomjs-2.1.1-linux-ppc64.tar.bz2 \
        && tar xf /tmp/phantomjs-2.1.1-linux-ppc64.tar.bz2 -C /usr/local/ \
        && ln -s /usr/local/phantomjs-2.1.1-linux-ppc64/bin/phantomjs /usr/bin/phantomjs; \
    fi

# Workaround for aarch64 since phantomjs does not support aarch64
RUN if [ "$(uname -m)" = "aarch64" ]; then \
        wget --no-verbose -O /tmp/phantomjs_2.1.1_arm64.tgz https://github.com/fg2it/phantomjs-on-raspberry/releases/download/v2.1.1-jessie-stretch-arm64/phantomjs_2.1.1_arm64.tgz \
        && tar zxvf /tmp/phantomjs_2.1.1_arm64.tgz -C /usr/local/bin \
        && cp /usr/local/bin/phantomjs /usr/bin/phantomjs \
        && chmod +x /usr/bin/phantomjs; \
    fi

RUN npm install && npm run postinstall
RUN npm run build

RUN mkdir -p ./server/dist && \
    echo ${COMMIT_HASH} > ./server/dist/COMMIT_HASH && \
    echo ${DATE} > ./server/dist/BUILD_DATE

# Generate the dependency licenses files (one for the UI and one for the webserver),
# concatenate them to one file under ./src/server
RUN npm i -D license-checker
RUN node gen_licenses . && node gen_licenses server && \
    cat dependency-licenses.txt >> server/dependency-licenses.txt

FROM node:12.14.1-alpine

COPY --from=build ./src/frontend/server /server
COPY --from=build ./src/frontend/build /client

WORKDIR /server

EXPOSE 3000
RUN npm run build
ENV API_SERVER_ADDRESS http://localhost:3001
CMD node dist/server.js ../client/ 3000
