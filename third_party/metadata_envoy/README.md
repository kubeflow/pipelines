## Metadata-Envoy

Folder to host build script to build metadata-envoy image. metadata-envoy image
is just a wrapper image on top of the OSS [envoy image](https://github.com/envoyproxy/envoy)
that contains license files for all the third party libraries used in envoy.

# Adding new library dependency

To add license file for a new library introduce in OSS envoy just update [dependencies.json](./dependencies.json)
file with the entry for the library and its corresponding license url path and
the run the python script:

```bash
python dependency_helper.py dependencies.json
```

# Building image

To build the image run:

```bash
docker build -t <image-tag> .
```