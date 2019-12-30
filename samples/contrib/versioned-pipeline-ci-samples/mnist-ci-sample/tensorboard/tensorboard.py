if __name__ == '__main__':
    import json
    import argparse

    parser = argparse.ArgumentParser()
    parser.add_argument('--logdir', type=str)
    parser.add_argument('--output_path', type=str, default='/')

    args = parser.parse_args()

    metadata = {
      'outputs' : [{
        'type': 'tensorboard',
        'source': args.logdir,
      }]
    }
    with open(args.output_path+'mlpipeline-ui-metadata.json', 'w') as f:
      json.dump(metadata, f)