from kfp import dsl

@dsl.component(base_image="python:3.10.12",packages_to_install = [
    'seaborn',
    'qiskit',
    'qiskit-machine-learning',
    'qiskit-aer',
    'numpy',
    'pandas',
    'scikit-learn',
    'matplotlib',
])
def QSVM():
    import numpy as np
    import pandas as pd
    from sklearn.model_selection import train_test_split
    from sklearn.preprocessing import StandardScaler
    from sklearn.metrics import confusion_matrix, accuracy_score
    from qiskit_aer import AerSimulator
    from qiskit.circuit.library import ZZFeatureMap
    from qiskit_machine_learning.algorithms import QSVC
    import matplotlib.pyplot as plt
    import seaborn as sns

    # Load dataset (you should replace this with your actual dataset loading)
    def load_parkinsons_data():
        #Load dataset
        p_now = [[1,0.85247,0.71826,0.57227,240,239,0.008064,0.000087,0.00218],[1,0.76686,0.69481,0.53966,234,233,0.008258,0.000073,0.00195],
        [1,0.85083,0.67604,0.58982,232,231,0.008340,0.000060,0.00176],
        [0,0.41121,0.79672,0.59257,178,177,0.010858,0.000183,0.00419],
        [0,0.32790,0.79782,0.53028,236,235,0.008162,0.002669,0.00535]]
        notp =[[1, 0.84881, 0.60125, 0.44782, 351, 350, 0.005510131, 7.27E-05, 0.00165],
        [1, 0.70649, 0.60081, 0.50228, 339, 338, 0.005697116, 8.77E-05, 0.00174],
        [1, 0.84703, 0.62323, 0.45198, 333, 332, 0.00579286, 5.74E-05, 0.00107],
        [1, 0.78601, 0.61539, 0.20724, 594, 593, 0.003249218, 3.45E-05, 0.00039],
        [1, 0.78213, 0.61626, 0.17873, 598, 597, 0.003226095, 2.5E-05, 0.00037]]
        X = np.vstack((p_now, notp))
        y = np.array([1]*len(p_now) + [0]*len(notp))
        return X, y

    # Load and preprocess data
    X, y = load_parkinsons_data()
    X_train, X_test, y_train, y_test = train_test_split(X, y, test_size=0.3, random_state=42)

    # Normalize the data
    scaler = StandardScaler()
    X_train_scaled = scaler.fit_transform(X_train)
    X_test_scaled = scaler.transform(X_test)

    # Set up the quantum instance
    backend = AerSimulator()

    # Define the feature map
    feature_map = ZZFeatureMap(feature_dimension=X_train_scaled.shape[1], reps=2)

    # Create and train the QSVC model
    qsvc = QSVC()
    qsvc.fit(X_train_scaled, y_train)

    # Make predictions
    y_pred = qsvc.predict(X_test_scaled)

    # Calculate accuracy
    accuracy = accuracy_score(y_test, y_pred)
    print(f"Accuracy: {accuracy:.4f}")

    # Plot confusion matrix
    cm = confusion_matrix(y_test, y_pred)
    plt.figure(figsize=(10,7))
    sns.heatmap(cm, annot=True, fmt='d', cmap='Blues')
    plt.title("Confusion Matrix for Parkinson's Disease Prediction")
    plt.ylabel('True Label')
    plt.xlabel('Predicted Label')
    plt.show()

    # Predict on new data
    new_data = np.array([[0, 0.80766, 0.73961, 0.20569, 445, 444, 0.004334704, 2.43E-05, 0.00072],
    [1, 0.83967, 0.80944, 0.45038, 259, 257, 0.007310325, 0.00251383, 0.00534],
    [1, 0.81525, 0.73462, 0.64849, 303, 302, 0.006381791, 0.000149359, 0.00292],
    [1, 0.79163, 0.80358, 0.43866, 330, 329, 0.005844644, 8.21E-05, 0.00278],
    [0, 0.7594, 0.68265, 0.39428, 443, 442, 0.004354498, 6.69E-05, 0.00076],
    [1, 0.81547, 0.64809, 0.60227, 243, 242, 0.007963248, 9.04E-05, 0.00294],
    [1, 0.82586, 0.59259, 0.44395, 354, 353, 0.005455777, 4.0E-05, 0.00129],
    [1, 0.73403, 0.61812, 0.50801, 343, 342, 0.00563253, 8.34E-05, 0.00167],
    [1, 0.87601, 0.62297, 0.43552, 346, 345, 0.005573364, 5.66E-05, 0.00118]
    ])
    new_data_scaled = scaler.transform(new_data)
    predictions = qsvc.predict(new_data_scaled)
    print("\nPredictions for new data:")
    for i, pred in enumerate(predictions):
        print(f"Sample {i+1}: {'Parkinsons' if pred == 1 else 'Healthy'}")



@dsl.pipeline
def qsvm_pipeline():
    qsvm_task = QSVM()

from kfp import compiler

compiler.Compiler().compile(qsvm_pipeline, 'qsvm_pipeline.yaml')