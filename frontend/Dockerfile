FROM node:9.4.0 as build

ARG COMMIT_HASH
ARG DATE

WORKDIR ./src

COPY . .

RUN npm install && cd server && npm install
RUN ./node_modules/bower/bin/bower i --allow-root
RUN npm run build

RUN mkdir -p ./server/dist && \
    echo ${COMMIT_HASH} > ./server/dist/COMMIT_HASH && \
    echo ${DATE} > ./server/dist/DATE

FROM node:9.4.0-alpine

COPY --from=build ./src/server /server
COPY --from=build ./src/dist /client

WORKDIR /server

EXPOSE 3000
RUN npm install
RUN npm run build
ENV API_SERVER_ADDRESS http://localhost:3001
CMD node dist/server.js ../client/ 3000
