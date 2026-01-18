#!/usr/bin/env python3
"""Test pipeline demonstrating HuggingFace Hub artifact import functionality."""

from kfp import dsl

@dsl.component(
    base_image='python:3.11',
    packages_to_install=['transformers', 'torch']
)
def analyze_model(model: dsl.Model) -> str:
    """Analyze a HuggingFace model and return basic info."""
    import os
    import json
    
    # Check what files are available
    model_path = model.path
    print(f"Model path: {model_path}")
    
    files = []
    if os.path.exists(model_path):
        for root, dirs, file_list in os.walk(model_path):
            for file in file_list:
                rel_path = os.path.relpath(os.path.join(root, file), model_path)
                files.append(rel_path)
                if len(files) <= 10:  # Limit output
                    print(f"Found file: {rel_path}")
    
    # Try to read config if available
    config_path = os.path.join(model_path, "config.json")
    config_info = "No config.json found"
    if os.path.exists(config_path):
        try:
            with open(config_path) as f:
                config = json.load(f)
                config_info = f"Model type: {config.get('model_type', 'unknown')}, Architecture: {config.get('architecture', 'unknown')}"
        except Exception as e:
            config_info = f"Error reading config: {e}"
    
    result = f"Model analysis:\n- Files found: {len(files)}\n- Config: {config_info}\n- Model URI: {model.uri}"
    print(result)
    return result

@dsl.pipeline(name='huggingface-model-test')
def huggingface_model_pipeline():
    """Pipeline to test HuggingFace Hub model import."""
    
    # Test 1: Import a small, well-known model (GPT-2)
    gpt2_model = dsl.importer(
        artifact_uri="huggingface://gpt2",
        artifact_class=dsl.Model,
        reimport=False
    ).output
    
    # Test 2: Import specific revision
    gpt2_v2 = dsl.importer(
        artifact_uri="huggingface://gpt2/main", 
        artifact_class=dsl.Model,
        reimport=False
    ).output
    
    # Test 3: Import with query parameters (repo_type)
    dataset_model = dsl.importer(
        artifact_uri="huggingface://squad?repo_type=dataset",
        artifact_class=dsl.Dataset,
        reimport=False
    ).output
    
    # Test 4: Import specific file from a model
    tokenizer_file = dsl.importer(
        artifact_uri="huggingface://gpt2/tokenizer.json",
        artifact_class=dsl.Artifact,
        reimport=False
    ).output
    
    # Analyze the first model
    analysis_task = analyze_model(model=gpt2_model)
    analysis_task.set_display_name("Analyze GPT-2 Model")
    
    # Analyze the second model
    analysis_task2 = analyze_model(model=gpt2_v2)
    analysis_task2.set_display_name("Analyze GPT-2 Main Branch")


if __name__ == "__main__":
    from kfp import compiler
    
    # Compile to pipeline spec
    print("Compiling pipeline to YAML...")
    compiler.Compiler().compile(
        pipeline_func=huggingface_model_pipeline,
        package_path="huggingface_test_pipeline.yaml"
    )
    print("✅ Pipeline compiled to huggingface_test_pipeline.yaml")
    
    # Test local execution
    print("\n🧪 Testing local execution...")
    try:
        from kfp import local
        local.init(runner=local.SubprocessRunner())
        
        # Run just the analysis component with a mock model
        print("Testing analysis component locally...")
        
        # Create a mock model artifact for testing
        mock_model = dsl.Model()
        mock_model.path = "/tmp/test_model"
        mock_model.uri = "huggingface://gpt2"
        
        import os
        import json
        os.makedirs("/tmp/test_model", exist_ok=True)
        with open("/tmp/test_model/config.json", "w") as f:
            json.dump({"model_type": "gpt2", "architecture": "GPT2LMHeadModel"}, f)
        
        result = analyze_model(model=mock_model)
        print(f"✅ Local test result: {result}")
        
    except ImportError as e:
        print(f"⚠️  Local execution test skipped (missing dependencies): {e}")
    except Exception as e:
        print(f"⚠️  Local execution test failed: {e}")