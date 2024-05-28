IMAGE=cculab509/webapp

# Build base TRTIS client image
git clone https://github.com/NVIDIA/tensorrt-inference-server.git

base=tensorrt-inference-server
git clone https://github.com/triton-inference-server/client.git $base/clientrepo
docker build -t base-trtis-client -f $base/Dockerfile.sdk $base
rm -rf $base

# Build & push webapp image
docker build -t $IMAGE .
docker push $IMAGE
