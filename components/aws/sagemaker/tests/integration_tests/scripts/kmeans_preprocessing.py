import pickle
import gzip
import numpy
import io
from sagemaker.amazon.common import write_numpy_to_dense_tensor


# Load the dataset
with gzip.open('/opt/ml/processing/input/mnist.pkl.gz', 'rb') as f:
    train_set, valid_set, test_set = pickle.load(f, encoding='latin1')

# process the data
# Convert the training data into the format required by the SageMaker KMeans algorithm
buf = io.BytesIO()
write_numpy_to_dense_tensor(buf, train_set[0], train_set[1])
buf.seek(0)

with open('/opt/ml/processing/output_train/train_data', 'wb') as train_data_file:
    pickle.dump(buf, train_data_file)

# Convert the test data into the format required by the SageMaker KMeans algorithm
write_numpy_to_dense_tensor(buf, test_set[0], test_set[1])
buf.seek(0)

with open('/opt/ml/processing/output_test/test_data', 'wb') as test_data_file:
    pickle.dump(buf, test_data_file)

# Convert the valid data into the format required by the SageMaker KMeans algorithm
numpy.savetxt('/opt/ml/processing/output_valid/valid-data.csv', valid_set[0], delimiter=',', fmt='%g')
