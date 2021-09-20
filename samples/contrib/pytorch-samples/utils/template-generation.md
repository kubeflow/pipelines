## PyTorch component.yaml templates

The templates for generating `component.yaml` are placed under `components/PyTorch/templates`.

[Link to component.yaml templates](../../../../components/PyTorch/templates)


## component.yaml generation

There are two different ways to generate `component.yaml` files for all the components.

### 1. Generate templates using utility

The following utility has been created to generate templates dynamically during runtime.
If there are no mapping scecified, the utility copies the templates into output directory (`yaml` folder)
If the mappings are specified, the utility replaces the key, value pairs and generates `component.yaml` into `yaml` folder.

Run the following command to generate the templates

`python utils/generate_templates.py cifar10/template_mapping.json`

Sample mapping file is shown as below

```
{
  "minio_component.yaml": {
    "implementation.container.image": "public.ecr.aws/pytorch-samples/kfp_samples:latest
  }

}
```

The above mentioned mapping will replace the image name in `minio_component.yaml`


### 2. Manually editing the templates

When there are more changes in the templates and new key value pairs has to be introduced, 
it would be easier to manually edit the templates placed under `components/PyTorch/templates`

Once the templates are manually edited, simply run the template generation script.

`python utils/generate_templates.py cifar10/template_mapping.json`
 
The script will copy over the edited templates to output folder name `yaml` under `pytorch-samples`
 
 