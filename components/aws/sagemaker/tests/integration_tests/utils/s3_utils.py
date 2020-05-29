def check_object_exists(client, bucket, key):
    waiter = client.get_waiter("object_exists")
    waiter.wait(Bucket=bucket, Key=key)
    return True
