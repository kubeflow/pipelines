import pickle

with open('my_function.pkl', 'rb') as f:
    depickled_function = pickle.load(f)

depickled_function()