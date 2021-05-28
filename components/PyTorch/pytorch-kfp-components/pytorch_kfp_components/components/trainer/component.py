"""This module is the component of the pipeline for the complete training of the models.
Calls the Executor for the PyTorch Lightning training to start."""
import inspect
import importlib
from typing import Optional, Dict
from pytorch_kfp_components.components.trainer.executor import Executor


class Trainer:
    """Initializes the Trainer class."""

    def __init__(
        self,
        module_file: Optional = None,
        data_module_file: Optional = None,
        data_module_args: Optional[Dict] = None,
        module_file_args: Optional[Dict] = None,
        trainer_args: Optional[Dict] = None,
    ):
        """
        Initializes the PyTorch Lightning training process.

        :param module_file : The module to inherit the model class for training.
        :param data_module_file : The module from which the data module class is inherited.
        :param data_module_args : The arguments of the data module.
        :param module_file_args : The arguments of the model class.
        :param trainer_args : These arguments are specific to the pytorch lightning trainer.
        """
        if not module_file:
            raise ValueError(f"module_file cannot be {module_file}")

        if not data_module_file:
            raise ValueError(f"data_module_file cannot be {data_module_file}")

        if module_file and data_module_file:
            # Both module file and data module file are present

            model_class = None
            data_module_class = None

            class_module = importlib.import_module(module_file.split(".")[0])
            data_module = importlib.import_module(data_module_file.split(".")[0])

            for cls in inspect.getmembers(
                class_module,
                lambda member: inspect.isclass(member)
                and member.__module__ == class_module.__name__,
            ):
                model_class = cls[1]

            if not model_class:
                raise ValueError(f"Unable to load module_file - {module_file}")

            for cls in inspect.getmembers(
                data_module,
                lambda member: inspect.isclass(member)
                and member.__module__ == data_module.__name__,
            ):
                data_module_class = cls[1]

            if not data_module_class:
                raise ValueError(f"Unable to load data_module_file - {data_module_file}")

            self.ptl_trainer = Executor().Do(
                model_class=model_class,
                data_module_class=data_module_class,
                data_module_args=data_module_args,
                module_file_args=module_file_args,
                trainer_args=trainer_args,
            )
        else:
            raise NotImplementedError(
                "Module file and Datamodule file are mandatory. "
                "Custom training methods are yet to be implemented"
            )
