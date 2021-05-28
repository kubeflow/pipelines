"""This module executues the training process and saves the model to checkpoint dir."""

import os
from argparse import Namespace
import pytorch_lightning as pl
import torch


class Executor:
    """Initializes the model training.
    This is called at the trainer Component to carry the training operation.
    """

    def __init__(self):
        pass

    def Do(
        self,
        model_class,
        data_module_class=None,
        data_module_args=None,
        module_file_args=None,
        trainer_args=None,
    ):
        """
        This function of the Executor invokes the PyTorch Lightning training loop.
        In this step the data module and model is set up and then the model is fitted and tested.
        At the end of the training, the model state_dict is saved in the given checkpoint directory.

        :param model_class : The name of modle class.
        :param data_module_class : The name of the data module.
        :param data_module_args : The arguments of the data module, viz num workers, train glob etc.
        :param module_file_args : The arguments of the model class, viz lr , weight_decay, etc.
        :param trainer_args : These arguments of the trainer includes max_epochs, checkpoints etc.

        :return : returns the trainer object of PyTorch Lightning
        """

        if data_module_class:
            data_module = data_module_class(**data_module_args if data_module_args else {})
            data_module.prepare_data()
            data_module.setup(stage="fit")
            model = model_class(**module_file_args if module_file_args else {})

            if (not module_file_args) and (not trainer_args):
                raise ValueError("Both module file args and trainer args cannot be empty")

            if not isinstance(trainer_args, dict):
                raise TypeError(f"trainer_args must be a dict")

            trainer_args.update(module_file_args)
            parser = Namespace(**trainer_args)
            trainer = pl.Trainer.from_argparse_args(parser)

            trainer.fit(model, data_module)
            trainer.test()

            if "checkpoint_dir" in module_file_args:
                model_save_path = module_file_args["checkpoint_dir"]
            else:
                model_save_path = "/tmp"

            if "model_name" in module_file_args:
                model_name = module_file_args["model_name"]
            else:
                model_name = "model_state_dict.pth"

            model_save_path = os.path.join(model_save_path, model_name)
            print("Saving model to {}".format(model_save_path))
            torch.save(model.state_dict(), model_save_path)

            return trainer
        else:
            raise NotImplementedError(
                "Data module class is mandatory. "
                "User defined training module is yet to be supported."
            )
