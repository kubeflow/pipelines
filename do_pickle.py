import pickle
import numpy as np


def my_function():
    print(np.random.rand(10, 20))


with open('my_function.pkl', 'wb') as f:
    pickled_function = pickle.dump(my_function, f)
