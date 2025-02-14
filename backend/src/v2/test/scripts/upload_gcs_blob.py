from google.cloud import storage


# This function is mainly written for environments without gsutil.
def upload_blob(source: str, destination: str):
    """Uploads a file to the bucket."""
    # source = "local/path/to/file"
    # destination = "gs://your-bucket-name/storage-object-name"

    storage_client = storage.Client()
    blob = storage.Blob.from_string(destination, storage_client)

    blob.upload_from_filename(source)

    print(f"File {source} uploaded to destination {destination}")


if __name__ == '__main__':
    import fire
    fire.Fire(upload_blob)
