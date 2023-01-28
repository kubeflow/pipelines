# kubernetes.py
from typing import List, Dict, Optional


def add_pvc(
    task: PipelineTask,
    pvc_name: str,
    volume_name: Optional[str],
    size: str,
    modes: List[str],
    storage_class: Optional[str] = None,
    annotations: Optional[Dict[str, str]] = None,
):
    """Mount a PVC to `task`.
       
       

    Args:
        task (_type_): _description_
        pvc_name (_type_): _description_
        volume_name (_type_): _description_
        size (_type_): _description_
        modes (_type_): _description_
        storage_class (_type_, optional): _description_. Defaults to None.
        annotations (_type_, optional): _description_. Defaults to None.
    """
