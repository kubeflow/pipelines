FROM node:9.4.0 as build

WORKDIR ./src

COPY . .

RUN npm install
RUN ./node_modules/bower/bin/bower i --allow-root
RUN npm run build

COPY --from=build ./src/server /app/server
COPY --from=build ./src/dist /app/dist

WORKDIR /app

EXPOSE 3000
RUN npm install express
ENV API_SERVER_ADDRESS http://localhost:3001
CMD node ./server/server.js ./dist/ 3000
