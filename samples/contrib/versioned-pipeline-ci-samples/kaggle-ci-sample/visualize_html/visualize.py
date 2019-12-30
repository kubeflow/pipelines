# visualizer with html

def datahtml(
    bucket_name,
    train_file_path
):
    import json
    import seaborn as sns
    import matplotlib.pyplot as plt
    import os
    image_path = os.path.join('gs://', bucket_name, 'visualization.png')
    image_url = os.path.join('https://storage.googleapis.com', bucket_name, 'visualization.png')
    html_path = os.path.join('gs://', bucket_name, 'kaggle.html')
    # ouptut visualization to a file

    import pandas as pd
    df_train = pd.read_csv(train_file_path)
    sns.set()
    cols = ['SalePrice', 'OverallQual', 'GrLivArea', 'GarageCars', 'TotalBsmtSF', 'FullBath', 'YearBuilt']
    sns.pairplot(df_train[cols], size = 2.5)
    plt.savefig('visualization.png')
    from tensorflow.python.lib.io import file_io
    file_io.copy('visualization.png', image_path)
    rendered_template = """
    <html>
        <head>
            <title>correlation image</title>
        </head>
        <body>
            <img src={}>
        </body>
    </html>""".format(image_url)
    file_io.write_string_to_file(html_path, rendered_template)

    metadata = {
        'outputs' : [{
        'type': 'web-app',
        'storage': 'gcs',
        'source': html_path,
        }]
    }
    with file_io.FileIO('/mlpipeline-ui-metadata.json', 'w') as f:
        json.dump(metadata, f)

if __name__ == '__main__':
    import argparse
    parser = argparse.ArgumentParser()
    parser.add_argument('--bucket_name', type = str)
    parser.add_argument('--train_file_path', type = str)
    args = parser.parse_args()

    datahtml(args.bucket_name, args.train_file_path)
