#!/usr/bin/env python3
"""
S3 helper for SeaweedFS namespace-isolation testing.

Exit codes (uniform for all operations):
    0 = ALLOWED / succeeded
    1 = DENIED (AccessDenied/Forbidden/403) or failed (NoSuchKey/error)

The caller (namespace_isolation_test.sh) decides whether "allowed" or "denied"
is the desired outcome; this script never infers intent from the key/prefix.

For 'list' the request shape is controlled explicitly, because tenant isolation
depends on the s3:prefix and s3:delimiter policy conditions:
    --prefix "x/"           -> ListObjectsV2(Prefix="x/")   (s3:prefix present)
    --prefix ""             -> ListObjectsV2(Prefix="")      (s3:prefix absent)
    --no-prefix             -> ListObjectsV2()               (s3:prefix absent)
    --delimiter "/"         -> adds Delimiter="/"            (s3:delimiter present)
"""

import argparse
import sys

import boto3
from botocore.exceptions import ClientError

# HTTP 403 is included because HeadBucket has no response body, so botocore
# synthesizes the error code from the status line ("403", not "Forbidden").
DENIED = ("AccessDenied", "Forbidden", "403")


def s3_client(access_key, secret_key, endpoint_url):
    return boto3.client(
        "s3",
        aws_access_key_id=access_key,
        aws_secret_access_key=secret_key,
        endpoint_url=endpoint_url,
        region_name="us-east-1",  # required by boto3, unused by SeaweedFS
    )


def head_bucket(client, bucket):
    try:
        client.head_bucket(Bucket=bucket)
        print(f"ALLOWED: head-bucket {bucket}")
        return True
    except ClientError as error:
        code = error.response["Error"]["Code"]
        print(f"{'DENIED' if code in DENIED else 'ERROR'}: head-bucket {bucket} ({code})")
        return False


def upload(client, bucket, key, content):
    try:
        client.put_object(Bucket=bucket, Key=key, Body=content.encode("utf-8"))
        print(f"ALLOWED: uploaded {bucket}/{key}")
        return True
    except ClientError as error:
        code = error.response["Error"]["Code"]
        print(f"{'DENIED' if code in DENIED else 'ERROR'}: upload {bucket}/{key} ({code})")
        return False


def download(client, bucket, key):
    try:
        body = client.get_object(Bucket=bucket, Key=key)["Body"].read().decode("utf-8")
        print(f"ALLOWED: downloaded {bucket}/{key} ({body!r})")
        return True
    except ClientError as error:
        code = error.response["Error"]["Code"]
        denied = code in DENIED or code == "NoSuchKey"
        print(f"{'DENIED' if denied else 'ERROR'}: download {bucket}/{key} ({code})")
        return False


def copy(client, bucket, key, source_bucket, source_key):
    try:
        client.copy_object(
            Bucket=bucket,
            Key=key,
            CopySource={"Bucket": source_bucket, "Key": source_key},
        )
        print(f"ALLOWED: copied {source_bucket}/{source_key} -> {bucket}/{key}")
        return True
    except ClientError as error:
        code = error.response["Error"]["Code"]
        denied = code in DENIED or code == "NoSuchKey"
        print(f"{'DENIED' if denied else 'ERROR'}: copy {source_bucket}/{source_key} -> {bucket}/{key} ({code})")
        return False


def delete(client, bucket, key):
    try:
        client.delete_object(Bucket=bucket, Key=key)
        print(f"ALLOWED: deleted {bucket}/{key}")
        return True
    except ClientError as error:
        code = error.response["Error"]["Code"]
        print(f"{'DENIED' if code in DENIED else 'ERROR'}: delete {bucket}/{key} ({code})")
        return False


def list_prefix(client, bucket, prefix, delimiter):
    # prefix is None -> omit Prefix entirely (s3:prefix absent)
    # prefix is ""   -> send Prefix="" (s3:prefix still absent at policy layer)
    # prefix is "x/" -> send Prefix="x/"
    # delimiter is None -> omit Delimiter; otherwise send Delimiter=<value>
    label = "<no prefix>" if prefix is None else f"'{prefix}'"
    if delimiter is not None:
        label += f" delimiter='{delimiter}'"
    kwargs = {"Bucket": bucket}
    if prefix is not None:
        kwargs["Prefix"] = prefix
    if delimiter is not None:
        kwargs["Delimiter"] = delimiter
    try:
        response = client.list_objects_v2(**kwargs)
        keys = [item["Key"] for item in response.get("Contents", [])]
        prefixes = [cp["Prefix"] for cp in response.get("CommonPrefixes", [])]
        print(f"ALLOWED: listed {bucket}/{label} (keys={keys}, prefixes={prefixes})")
        return True
    except ClientError as error:
        code = error.response["Error"]["Code"]
        print(f"{'DENIED' if code in DENIED else 'ERROR'}: list {bucket}/{label} ({code})")
        return False


def main():
    parser = argparse.ArgumentParser(description="S3 ops for SeaweedFS isolation tests")
    parser.add_argument("operation", choices=["headbucket", "upload", "download", "list", "copy", "delete"])
    parser.add_argument("--access-key", required=True)
    parser.add_argument("--secret-key", required=True)
    parser.add_argument("--endpoint-url", required=True)
    parser.add_argument("--bucket", required=True)
    parser.add_argument("--key", help="object key (upload/download/copy destination/delete)")
    parser.add_argument("--source-key", help="source object key (copy)")
    parser.add_argument("--prefix", help="key prefix for list (omit with --no-prefix)")
    parser.add_argument("--no-prefix", action="store_true",
                        help="send ListObjectsV2 with NO Prefix arg at all")
    parser.add_argument("--delimiter", help="delimiter for list (e.g. '/')")
    parser.add_argument("--content", help="content to upload (upload)")
    args = parser.parse_args()

    client = s3_client(args.access_key, args.secret_key, args.endpoint_url)

    if args.operation == "headbucket":
        allowed = head_bucket(client, args.bucket)
    elif args.operation == "upload":
        allowed = upload(client, args.bucket, args.key, args.content)
    elif args.operation == "download":
        allowed = download(client, args.bucket, args.key)
    elif args.operation == "copy":
        allowed = copy(client, args.bucket, args.key, args.bucket, args.source_key)
    elif args.operation == "delete":
        allowed = delete(client, args.bucket, args.key)
    else:  # list
        list_prefix_arg = None if args.no_prefix else (args.prefix if args.prefix is not None else "")
        allowed = list_prefix(client, args.bucket, list_prefix_arg, args.delimiter)

    sys.exit(0 if allowed else 1)


if __name__ == "__main__":
    main()
