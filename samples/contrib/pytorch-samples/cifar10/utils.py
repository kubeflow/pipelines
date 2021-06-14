import pandas as pd
import json
import os

from sklearn.metrics import confusion_matrix
from io import StringIO
import boto3


class Visualization:
    def __init__(self):
        self.parser_args = None

    def _generate_confusion_matrix_metadata(self, confusion_matrix_path, vocab):
        print("Generating Confusion matrix Metadata")
        metadata = {
            "type": "confusion_matrix",
            "format": "csv",
            "schema": [
                {"name": "target", "type": "CATEGORY"},
                {"name": "predicted", "type": "CATEGORY"},
                {"name": "count", "type": "NUMBER"},
            ],
            "source": confusion_matrix_path,
            # Convert vocab to string because for bealean values we want "True|False" to match csv data.
            "labels": list(map(str, vocab)),
        }
        self._write_ui_metadata(
            metadata_filepath="/mlpipeline-ui-metadata.json", metadata_dict=metadata
        )

    def _write_ui_metadata(self, metadata_filepath, metadata_dict, key="outputs"):
        if not os.path.exists(metadata_filepath):
            metadata = {key: [metadata_dict]}
        else:
            with open(metadata_filepath) as fp:
                metadata = json.load(fp)
                metadata_outputs = metadata[key]
                metadata_outputs.append(metadata_dict)

        print("Writing to file: {}".format(metadata_filepath))
        with open(metadata_filepath, "w") as fp:
            json.dump(metadata, fp)

    def _enable_tensorboard_visualization(self, tensorboard_root):
        print("Enabling Tensorboard Visualization")
        metadata = {
            "type": "tensorboard",
            "source": tensorboard_root,
        }

        import os

        os.environ["AWS_REGION"] = "us-east-2"

        self._write_ui_metadata(
            metadata_filepath="/mlpipeline-ui-metadata.json", metadata_dict=metadata
        )

    def _visualize_accuracy_metric(self, accuracy):
        metadata = {
            "name": "accuracy-score",
            "numberValue": accuracy,
            "format": "PERCENTAGE",
        }
        self._write_ui_metadata(
            metadata_filepath="/mlpipeline-metrics.json", metadata_dict=metadata, key="metrics"
        )

    def _generate_confusion_matrix(self, confusion_matrix_dict):
        actuals = confusion_matrix_dict["actuals"]
        preds = confusion_matrix_dict["preds"]

        bucket_name = confusion_matrix_dict["bucket_name"]
        folder_name = confusion_matrix_dict["folder_name"]

        # Generating confusion matrix
        df = pd.DataFrame(list(zip(actuals, preds)), columns=["target", "predicted"])
        vocab = list(df["target"].unique())
        cm = confusion_matrix(df["target"], df["predicted"], labels=vocab)
        data = []
        for target_index, target_row in enumerate(cm):
            for predicted_index, count in enumerate(target_row):
                data.append((vocab[target_index], vocab[predicted_index], count))

        confusion_matrix_df = pd.DataFrame(data, columns=["target", "predicted", "count"])

        confusion_matrix_key = os.path.join(folder_name, "confusion_df.csv")

        # Logging confusion matrix to ss3
        csv_buffer = StringIO()
        confusion_matrix_df.to_csv(csv_buffer, index=False, header=False)
        s3_resource = boto3.resource("s3")
        s3_resource.Object(bucket_name, confusion_matrix_key).put(Body=csv_buffer.getvalue())

        # Generating metadata
        confusion_matrix_s3_path = s3_path = "s3://" + bucket_name + "/" + confusion_matrix_key
        self._generate_confusion_matrix_metadata(confusion_matrix_s3_path, vocab)

    def generate_visualization(
        self, tensorboard_root=None, accuracy=None, confusion_matrix_dict=None
    ):
        print("Tensorboard Root: {}".format(tensorboard_root))
        print("Accuracy: {}".format(accuracy))

        if tensorboard_root:
            self._enable_tensorboard_visualization(tensorboard_root)

        if accuracy:
            self._visualize_accuracy_metric(accuracy=accuracy)

        if confusion_matrix_dict:
            self._generate_confusion_matrix(confusion_matrix_dict=confusion_matrix_dict)
