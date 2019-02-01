
build:
	./build.sh

upload-to-s3:
	aws s3 cp kfp.tar.gz s3://${S3_BUCKET}/${S3_LOCATION}/
