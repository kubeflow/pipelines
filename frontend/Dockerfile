FROM node:12.14.1 as build

ARG COMMIT_HASH
ARG DATE

WORKDIR ./src

COPY . .

WORKDIR ./frontend

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
