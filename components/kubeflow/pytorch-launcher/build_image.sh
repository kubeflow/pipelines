LAUNCHER_IMAGE_NAME_DEFAULT=kubeflow-pytorchjob-launcher

while getopts ":hr:t:i:" opt; do
  case "${opt}" in
    h) echo "-r: repo name (including gcr.io/, etc., if not in Docker Hub)"
       echo "-i: image name (default is $LAUNCHER_IMAGE_NAME_DEFAULT)"
       echo "-t: image tag (default is inferred from date/git)"
       exit
      ;;
    r) REPO_NAME=${OPTARG}
      ;;
    t) TAG_NAME=${OPTARG}
      ;;
    i) LAUNCHER_IMAGE_NAME=${OPTARG}
      ;;
    \? ) echo "Usage: cmd [-p] project [-t] tag [-i] image"
      exit
      ;;
  esac
done

# Apply defaults/interpret inputs
LAUNCHER_IMAGE_NAME=${LAUNCHER_IMAGE_NAME:-$LAUNCHER_IMAGE_NAME_DEFAULT}
TAG_NAME${TAG_NAME:-$(date +v%Y%m%d)-$(git describe --tags --always --dirty)-$(git diff | shasum -a256 | cut -c -6)}}

if [ -n "${REPO_NAME}" ]; then
  # Ensure ends with /
  if [[ "$REPO_NAME" != */ ]]; then
    REPO_NAME+=/
  fi
fi

FULL_NAME=${REPO_NAME}${LAUNCHER_IMAGE_NAME}:${TAG_NAME}

mkdir -p ./build
rsync -arvp ./src/ ./build/
rsync -arvp ../common/ ./build/

docker build -t ${FULL_NAME} .
docker push ${FULL_NAME}

rm -rf ./build
