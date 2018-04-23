FROM node:9.4.0 as build

WORKDIR ./src

COPY . .

RUN npm install && cd server && npm install
RUN ./node_modules/bower/bin/bower i --allow-root
RUN npm run build

FROM node:9.4.0-alpine

COPY --from=build ./src/server /app
COPY --from=build ./src/dist /app/dist

WORKDIR /app

EXPOSE 3000
RUN npm install
RUN npm run build
ENV API_SERVER_ADDRESS http://localhost:3001
CMD node dist/server.js ./dist/ 3000
