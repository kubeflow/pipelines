FROM node:9.4.0 as build

ARG COMMIT_HASH
ARG DATE

WORKDIR ./src

COPY . .

WORKDIR ./frontend

# Workaround for ppc64le since phantomjs does not support ppc64le
RUN if [ "$(uname -m)" = "ppc64le" ]; then \
        wget -O /tmp/phantomjs-2.1.1-linux-ppc64.tar.bz2 https://github.com/ibmsoe/phantomjs/releases/download/2.1.1/phantomjs-2.1.1-linux-ppc64.tar.bz2 \
        && tar xf /tmp/phantomjs-2.1.1-linux-ppc64.tar.bz2 -C /usr/local/ \
        && ln -s /usr/local/phantomjs-2.1.1-linux-ppc64/bin/phantomjs /usr/bin/phantomjs; \
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

FROM node:9.4.0-alpine

COPY --from=build ./src/frontend/server /server
COPY --from=build ./src/frontend/build /client

WORKDIR /server

EXPOSE 3000
RUN npm run build
ENV API_SERVER_ADDRESS http://localhost:3001
CMD node dist/server.js ../client/ 3000
