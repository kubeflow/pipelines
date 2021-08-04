#!/bin/bash

if (( $# != 2 ))
then
    echo "Usage: ./build.sh <path-to-example> <dockerhub-username>"
    echo "Ex: ./build.sh pytorch_pipeline/examples/cifar10 foobar"
    exit 1
fi


## Generating current timestamp
python3 gen_image_timestamp.py > curr_time.txt

export images_tag=$(cat curr_time.txt)
echo ++++ Building component images with tag=$images_tag


full_image_name=$2/pytorch_kfp_components:$images_tag

echo IMAGE TO BUILD: $full_image_name

export full_image_name=$full_image_name


## build and push docker - to fetch the latest changes and install dependencies
# cd pytorch_kfp_components

docker build --no-cache -t $full_image_name .
docker push $full_image_name

# cd ..

python utils/generate_templates.py $1/template_mapping.json

## Update component.yaml files with the latest docker image name

find "yaml" -name "*.yaml" | grep -v 'deploy' | grep -v "tensorboard"  | grep -v "prediction" | while read -d $'\n' file
do
    yq -i eval ".implementation.container.image =  \"$full_image_name\"" $file
done


## compile pipeline

echo Running pipeline compilation
echo "$1/pipeline.py"
python3 "$1/pipeline.py" --target kfp
