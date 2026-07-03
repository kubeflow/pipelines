## V2 mock pipelines specification

This folder is for KFP UI to mock backend response, the files here are copied from 
[test_data/sdk_compiled_pipelines/](../../../../../test_data/sdk_compiled_pipelines/). This folder doesn't need to be in sync with the compiled pipeline specs in that directory, but you can do it manually if there is a need to facilitate frontend development by using newer version of compiled pipeline definition.

Also, make sure to update the corresponding PipelineSpec files in [frontend/src/data/test](frontend/src/data/test). These files are used for unit testing.