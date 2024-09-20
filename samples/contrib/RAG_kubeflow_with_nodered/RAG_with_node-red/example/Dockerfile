FROM nodered/node-red:2.2.3-12

ARG PREFIX

ENV NODE_OPTIONS=--max_old_space_size=128

USER root
RUN apk update && \
  apk add py3-pip python3-dev libffi-dev

USER root
RUN apk update && \
  apk add py3-pip
RUN apk update && apk add python3-dev
USER node-red

# Copy package.json to the WORKDIR so npm builds all
# of your added nodes modules for Node-RED
COPY package.json .
RUN npm install --unsafe-perm --no-update-notifier --no-fund --only=production

ADD scripts/entrypoint.sh .

ENTRYPOINT ["./entrypoint.sh"]