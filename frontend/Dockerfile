# syntax=docker/dockerfile:1
# Pass NODE_VERSION from frontend/.nvmrc when building this image.
ARG NODE_VERSION
ARG BASE_IMAGE=alpine
# The supported production tag uses the digest below. Any other requested tag
# uses its own base, including downstream Debian variants.
ARG RUNTIME_TAG=${NODE_VERSION}-${BASE_IMAGE}
ARG CUSTOM_TAG=${RUNTIME_TAG#24.22.1-alpine}
ARG CUSTOM_STAGE=${CUSTOM_TAG:+custom-runtime}

FROM node:${NODE_VERSION}-slim AS build

ARG COMMIT_HASH
ENV COMMIT_HASH=${COMMIT_HASH}
ARG TAG_NAME
ENV TAG_NAME=${TAG_NAME}

ARG DATE

WORKDIR ./src

# Install all three dependency trees before copying source so source-only changes
# reuse the dependency layer. The root postinstall installs the nested packages.
COPY frontend/package.json frontend/package-lock.json frontend/.npmrc ./frontend/
COPY frontend/server/package.json frontend/server/package-lock.json ./frontend/server/
COPY frontend/mock-backend/package.json frontend/mock-backend/package-lock.json ./frontend/mock-backend/

WORKDIR ./frontend

RUN npm install --global "$(node -p 'require("./package.json").packageManager')"

# Retry with cache clean as a workaround for an intermittent npm EEXIST/ENOENT
# race condition where parallel download workers collide on the same cache entry.
# npm ci runs the root postinstall script, which installs server and mock-backend deps.
RUN (npm ci || (npm cache clean --force && npm ci)) && \
    echo "Frontend build dependencies installed"

WORKDIR ..
COPY . .
WORKDIR ./frontend
RUN npm run build

RUN mkdir -p ./server/dist && \
    echo ${COMMIT_HASH} > ./server/dist/COMMIT_HASH && \
    echo ${DATE} > ./server/dist/BUILD_DATE && \
    echo ${TAG_NAME} > ./server/dist/TAG_NAME

FROM node:${NODE_VERSION}-${BASE_IMAGE} AS custom-runtime

FROM node:24.22.1-alpine@sha256:06d7adf3e0ef61a13f1d28682726366f97a15aa1888c934d3ecec86c33918b53 AS pinned-runtime
RUN apk add --no-cache 'libssl3>=3.5.9-r1' 'libcrypto3>=3.5.9-r1'

FROM ${CUSTOM_STAGE:-pinned-runtime}

RUN if command -v apk >/dev/null; then apk add --no-cache 'zlib>=1.4-r0'; \
    else apt-get update && apt-get install -y --no-install-recommends zlib1g \
        && dpkg --compare-versions "$(dpkg-query -W -f='${Version}' zlib1g)" ge '1:1.3.dfsg+really1.3.1-1+deb13u1' \
        && rm -rf /var/lib/apt/lists/*; fi

COPY --from=build ./src/frontend/package.json /tmp/frontend-package.json
RUN npm install --global "$(node -p 'require("/tmp/frontend-package.json").packageManager')" && \
    rm /tmp/frontend-package.json

COPY --from=build ./src/frontend/server /server
COPY --from=build ./src/frontend/build /client

WORKDIR /server

EXPOSE 3000
# Retry with cache clean as a workaround for an intermittent npm EEXIST/ENOENT
# race condition where parallel download workers collide on the same cache entry.
RUN (npm ci || (npm cache clean --force && npm ci)) && \
    echo "Frontend server dependencies installed"
RUN npm run build
ENV API_SERVER_ADDRESS http://localhost:3001
CMD node dist/server.js ../client/ 3000
