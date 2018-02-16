# ML Pipeline Artifacts


Files in this directory include detailed definitions of artifacts. Artifacts
are inputs and outputs of pipeline tasks. ML Tasks always materialize results,
so these artifacts are the interface between ML pipeline tasks.

Each artifact class comes with "static" validation. For example if storage is
"gs", source needs to start with "gs://". An instance of one artifact class
usually represent data that have not been produced, so each task may also do
"run time" validaiton, such as existence of files, content of files, etc.
