from kfp import dsl
from kfp import compiler
from kfp.dsl import (Input, Output, Dataset, Model, Metrics, ClassificationMetrics)

@dsl.component(
        base_image="python:3.10", 
        packages_to_install=["numpy", "pandas", "loguru"])
def load_data(
    data_url: str, 
    data_size: int, 
    credit_risk_dataset: Output[Dataset]):

    '''
    Downloads credit_risk_dataset.csv file and generates  
    additional synthetic data for benchmarking and testing purposes.
    
    Input Parameters
    ----------------
    data_url : str
        url where the dataset is hosted
    data_size : int
        size of final dataset desired, default 1M rows
    
    Output Artifacts
    ----------------
    credit_risk_dataset : Dataset
        data that has been synthetically augmented or loaded from URL provided
    '''

    import numpy as np
    import pandas as pd
    from loguru import logger
    
    logger.info("Loading csv from {}", data_url)
    data = pd.read_csv(data_url)
    logger.info("Done!")

    # number of rows to generate
    if data_size < data.shape[0]:
        pass
    else:
        logger.info("Generating {:,} rows of data...", data_size)
        repeats = data_size // len(data)
        data = data.loc[np.repeat(data.index.values, repeats + 1)]
        data = data.iloc[:data_size]
        
        # perturbing all int/float columns
        person_age = data["person_age"].values + np.random.randint(
            -1, 1, size=len(data)
        )
        person_income = data["person_income"].values + np.random.normal(
            0, 10, size=len(data)
        )
        person_emp_length = data[
            "person_emp_length"
        ].values + np.random.randint(-1, 1, size=len(data))
        loan_amnt = data["loan_amnt"].values + np.random.normal(
            0, 5, size=len(data)
        )
        loan_int_rate = data["loan_int_rate"].values + np.random.normal(
            0, 0.2, size=len(data)
        )
        loan_percent_income = data["loan_percent_income"].values + (
            np.random.randint(0, 100, size=len(data)) / 1000
        )
        cb_person_cred_hist_length = data[
            "cb_person_cred_hist_length"
        ].values + np.random.randint(0, 2, size=len(data))
        
        # perturbing all binary columns
        perturb_idx = np.random.rand(len(data)) > 0.1
        random_values = np.random.choice(
            data["person_home_ownership"].unique(), len(data)
        )
        person_home_ownership = np.where(
            perturb_idx, data["person_home_ownership"], random_values
        )
        perturb_idx = np.random.rand(len(data)) > 0.1
        random_values = np.random.choice(
            data["loan_intent"].unique(), len(data)
        )
        loan_intent = np.where(perturb_idx, data["loan_intent"], random_values)
        perturb_idx = np.random.rand(len(data)) > 0.1
        random_values = np.random.choice(
            data["loan_grade"].unique(), len(data)
        )
        loan_grade = np.where(perturb_idx, data["loan_grade"], random_values)
        perturb_idx = np.random.rand(len(data)) > 0.1
        random_values = np.random.choice(
            data["cb_person_default_on_file"].unique(), len(data)
        )
        cb_person_default_on_file = np.where(
            perturb_idx, data["cb_person_default_on_file"], random_values
        )
        data = pd.DataFrame(
            list(
                zip(
                    person_age,
                    person_income,
                    person_home_ownership,
                    person_emp_length,
                    loan_intent,
                    loan_grade,
                    loan_amnt,
                    loan_int_rate,
                    data["loan_status"].values,
                    loan_percent_income,
                    cb_person_default_on_file,
                    cb_person_cred_hist_length,
                )
            ),
            columns = data.columns,
        )

        data = data.drop_duplicates()
        assert len(data) == data_size
        data.reset_index(drop = True)

    data.to_csv(credit_risk_dataset.path, index = None)

@dsl.component(
        base_image="python:3.10", 
        packages_to_install=["pandas", "scikit-learn", "loguru"])
def create_train_test_set(
    data: Input[Dataset],
    x_train_data: Output[Dataset], 
    y_train_data: Output[Dataset],
    x_test_data: Output[Dataset],
    y_test_data: Output[Dataset]):

    '''
    Creates 75:25 split of input dataset for model evaluation.
    
    Input Artifacts
    ---------------
    data : Dataset
        dataset that has been synthetically augmented by the load_data() function
    
    Output Artifacts
    ----------------
    x_train_data : Dataset
        training features, 75% of original dataset
    y_train_data : Dataset
        training labels of target variable, loan_status 
    x_test_data : Dataset
        test features, 25% of original dataset
    y_test_data : Dataset
        test labels of target variable, loan_status
    '''

    import pandas as pd
    from loguru import logger
    from sklearn.model_selection import train_test_split

    data = pd.read_csv(data.path)

    logger.info("Creating training and test sets...")
    train, test = train_test_split(data, test_size = 0.25, random_state = 0)

    X_train = train.drop(["loan_status"], axis = 1)
    y_train = train["loan_status"]
    
    X_test = test.drop(["loan_status"], axis = 1)
    y_test = test["loan_status"]
    
    logger.info("Training and test sets created.\n" \
                "X_train size: {}, y_train size: {}\n" \
                "X_test size: {}, y_test size: {}", 
                X_train.shape, y_train.shape, X_test.shape, y_test.shape)
    
    X_train.to_csv(x_train_data.path, index = False)
    y_train.to_csv(y_train_data.path, index = False, header = None)
    X_test.to_csv(x_test_data.path, index = False)
    y_test.to_csv(y_test_data.path, index = False, header = None)

@dsl.component(
        base_image="python:3.10", 
        packages_to_install=["pandas", "scikit-learn"])
def preprocess_features(
    x_train: Input[Dataset],
    x_test: Input[Dataset], 
    x_train_processed: Output[Dataset],
    x_test_processed: Output[Dataset]):

    '''
    Performs data preprocessing of training and test features.
    
    Input Artifacts
    ---------------
    x_train : Dataset
        original unprocessed training features 
    x_test : Dataset
        original unprocessed test features
    
    Output Artifacts
    ----------------
    x_train_processed : Dataset
        processed and scaled training features
    x_test_processed : Dataset
        processed and scaled test features 
    '''

    import pandas as pd    
    from sklearn.compose import ColumnTransformer
    from sklearn.impute import SimpleImputer
    from sklearn.pipeline import Pipeline
    from sklearn.preprocessing import OneHotEncoder, PowerTransformer
    
    X_train = pd.read_csv(x_train.path)
    X_test = pd.read_csv(x_test.path)

    # data processing pipeline
    num_imputer = Pipeline(steps=[("imputer", SimpleImputer(strategy = "median"))])
    pow_transformer = PowerTransformer()
    cat_transformer = OneHotEncoder(handle_unknown = "ignore")
    preprocessor = ColumnTransformer(
        transformers = [
            (
                "num",
                num_imputer,
                [
                    "loan_int_rate",
                    "person_emp_length",
                    "cb_person_cred_hist_length",
                ],
            ),
            (
                "pow",
                pow_transformer,
                ["person_age", "person_income", "loan_amnt", "loan_percent_income"],
            ),
            (
                "cat",
                cat_transformer,
                [
                    "person_home_ownership",
                    "loan_intent",
                    "loan_grade",
                    "cb_person_default_on_file",
                ],
            ),
        ],
        remainder="passthrough",
    )

    preprocess = Pipeline(steps = [("preprocessor", preprocessor)])

    X_train = pd.DataFrame(preprocess.fit_transform(X_train))
    X_test = pd.DataFrame(preprocess.transform(X_test))
    
    X_train.to_csv(x_train_processed.path, index = False, header = None)
    X_test.to_csv(x_test_processed.path, index = False, header = None)

@dsl.component(
        base_image="python:3.10", 
        packages_to_install=["pandas", "xgboost", "joblib", "loguru"])
def train_xgboost_model(
    x_train: Input[Dataset],
    y_train: Input[Dataset], 
    xgb_model: Output[Model]):

    '''
    Trains an XGBoost classification model.
    
    Input Artifacts
    ---------------
    x_train : Dataset
        processed and scaled training features 
    y_train : Dataset
        training labels of target variable, loan_status
    
    Output Artifacts
    ----------------
    xgb_model : Model
        trained XGBoost model
    '''

    import joblib
    import pandas as pd
    import xgboost as xgb
    from loguru import logger
    
    X_train = pd.read_csv(x_train.path, header = None)
    y_train = pd.read_csv(y_train.path, header = None)

    dtrain = xgb.DMatrix(X_train.values, y_train.values)
    
    # define model parameters
    params = {
        "objective": "binary:logistic",
        "eval_metric": "logloss",
        "nthread": 4,  # num_cpu
        "tree_method": "hist",
        "learning_rate": 0.02,
        "max_depth": 10,
        "min_child_weight": 6,
        "n_jobs": 4,  # num_cpu,
        "verbosity": 1
    }
    
    # train XGBoost model
    logger.info("Training XGBoost model...")
    clf = xgb.train(params = params, 
                    dtrain = dtrain, 
                    num_boost_round = 500)
        
    with open(xgb_model.path, "wb") as file_writer:
        joblib.dump(clf, file_writer) 

@dsl.component(
        base_image="python:3.10", 
        packages_to_install=["daal4py", "joblib", "loguru"])
def convert_xgboost_to_daal4py(
    xgb_model: Input[Model],
    daal4py_model: Output[Model]):

    '''
    Converts XGBoost model to inference-optimized daal4py classifier.
    
    Input Artifacts
    ---------------
    xgb_model : Model
        trained XGBoost classifier 
    
    Output Artifacts
    ----------------
    daal4py_model : Model
        inference-optimized daal4py classifier
    '''

    import daal4py as d4p
    import joblib
    from loguru import logger
    
    with open(xgb_model.path, "rb") as file_reader:
        clf = joblib.load(file_reader)
        
    logger.info("Converting XGBoost model to Daal4py...")
    daal_model = d4p.get_gbt_model_from_xgboost(clf)
    logger.info("Done!")
    
    with open(daal4py_model.path, "wb") as file_writer:
        joblib.dump(daal_model, file_writer)

@dsl.component(
        base_image="python:3.10", 
        packages_to_install=["daal4py", "pandas", "scikit-learn",
                             "scikit-learn-intelex", "joblib"])
def daal4py_inference(
    x_test: Input[Dataset],
    y_test: Input[Dataset],
    daal4py_model: Input[Model],
    prediction_data: Output[Dataset],
    report: Output[Dataset],
    metrics: Output[Metrics]
):
    
    '''
    Computes predictions using the inference-optimized daal4py classifier 
    and evaluates model performance.
    
    Input Artifacts
    ---------------
    x_test : Dataset
        processed and scaled test features 
    y_test : Dataset
        test labels of target variable, loan_status 
    daal4py_model : Model
        inference-optimized daal4py classifier
    
    Output Artifacts
    ----------------
    prediction_data : Dataset
        dataset containing true test labels and predicted probabilities
    report : Dataset
        summary of the precision, recall, F1 score for each class
    metrics : Metrics
        scalar classification metrics containing the model's AUC and accuracy
    '''
    
    import daal4py as d4p
    import joblib
    import pandas as pd

    from sklearnex import patch_sklearn
    patch_sklearn()
    from sklearn.metrics import roc_auc_score, accuracy_score, classification_report
    
    X_test = pd.read_csv(x_test.path, header = None)
    y_test = pd.read_csv(y_test.path, header = None)
    
    with open(daal4py_model.path, "rb") as file_reader:
        daal_model = joblib.load(file_reader)
    
    daal_prediction = d4p.gbt_classification_prediction(
            nClasses = 2, 
            resultsToEvaluate = "computeClassLabels|computeClassProbabilities"
        ).compute(X_test, daal_model)
        
    y_pred = daal_prediction.prediction
    y_prob = daal_prediction.probabilities[:,1]
    
    results = classification_report(
        y_test, y_pred,
        target_names = ["Non-Default", "Default"],
        output_dict = True
    )
    results = pd.DataFrame(results).transpose()
    results.to_csv(report.path)
    
    auc = roc_auc_score(y_test, y_prob)
    metrics.log_metric('AUC', auc)
    
    accuracy = (accuracy_score(y_test, y_pred)*100)
    metrics.log_metric('Accuracy', accuracy)
    
    predictions = pd.DataFrame({'y_test': y_test.values.flatten(), 
                                'y_prob': y_prob})
    predictions.to_csv(prediction_data.path, index = False)

@dsl.component(
        base_image="python:3.10", 
        packages_to_install=["numpy", "pandas", "scikit-learn",
                             "scikit-learn-intelex"])
def plot_roc_curve(
    predictions: Input[Dataset],
    class_metrics: Output[ClassificationMetrics]
):
    
    '''
    Function to plot Receiver Operating Characteristic (ROC) curve.
    
    Input Artifacts
    ---------------
    predictions : Dataset
        dataset containing true test labels and predicted probabilities
    
    Output Artifacts
    ----------------
    class_metrics : ClassificationMetrics
        classification metrics containing fpr, tpr, and thresholds
    '''
    
    import pandas as pd
    from numpy import inf
    
    from sklearnex import patch_sklearn
    patch_sklearn()
    from sklearn.metrics import roc_curve
    
    prediction_data = pd.read_csv(predictions.path)

    fpr, tpr, thresholds = roc_curve(
        y_true = prediction_data['y_test'], 
        y_score = prediction_data['y_prob'], 
        pos_label = 1)    
    thresholds[thresholds == inf] = 0

    class_metrics.log_roc_curve(fpr, tpr, thresholds)

@dsl.pipeline
def intel_xgboost_daal4py_pipeline(
    data_url: str, 
    data_size: int):

    load_data_op = load_data(
        data_url = data_url, data_size = data_size
    )
    
    create_train_test_set_op = create_train_test_set(
        data = load_data_op.outputs['credit_risk_dataset']
    )
    
    preprocess_features_op = preprocess_features(
        x_train = create_train_test_set_op.outputs['x_train_data'],
        x_test = create_train_test_set_op.outputs['x_test_data']
    )
    
    train_xgboost_model_op = train_xgboost_model(
        x_train = preprocess_features_op.outputs['x_train_processed'], 
        y_train = create_train_test_set_op.outputs['y_train_data']        
    )

    convert_xgboost_to_daal4py_op = convert_xgboost_to_daal4py(
        xgb_model = train_xgboost_model_op.outputs['xgb_model']
    )

    daal4py_inference_op = daal4py_inference(
        x_test = preprocess_features_op.outputs['x_test_processed'],
        y_test = create_train_test_set_op.outputs['y_test_data'],
        daal4py_model = convert_xgboost_to_daal4py_op.outputs['daal4py_model']
    )

    plot_roc_curve_op = plot_roc_curve(
        predictions = daal4py_inference_op.outputs['prediction_data']
    )

if __name__ == '__main__':
    # Compiling the pipeline
    compiler.Compiler().compile(
        pipeline_func = intel_xgboost_daal4py_pipeline, 
        package_path = 'intel-xgboost-daal4py-pipeline.yaml')