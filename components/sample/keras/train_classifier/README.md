## Train classification model using Keras ##

Usage:

~~~~
#Load the component
train_op = comp.load_component(url='https://raw.githubusercontent.com/Ark-kun/pipelines/Added-sample-component/components/sample/keras/train_classifier/component.yaml')

#Use the component as part of the pipeline
def pipeline():
    train_task = train_op(
        training_set_features_path=os.path.join(testdata_root, 'training_set_features.tsv'),
        training_set_labels_path=os.path.join(testdata_root, 'training_set_labels.tsv'),
        output_model_uri=os.path.join(temp_dir_name, 'outputs/output_model/data'),
        model_config=Path(testdata_root).joinpath('model_config.json').read_text(),
        number_of_classes=2,
        number_of_epochs=10,
        batch_size=32,
    )
~~~~
