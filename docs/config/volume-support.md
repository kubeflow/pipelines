# volume-support

> Alpha
>
> This kubeflow pipeline volume support has **alpha** status 

Now you can use volume and volume mount to support [tensorboard viewer and other visualize results](https://www.kubeflow.org/docs/pipelines/sdk/output-viewer). following steps is the detail information to do this.


## deploy ml-pipeline-ui to support tensorboard viewer
You can write your tensorboard logs to any volume, and view it from pipeline UI. 
To support this you should:

1. create config map for tensorboard viewer
```bash
kubectl apply -f - <<EOF
apiVersion: v1
data:
  view-spec-template.json: |-
    {
      "spec": {
          "containers": [{
            "volumeMounts": [{
              "name": " your_volume_name",
              "mountPath": "/your/mount/path"
            }]
          }],
          "volumes": [{
            "name": " your_volume_name",
            "persistentVolumeClaim": {
              "claimName": "your_pvc_name"
            }
          }]
        }
    }
kind: ConfigMap
metadata:
  name: ui-view-spec-template-configmap
  namespace: kubeflow
EOF
```
The purpose of this patch is to specify where to find your tensorboard log data. 


2. re-deploy `ml-pipeline-ui` deployment (in KFP multi user mode it may be named `ml-pipeline-ui-artifact`) 
```bash
kubectl patch deployment ml-pipeline-ui -nkubeflow --type strategic --patch '
spec:
  template:
    spec:
      containers:
      - env:
        - name: VIEWER_TENSORBOARD_POD_TEMPLATE_SPEC_PATH
          value: "/config/view_spec_template.json"
        name: ml-pipeline-ui
        volumeMounts:
          - mountPath: /config/view_spec_template.json
            subPath: view_spec_template.json
            name: view-template-config
      volumes:
        - name: view-template-config
          configMap:
            name: ui-view-spec-template-configmap
            items:
              - key: view-spec-template.json
                path: view_spec_template.json
'
```
The purpose of this patch is to let `ml-pipeline-ui` know where to find your tensorboard viewer config.

After you do this patch, you can write you tensorboard log dir at any point during the pipeline execution into your volume `your_volume_name`, and writing out metadata for the output viewers: 
```bash
metadata = {
    'outputs' : [{
      'type': 'tensorboard',
      'source': volume://your_volume_name/your/tensorboard/logdir,
    }]
}
with open('/mlpipeline-ui-metadata.json', 'w') as f:
   json.dump(metadata, f)
```
**Note:** `source` must be specified with volume schema which starts with `volume://` and following with `your_volume_name`, and source path `your/tensorboard/logdir` is the path in volume. 


Above example mount volume `your_volume_name` to` /your/mount/path`, so the finally path in viewer pod is `/your/mount/path/your/tensorboard/logdir`


## deploy ml-pipeline-ui to support other visualize results
`ml-pipeline-ui` deployment (in KFP multi user mode it may be named `ml-pipeline-ui-artifact`) not configured with visualize volume in default. If you want to visualize `Confusion matrix`, `Markdown`, `ROC curve`, `Table` or `Web app` result in pipeline UI, you can do this with below patch code:
```bash
kubectl patch deployment ml-pipeline-ui -nkubeflow --type strategic --patch '
spec:
  template:
    spec:
      containers:
        name: ml-pipeline-ui
        volumeMounts:
          - mountPath: /your/mount/path
            name: your_volume_name
      volumes:
        - name: your_volume_name
          persistentVolumeClaim:
                claimName: your_pvc_name,
                readOnly: true
'
```
The purpose of this patch is to add volumes `your_volume_name` and volume mounts `/your/mount/path` for visualize results to be read. After you deploy this patch, you can write your visualize data at any pipeline components into volume `your_volume_name` and you may find visualize result in pipeline UI.


**Example:**
```bash
 metadata = {
    'outputs' : [{
      'type': 'confusion_matrix',
      'format': 'csv',
      'schema': [
        {'name': 'target', 'type': 'CATEGORY'},
        {'name': 'predicted', 'type': 'CATEGORY'},
        {'name': 'count', 'type': 'NUMBER'},
      ],
      'source': volume://your_volume_name/your/confusion/matrix/path,
      # Convert vocab to string because for bealean values we want "True|False" to match csv data.
      'labels': list(map(str, vocab)),
    }]
  }
  with file_io.FileIO('/mlpipeline-ui-metadata.json', 'w') as f:
    json.dump(metadata, f)
```
**Note:** `source` must be specified with volume schema which starts with `volume://` and following with `your_volume_name`, and source path `your/confusion/matrix/path` is the path in volume. 


Above example mount volume `your_volume_name` to` /your/mount/path`, so the finally path is `/your/mount/path/your/confusion/matrix/path`

You can also re-deploy `ml-pipeline-ui` with one patch to support tensorboard and other visualize results:
```bash
kubectl patch deployment ml-pipeline-ui -nkubeflow --type strategic --patch '
spec:
  template:
    spec:
      containers:
      - env:
        - name: VIEWER_TENSORBOARD_POD_TEMPLATE_SPEC_PATH
          value: "/config/view_spec_template.json"
        name: ml-pipeline-ui
        volumeMounts:
          - mountPath: /config/view_spec_template.json
            subPath: view_spec_template.json
            name: view-template-config
          - mountPath: /your/mount/path
            name: your_volume_name
      volumes:
        - name: view-template-config
          configMap:
            name: ui-view-spec-template-configmap
            items:
              - key: view-spec-template.json
                path: view_spec_template.json
        - name: your_volume_name
          persistentVolumeClaim:
                claimName: your_pvc_name,
                readOnly: true
'
```


## support subpath volume mount
If you config subpath under volume mounts, you can use volume schema `volume://your_volume_name/your_volume_mount_sub_path/your/data/path` to distinguish different subpath volume mounts. For example, you may patch your `ml-pipeline-ui` with below code to support other visualize results with subpath volume mount:
```bash
kubectl patch deployment ml-pipeline-ui -nkubeflow --type strategic --patch '
spec:
  template:
    spec:
      containers:
        name: ml-pipeline-ui
        volumeMounts:
          - mountPath: /your/mount/path
            subPath: your_volume_mount_sub_path
            name: your_volume_name
      volumes:
        - name: your_volume_name
          persistentVolumeClaim:
                claimName: your_pvc_name,
                readOnly: true
'
```
and you can writing out metadata for the output viewers: 
```bash
 metadata = {
    'outputs' : [{
      'type': 'confusion_matrix',
      'format': 'csv',
      'schema': [
        {'name': 'target', 'type': 'CATEGORY'},
        {'name': 'predicted', 'type': 'CATEGORY'},
        {'name': 'count', 'type': 'NUMBER'},
      ],
      'source': volume://your_volume_name/your_volume_mount_sub_path/your/confusion/matrix/path,
      # Convert vocab to string because for bealean values we want "True|False" to match csv data.
      'labels': list(map(str, vocab)),
    }]
  }
  with file_io.FileIO('/mlpipeline-ui-metadata.json', 'w') as f:
    json.dump(metadata, f)
```
because volume mount configured with subpath, so `your_volume_name/your_volume_mount_sub_path` will be concat to match volume schema `volume://your_volume_name/your_volume_mount_sub_path/your/confusion/matrix/path`. It matched, and the finally path is `/your/mount/path/your/confusion/matrix/path`

