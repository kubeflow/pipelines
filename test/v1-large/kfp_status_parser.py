import json
from typing import Dict, List, Any, Optional
from dataclasses import dataclass
from datetime import datetime


@dataclass
class NodeStatus:
    """Represents the status of a single pipeline node/component."""
    id: str
    name: str
    display_name: str
    type: str  # Pod, DAG, etc.
    template_name: str
    phase: str  # Succeeded, Failed, Running, etc.
    started_at: Optional[str]
    finished_at: Optional[str]
    message: Optional[str] = None
    exit_code: Optional[str] = None
    boundary_id: Optional[str] = None
    host_node_name: Optional[str] = None


def parse_kfp_response(response: Dict[str, Any]) -> Dict[str, NodeStatus]:
    """
    Parse KFP v1 API response to extract node statuses.
    
    Args:
        response: The full KFP API response dictionary
        
    Returns:
        Dictionary mapping node IDs to NodeStatus objects
    """
    # Extract workflow manifest from the response
    workflow_manifest_str = response['pipeline_runtime']['workflow_manifest']
    workflow_manifest = json.loads(workflow_manifest_str)
    
    # Get nodes from the workflow status
    nodes = workflow_manifest.get('status', {}).get('nodes', {})
    
    node_statuses = {}
    
    for node_id, node_data in nodes.items():
        status = NodeStatus(
            id=node_id,
            name=node_data.get('name', ''),
            display_name=node_data.get('displayName', ''),
            type=node_data.get('type', ''),
            template_name=node_data.get('templateName', ''),
            phase=node_data.get('phase', ''),
            started_at=node_data.get('startedAt'),
            finished_at=node_data.get('finishedAt'),
            message=node_data.get('message'),
            boundary_id=node_data.get('boundaryID'),
            host_node_name=node_data.get('hostNodeName')
        )
        
        # Extract exit code from outputs if available
        outputs = node_data.get('outputs', {})
        if 'exitCode' in outputs:
            status.exit_code = outputs['exitCode']
            
        node_statuses[node_id] = status
    
    return node_statuses


def get_component_statuses(response: Dict[str, Any]) -> List[NodeStatus]:
    """
    Get only the component (Pod type) statuses, filtering out DAG nodes.
    
    Args:
        response: The full KFP API response dictionary
        
    Returns:
        List of NodeStatus objects for components only
    """
    all_statuses = parse_kfp_response(response)
    return [status for status in all_statuses.values() if status.type == 'Pod']


def get_failed_components(response: Dict[str, Any]) -> List[NodeStatus]:
    """
    Get only the failed components.
    
    Args:
        response: The full KFP API response dictionary
        
    Returns:
        List of NodeStatus objects for failed components
    """
    component_statuses = get_component_statuses(response)
    return [status for status in component_statuses if status.phase == 'Failed']


def get_pipeline_summary(response: Dict[str, Any]) -> Dict[str, Any]:
    """
    Get a summary of the entire pipeline execution.
    
    Args:
        response: The full KFP API response dictionary
        
    Returns:
        Dictionary with pipeline summary information
    """
    workflow_manifest_str = response['pipeline_runtime']['workflow_manifest']
    workflow_manifest = json.loads(workflow_manifest_str)
    
    # Get overall workflow status
    workflow_status = workflow_manifest.get('status', {})
    
    # Get run information
    run_info = response.get('run', {})
    
    # Count nodes by status
    nodes = workflow_status.get('nodes', {})
    status_counts = {}
    component_counts = {}
    
    for node in nodes.values():
        phase = node.get('phase', 'Unknown')
        node_type = node.get('type', 'Unknown')
        
        # Count all nodes
        status_counts[phase] = status_counts.get(phase, 0) + 1
        
        # Count only components (Pods)
        if node_type == 'Pod':
            component_counts[phase] = component_counts.get(phase, 0) + 1
    
    return {
        'pipeline_name': run_info.get('name', ''),
        'run_id': run_info.get('id', ''),
        'overall_status': run_info.get('status', ''),
        'started_at': run_info.get('created_at'),
        'finished_at': run_info.get('finished_at'),
        'workflow_phase': workflow_status.get('phase', ''),
        'progress': workflow_status.get('progress', ''),
        'total_nodes': len(nodes),
        'node_status_counts': status_counts,
        'component_status_counts': component_counts
    }


def print_status_report(response: Dict[str, Any]) -> None:
    """
    Print a formatted status report for the pipeline execution.
    
    Args:
        response: The full KFP API response dictionary
    """
    summary = get_pipeline_summary(response)
    component_statuses = get_component_statuses(response)
    
    print("=" * 60)
    print("PIPELINE EXECUTION REPORT")
    print("=" * 60)
    print(f"Pipeline: {summary['pipeline_name']}")
    print(f"Run ID: {summary['run_id']}")
    print(f"Overall Status: {summary['overall_status']}")
    print(f"Workflow Phase: {summary['workflow_phase']}")
    print(f"Progress: {summary['progress']}")
    print(f"Started: {summary['started_at']}")
    print(f"Finished: {summary['finished_at']}")
    print()
    
    print("COMPONENT STATUS SUMMARY:")
    print("-" * 30)
    for status, count in summary['component_status_counts'].items():
        print(f"{status}: {count}")
    print()
    
    print("DETAILED COMPONENT STATUS:")
    print("-" * 40)
    for status in component_statuses:
        duration = ""
        if status.started_at and status.finished_at:
            # Note: This is a simplified duration calculation
            duration = f" (Duration: {status.started_at} - {status.finished_at})"
        
        message_info = f" - {status.message}" if status.message else ""
        exit_code_info = f" (Exit: {status.exit_code})" if status.exit_code else ""
        
        print(f"• {status.display_name}")
        print(f"  Status: {status.phase}{exit_code_info}{message_info}")
        print(f"  Template: {status.template_name}")
        if status.host_node_name:
            print(f"  Host: {status.host_node_name}")
        print()


# Example usage with your response data
if __name__ == "__main__":
    # Your response data would go here
    sample_response = {
        # ... your response data ...
    }
    
    # Parse and display results
    node_statuses = parse_kfp_response(sample_response)
    component_statuses = get_component_statuses(sample_response)
    failed_components = get_failed_components(sample_response)
    
    print("All node statuses:")
    for node_id, status in node_statuses.items():
        print(f"{node_id}: {status.phase}")
    
    print("\nComponent statuses:")
    for status in component_statuses:
        print(f"{status.display_name}: {status.phase}")
    
    print("\nFailed components:")
    for status in failed_components:
        print(f"{status.display_name}: {status.phase} - {status.message}")
