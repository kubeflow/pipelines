# Copyright 2018 Google LLC
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#      http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.

import argparse
import json
import os
from pathlib import Path

import keras
import numpy as np

parser = argparse.ArgumentParser(description='Train classifier model using Keras')

parser.add_argument('--training-set-features-path', type=str, help='Local or GCS path to the training set features table.')
parser.add_argument('--training-set-labels-path', type=str, help='Local or GCS path to the training set labels (each label is a class index from 0 to num-classes - 1).')
parser.add_argument('--output-model-path', type=str, help='Local or GCS path specifying where to save the trained model. The model (topology + weights + optimizer state) is saved in HDF5 format and can be loaded back by calling keras.models.load_model')
parser.add_argument('--model-config-json', type=str, help='JSON string containing the serialized model structure. Can be obtained by calling model.to_json() on a Keras model.')
parser.add_argument('--num-classes', type=int, help='Number of classifier classes.')
parser.add_argument('--num-epochs', type=int, default=100, help='Number of epochs to train the model. An epoch is an iteration over the entire `x` and `y` data provided.')
parser.add_argument('--batch-size', type=int, default=32, help='Number of samples per gradient update.')

parser.add_argument('--output-model-path-file', type=str, help='Path to a local file containing the output model URI. Needed for data passing until the artifact support is checked in.') #TODO: Remove after the team agrees to let me check in artifact support.
args = parser.parse_args()

# The data, split between train and test sets:
#(x_train, y_train), (x_test, y_test) = cifar10.load_data()
x_train = np.loadtxt(args.training_set_features_path)
y_train = np.loadtxt(args.training_set_labels_path)
print('x_train shape:', x_train.shape)
print(x_train.shape[0], 'train samples')

# Convert class vectors to binary class matrices.
y_train = keras.utils.to_categorical(y_train, args.num_classes)

model = keras.models.model_from_json(args.model_config_json)

model.add(keras.layers.Activation('softmax'))

# initiate RMSprop optimizer
opt = keras.optimizers.rmsprop(lr=0.0001, decay=1e-6)

# Let's train the model using RMSprop
model.compile(loss='categorical_crossentropy',
              optimizer=opt,
              metrics=['accuracy'])

x_train = x_train.astype('float32')
x_train /= 255

model.fit(
    x_train,
    y_train,
    batch_size=args.batch_size,
    epochs=args.num_epochs,
    shuffle=True
)

# Save model and weights
if not args.output_model_path.startswith('gs://'):
    save_dir = os.path.dirname(args.output_model_path)
    if not os.path.isdir(save_dir):
        os.makedirs(save_dir)

model.save(args.output_model_path)
print('Saved trained model at %s ' % args.output_model_path)

Path(args.output_model_path_file).parent.mkdir(parents=True, exist_ok=True)
Path(args.output_model_path_file).write_text(args.output_model_path)
