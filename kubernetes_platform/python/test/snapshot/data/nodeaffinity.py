import kfp
from kfp import dsl
from kfp.kubernetes.node_affinity import add_node_affinity, add_node_affinity_json

@dsl.component
def print_hello_with_explicit_affinity():
    pass

@dsl.component
def print_hello_with_json_affinity():
    pass

@dsl.component
def print_hello_with_required_affinity():
    pass

@dsl.component
def print_hello_with_preferred_affinity():
    pass

@dsl.component
def print_hello_with_combined_affinity():
    pass

@dsl.component
def print_hello_with_empty_json():
    pass

@dsl.pipeline(name="test-node-affinity")
def my_pipeline():    
    # Task 1: Explicit node affinity with matchExpressions (required scheduling)
    task1 = print_hello_with_explicit_affinity()
    task1 = add_node_affinity(
        task1,
        match_expressions=[
            {"key": "disktype", "operator": "In", "values": ["ssd", "nvme"]}
        ]
    )
   
    # Task 2: JSON-based node affinity with preferred scheduling
    task2 = print_hello_with_json_affinity()
    task2 = add_node_affinity_json(
        task2,
        {
            "preferredDuringSchedulingIgnoredDuringExecution": [
                {
                    "weight": 100,
                    "preference": {
                        "matchExpressions": [
                            {
                                "key": "disktype",
                                "operator": "In",
                                "values": ["ssd"]
                            }
                        ]
                    }
                }
            ]
        }
    )

    # Task 3: Required scheduling with matchFields
    task3 = print_hello_with_required_affinity()
    task3 = add_node_affinity(
        task3,
        match_fields=[
            {"key": "metadata.name", "operator": "In", "values": ["node-1", "node-2", "node-3"]}
        ]
    )

    # Task 4: Preferred scheduling with weight
    task4 = print_hello_with_preferred_affinity()
    task4 = add_node_affinity(
        task4,
        match_expressions=[
            {"key": "zone", "operator": "In", "values": ["us-west-1", "us-west-2"]}
        ],
        weight=50
    )

    # Task 5: Combined matchExpressions and matchFields
    task5 = print_hello_with_combined_affinity()
    task5 = add_node_affinity(
        task5,
        match_expressions=[
            {"key": "disktype", "operator": "In", "values": ["ssd"]},
            {"key": "gpu", "operator": "Exists"}
        ],
        match_fields=[
            {"key": "metadata.name", "operator": "NotIn", "values": ["node-5", "node-6"]}
        ]
    )

    # Task 6: JSON with required scheduling
    task6 = print_hello_with_json_affinity()
    task6 = add_node_affinity_json(
        task6,
        {
            "requiredDuringSchedulingIgnoredDuringExecution": {
                "nodeSelectorTerms": [
                    {
                        "matchExpressions": [
                            {
                                "key": "environment",
                                "operator": "In",
                                "values": ["production", "staging"]
                            }
                        ],
                        "matchFields": [
                            {
                                "key": "metadata.name",
                                "operator": "In",
                                "values": ["prod-node-1", "prod-node-2"]
                            }
                        ]
                    }
                ]
            }
        }
    )

    # Task 7: Empty JSON (should not set any affinity)
    task7 = print_hello_with_empty_json()
    task7 = add_node_affinity_json(task7, {})

    # Task 8: Complex JSON with multiple preferred terms
    task8 = print_hello_with_json_affinity()
    task8 = add_node_affinity_json(
        task8,
        {
            "preferredDuringSchedulingIgnoredDuringExecution": [
                {
                    "weight": 100,
                    "preference": {
                        "matchExpressions": [
                            {
                                "key": "zone",
                                "operator": "In",
                                "values": ["us-west-1"]
                            }
                        ]
                    }
                },
                {
                    "weight": 50,
                    "preference": {
                        "matchExpressions": [
                            {
                                "key": "instance-type",
                                "operator": "In",
                                "values": ["n1-standard-4", "n1-standard-8"]
                            }
                        ]
                    }
                }
            ]
        }
    )
