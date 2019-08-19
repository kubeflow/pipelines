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
import time
import os

import pandas as pd

np.random.seed(99)

def preprocessing(image_dir, result_dir):

    """ Load and Process Images """

    races_to_consider = [0,4]
    unprivileged_groups = [{'race': 4.0}]
    privileged_groups = [{'race': 0.0}]
    favorable_label = 0.0
    unfavorable_label = 1.0

    img_size = 64

    protected_race = []
    outcome_gender = []
    feature_image = []
    feature_age = []

    for i, image_path in enumerate(glob.glob(image_dir + "*.jpg")):
        try:
            age, gender, race = image_path.split('/')[-1].split("_")[:3]
            age = int(age)
            gender = int(gender)
            race = int(race)

            if race in races_to_consider:
                protected_race.append(race)
                outcome_gender.append(gender)
                feature_image.append(np.array(Image.open(image_path).resize((img_size, img_size))))
                feature_age.append(age)
        except:
            print("Missing: " + image_path)

    feature_image_mat = np.array(feature_image)
    outcome_gender_mat =  np.array(outcome_gender)
    protected_race_mat =  np.array(protected_race)
    age_mat = np.array(feature_age)

    """ Split the dataset into train and test """

    feature_image_mat_normed = 2.0 *feature_image_mat.astype('float32')/256.0 - 1.0

    N = len(feature_image_mat_normed)
    ids = np.random.permutation(N)
    train_size=int(0.7 * N)
    X_train = feature_image_mat_normed[ids[0:train_size]]
    y_train = outcome_gender_mat[ids[0:train_size]]
    X_test = feature_image_mat_normed[ids[train_size:]]
    y_test = outcome_gender_mat[ids[train_size:]]

    p_train = protected_race_mat[ids[0:train_size]]
    p_test = protected_race_mat[ids[train_size:]]

    age_train = age_mat[ids[0:train_size]]
    age_test = age_mat[ids[train_size:]]

    batch_size = 64

    X_train = X_train.transpose(0,3,1,2)
    X_test = X_test.transpose(0,3,1,2)

    os.system('rm -r ' + result_dir)
    os.system('mkdir ' + result_dir)

    np.save(result_dir + 'X_train', X_train)
    np.save(result_dir + 'X_test', X_test)
    np.save(result_dir + 'y_train', y_train)
    np.save(result_dir + 'y_test', y_test)
    np.save(result_dir + 'p_train', p_train)
    np.save(result_dir + 'p_test', p_test)
