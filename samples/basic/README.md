## Compile
Follow [README.md](https://github.com/googleprivate/ml/blob/master/samples/README.md) to install the compiler and 
compile your sample python into workflow yaml.

## Deploy
The directory includes a pre-compiled pipeline (sequential.yaml).  
Open the ML pipeline UI.  
The sequential example expects one parameter: 

```
url: gs://[YOUR_GCS_BUCKET]
```

For example, you can use gs://ml-pipeline/shakespeare1.txt.
