ARG BASE_IMAGE=alpine
FROM node:24.14.0-slim AS build

ARG COMMIT_HASH
ENV COMMIT_HASH=${COMMIT_HASH}
ARG TAG_NAME
ENV TAG_NAME=${TAG_NAME}

ARG DATE

WORKDIR ./src

COPY . .

WORKDIR ./frontend

# Retry with cache clean as a workaround for an intermittent npm EEXIST/ENOENT
# race condition where parallel download workers collide on the same cache entry.
RUN npm ci && npm run postinstall || (npm cache clean --force && npm ci && npm run postinstall)
RUN npm run build

RUN mkdir -p ./server/dist && \
    echo ${COMMIT_HASH} > ./server/dist/COMMIT_HASH && \
    echo ${DATE} > ./server/dist/BUILD_DATE && \
    echo ${TAG_NAME} > ./server/dist/TAG_NAME

FROM node:24.14.0-${BASE_IMAGE}

COPY --from=build ./src/frontend/server /server
COPY --from=build ./src/frontend/build /client

WORKDIR /server

EXPOSE 3000
# Retry with cache clean as a workaround for an intermittent npm EEXIST/ENOENT
# race condition where parallel download workers collide on the same cache entry.
RUN npm ci || (npm cache clean --force && npm ci)
RUN npm run build
ENV API_SERVER_ADDRESS http://localhost:3001
CMD node dist/server.js ../client/ 3000
