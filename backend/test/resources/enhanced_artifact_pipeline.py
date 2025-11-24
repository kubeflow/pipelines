import kfp
from kfp import dsl
from kfp.components import create_component_from_func, OutputPath, InputPath

# Component 1: Data Processing
def process_data(input_text: str, processed_data: OutputPath()):
    """Process input data and create training dataset"""
    # Simulate data processing
    processed_content = f"PROCESSED: {input_text.upper()}"
    
    with open(processed_data, 'w') as f:
        f.write(processed_content)
    
    print(f"Processed data saved to: {processed_data}")

# Component 2: Training (consumes processed data)
def train_model(training_data: InputPath(), model_path: OutputPath()):
    """Train a simple model using processed data"""
    # Read processed data
    with open(training_data, 'r') as f:
        data = f.read()
    
    # Simulate model training
    model_content = f"MODEL_TRAINED_ON: {data}"
    
    with open(model_path, 'w') as f:
        f.write(model_content)
    
    print(f"Model trained and saved to: {model_path}")

# Component 3: Evaluation
def evaluate_model(model: InputPath(), training_data: InputPath(), results: OutputPath()):
    """Evaluate model and produce results"""
    # Read inputs
    with open(model, 'r') as f:
        model_data = f.read()
    
    with open(training_data, 'r') as f:
        train_data = f.read()
    
    # Simulate evaluation
    evaluation_results = f"EVALUATION: Model '{model_data}' trained on '{train_data}' - Score: 95%"
    
    with open(results, 'w') as f:
        f.write(evaluation_results)
    
    print(f"Evaluation results saved to: {results}")
    return evaluation_results

# Create components
process_data_op = create_component_from_func(process_data)
train_model_op = create_component_from_func(train_model)  
evaluate_model_op = create_component_from_func(evaluate_model)

@dsl.pipeline(name="enhanced-ml-workflow-test")
def enhanced_artifact_pipeline(input_text: str = "kubeflow pipelines"):
    """Enhanced pipeline with realistic ML workflow: Data → Train → Evaluate"""
    
    # Data processing task
    process_task = process_data_op(input_text)
    process_task.set_display_name("Process Training Data")
    
    # Model training task (depends on processed data)
    train_task = train_model_op(process_task.outputs['processed_data'])
    train_task.set_display_name("Train ML Model")
    
    # Model evaluation task (depends on both model and training data)
    eval_task = evaluate_model_op(
        train_task.outputs['model_path'],
        process_task.outputs['processed_data']
    )
    eval_task.set_display_name("Evaluate Model")
    
    # Return final results
    return eval_task.outputs

if __name__ == "__main__":
    kfp.compiler.Compiler().compile(
        pipeline_func=enhanced_artifact_pipeline,
        package_path="enhanced_artifact_pipeline.yaml"
    )
    print("Enhanced pipeline compiled!")
