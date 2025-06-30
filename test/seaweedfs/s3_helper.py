#!/usr/bin/env python3
"""
S3 helper script for SeaweedFS namespace isolation testing.
Uses boto3 to perform S3 operations for security testing.
"""

import sys
import boto3
from botocore.exceptions import ClientError, NoCredentialsError
import argparse


def create_s3_client(access_key, secret_key, endpoint_url):
    """Create S3 client with given credentials."""
    return boto3.client(
        's3',
        aws_access_key_id=access_key,
        aws_secret_access_key=secret_key,
        endpoint_url=endpoint_url,
        region_name='us-east-1'  # Required but not used by SeaweedFS
    )


def upload_file(access_key, secret_key, endpoint_url, bucket, key, content):
    """Upload a file to S3."""
    try:
        s3_client = create_s3_client(access_key, secret_key, endpoint_url)
        s3_client.put_object(
            Bucket=bucket,
            Key=key,
            Body=content.encode('utf-8')
        )
        print(f"✓ Successfully uploaded file to s3://{bucket}/{key}")
        return True
    except Exception as e:
        print(f"✗ Failed to upload file: {e}")
        return False


def download_file(access_key, secret_key, endpoint_url, bucket, key):
    """Download a file from S3."""
    try:
        s3_client = create_s3_client(access_key, secret_key, endpoint_url)
        response = s3_client.get_object(Bucket=bucket, Key=key)
        content = response['Body'].read().decode('utf-8')
        print(f"✓ Successfully downloaded file from s3://{bucket}/{key}")
        print(f"File content: {content}")
        return True, content
    except ClientError as e:
        error_code = e.response['Error']['Code']
        if error_code in ['NoSuchKey', 'AccessDenied', 'Forbidden']:
            print(f"✓ Access denied as expected: {error_code}")
            return False, None
        else:
            print(f"✗ Unexpected error: {e}")
            return False, None
    except Exception as e:
        print(f"✗ Failed to download file: {e}")
        return False, None


def main():
    parser = argparse.ArgumentParser(description='S3 operations for SeaweedFS testing')
    parser.add_argument('operation', choices=['upload', 'download'], help='Operation to perform')
    parser.add_argument('--access-key', required=True, help='AWS access key')
    parser.add_argument('--secret-key', required=True, help='AWS secret key')
    parser.add_argument('--endpoint-url', required=True, help='S3 endpoint URL')
    parser.add_argument('--bucket', required=True, help='S3 bucket name')
    parser.add_argument('--key', required=True, help='S3 object key')
    parser.add_argument('--content', help='Content to upload (for upload operation)')
    
    args = parser.parse_args()
    
    if args.operation == 'upload':
        if not args.content:
            print("Error: --content is required for upload operation")
            sys.exit(1)
        success = upload_file(args.access_key, args.secret_key, args.endpoint_url, 
                             args.bucket, args.key, args.content)
        sys.exit(0 if success else 1)
    
    elif args.operation == 'download':
        success, content = download_file(args.access_key, args.secret_key, args.endpoint_url,
                                        args.bucket, args.key)
        # For security test: success=True means unauthorized access (bad)
        # success=False means access denied (good)
        if args.key.startswith('private-artifacts/') and '/' in args.key[18:]:
            # This is a cross-namespace access attempt
            sys.exit(1 if success else 0)
        else:
            sys.exit(0 if success else 1)


if __name__ == '__main__':
    main()
