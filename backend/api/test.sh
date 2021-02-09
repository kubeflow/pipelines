set -ex
rm -rf go_client go_http_client
docker build ../.. -t builder -f ../Dockerfile.buildapi
./generate_api.sh
