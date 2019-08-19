# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
import glob
import PIL
from PIL import Image
import numpy as np
import argparse

import torch
import torch.utils.data
from torch.autograd import Variable
import torch.nn as nn
from torchsummary import summary
import time

import pandas as pd
from minio import Minio

from preprocessing import preprocessing
from PyTorchModel import ThreeLayerCNN

np.random.seed(99)
torch.manual_seed(99)


def create_bucket(cos, name):
    if not cos.bucket_exists(name):
        cos.make_bucket(name)


if __name__ == "__main__":
    parser = argparse.ArgumentParser()
    parser.add_argument('--dummy', type=int, default=10000, help='dummy parameters')
    parser.add_argument('--training_id', type=str, default='training-example', help='Training_job_id')
    parser.add_argument('--data_bucket', type=str, default='training-data', help='Bucket to store processed data')
    parser.add_argument('--result_bucket', type=str, default='training-result', help='Bucket to store training results')
    args = parser.parse_args()

    training_id = args.training_id
    data_bucket = args.data_bucket
    result_bucket = args.result_bucket

    image_dir = "/tmp/data/UTKFace/"
    result_dir = "/tmp/data/"

    cos = Minio('minio-service:9000',
                access_key='minio',
                secret_key='minio123',
                secure=False)

    create_bucket(cos, data_bucket)
    create_bucket(cos, result_bucket)

    preprocessing(image_dir, result_dir)

    img_size = 64
    batch_size = 64

    X_train = np.load(result_dir + 'X_train.npy')
    y_train = np.load(result_dir + 'y_train.npy')
    X_test = np.load(result_dir + 'X_test.npy')
    y_test = np.load(result_dir + 'y_test.npy')

    train = torch.utils.data.TensorDataset(Variable(torch.FloatTensor(X_train.astype('float32'))), Variable(torch.LongTensor(y_train.astype('float32'))))
    train_loader = torch.utils.data.DataLoader(train, batch_size=batch_size, shuffle=True)
    test = torch.utils.data.TensorDataset(Variable(torch.FloatTensor(X_test.astype('float32'))), Variable(torch.LongTensor(y_test.astype('float32'))))
    test_loader = torch.utils.data.DataLoader(test, batch_size=batch_size, shuffle=False)

    device = torch.device('cuda:0' if torch.cuda.is_available() else 'cpu')
    model = ThreeLayerCNN().to(device)
    summary(model, (3, img_size, img_size))

    """ Training the network """

    num_epochs = 5
    learning_rate = 0.001
    print_freq = 100

    # Specify the loss and the optimizer
    criterion = nn.CrossEntropyLoss()
    optimizer = torch.optim.Adam(model.parameters(), lr=learning_rate)

    # Start training the model
    num_batches = len(train_loader)
    for epoch in range(num_epochs):
        for idx, (images, labels) in enumerate(train_loader):
            images = images.to(device)
            labels = labels.to(device)

            outputs = model(images)
            loss = criterion(outputs, labels)
            optimizer.zero_grad()
            loss.backward()
            optimizer.step()

            if (idx+1) % print_freq == 0:
                print ('Epoch [{}/{}], Step [{}/{}], Loss: {:.4f}' .format(epoch+1, num_epochs, idx+1, num_batches, loss.item()))

    # Run model on test set in eval mode.
    model.eval()
    correct = 0
    y_pred = []
    with torch.no_grad():
        for images, labels in test_loader:
            images = images.to(device)
            labels = labels.to(device)
            outputs = model(images)
            _, predicted = torch.max(outputs.data, 1)
            correct += predicted.eq(labels.data.view_as(predicted)).sum().item()
            y_pred += predicted.tolist()
        print('accuracy=' + str(1. * correct / len(test_loader.dataset)))
    # convert y_pred to np array
    y_pred = np.array(y_pred)

    # Save the entire model to enable automated serving
    torch.save(model.state_dict(), result_dir + 'model.pt')
    print("Model saved at " + result_dir + 'model.pt')

    # Upload test set and model to S3
    cos.fput_object(data_bucket, 'processed_data/X_test.npy', result_dir + 'X_test.npy')
    cos.fput_object(data_bucket, 'processed_data/y_test.npy', result_dir + 'y_test.npy')
    cos.fput_object(data_bucket, 'processed_data/p_train.npy', result_dir + 'p_train.npy')
    cos.fput_object(data_bucket, 'processed_data/p_test.npy', result_dir + 'p_test.npy')
    cos.fput_object(result_bucket, training_id +  '/_submitted_code/model.zip', './model.zip')
    cos.fput_object(result_bucket, training_id +  '/model.pt', result_dir + '/model.pt')
