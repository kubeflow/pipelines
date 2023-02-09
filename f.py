# import numpy as np
# from sklearn import datasets
# from sklearn import linear_model
# from sklearn import model_selection

# total_samples = 442
# test_size = 0.2

# diabetes_X, diabetes_y = datasets.load_diabetes(return_X_y=True)

# X = np.random.rand(10, 10)

# def drop_features():
#     return diabetes_X[:, np.newaxis, 2]

# diabetes_X = drop_features(diabetes_X)

# diabetes_X_train, diabetes_X_test, diabetes_y_train, diabetes_y_test = model_selection.train_test_split(
#     diabetes_X, diabetes_y, shuffle=False, test_size=test_size)

# model = linear_model.LinearRegression()
# model.fit(diabetes_X_train, diabetes_y_train)

from kfp import components, dsl

# comp = components.load_component_from_file('f.yaml')
comp = components.load_component_from_file('tablesin.yaml')
ir_file = 'tablesout.yaml'

ir_file = __file__.replace('.py', '.yaml')

if __name__ == '__main__':
    import datetime
    import warnings
    import webbrowser

    from google.cloud import aiplatform

    from kfp import compiler

    warnings.filterwarnings('ignore')

    compiler.Compiler().compile(pipeline_func=comp, package_path=ir_file)
    pipeline_name = __file__.split('/')[-1].replace('_', '-').replace('.py', '')
    display_name = datetime.datetime.now().strftime('%m-%d-%Y-%H-%M-%S')
    job_id = f'{pipeline_name}-{display_name}'
    aiplatform.PipelineJob(
        template_path=ir_file,
        pipeline_root='gs://cjmccarthy-kfp-default-bucket',
        display_name=pipeline_name,
        job_id=job_id,
        # parameter_values={
        #     'single_run_max_secs': 2,
        #     'project': '271009669852',
        #     'location': '',
        #     'num_selected_trials': 2,
        #     'deadline_hours': 1,
        #     'num_parallel_trials': 2,
        #     'root_dir': ''
        # },
    ).submit()
    url = f'https://console.cloud.google.com/vertex-ai/locations/us-central1/pipelines/runs/{pipeline_name}-{display_name}?project=271009669852'
    webbrowser.open_new_tab(url)