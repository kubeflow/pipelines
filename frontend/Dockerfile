FROM node:22.13 as build

ARG COMMIT_HASH
ENV COMMIT_HASH=${COMMIT_HASH}
ARG TAG_NAME
ENV TAG_NAME=${TAG_NAME}

ARG DATE

WORKDIR ./src

COPY . .

WORKDIR ./frontend

RUN npm ci && npm run postinstall
RUN npm run build

RUN mkdir -p ./server/dist && \
    echo ${COMMIT_HASH} > ./server/dist/COMMIT_HASH && \
    echo ${DATE} > ./server/dist/BUILD_DATE && \
    echo ${TAG_NAME} > ./server/dist/TAG_NAME

# Generate the dependency licenses files (one for the UI and one for the webserver),
# concatenate them to one file under ./src/server
RUN ./scripts/yarn-licenses.sh

FROM node:22.13-alpine

COPY --from=build ./src/frontend/server /server
COPY --from=build ./src/frontend/build /client

WORKDIR /server

EXPOSE 3000
RUN npm ci
RUN npm run build
ENV API_SERVER_ADDRESS http://localhost:3001
CMD node dist/server.js ../client/ 3000
