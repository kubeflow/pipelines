import os

import pytorch_lightning as pl
import webdataset as wds
from torch.utils.data import DataLoader
from torchvision import transforms


class CIFAR10DataModule(pl.LightningDataModule):
    def __init__(self, **kwargs):
        """
        Initialization of inherited lightning data module
        """
        super(CIFAR10DataModule, self).__init__()

        self.train_dataset = None
        self.valid_dataset = None
        self.test_dataset = None
        self.train_data_loader = None
        self.val_data_loader = None
        self.test_data_loader = None
        self.normalize = transforms.Normalize(mean=[0.5, 0.5, 0.5], std=[0.5, 0.5, 0.5])
        self.valid_transform = transforms.Compose(
            [
                transforms.ToTensor(),
                self.normalize,
            ]
        )

        self.train_transform = transforms.Compose(
            [
                transforms.RandomResizedCrop(32),
                transforms.RandomHorizontalFlip(),
                transforms.ToTensor(),
                self.normalize,
            ]
        )
        self.args = kwargs

    def prepare_data(self):
        """
        Implementation of abstract class
        """

    @staticmethod
    def getNumFiles(input_path):
        return len(os.listdir(input_path)) - 1

    def setup(self, stage=None):
        """
        Downloads the data, parse it and split the data into train, test, validation data

        :param stage: Stage - training or testing
        """

        data_path = self.args.get("train_glob", "/pvc/output/processing")

        train_base_url = data_path + "/train"
        val_base_url = data_path + "/val"
        test_base_url = data_path + "/test"

        train_count = self.getNumFiles(train_base_url)
        val_count = self.getNumFiles(val_base_url)
        test_count = self.getNumFiles(test_base_url)

        train_url = "{}/{}-{}".format(train_base_url, "train", "{0.." + str(train_count) + "}.tar")
        valid_url = "{}/{}-{}".format(val_base_url, "val", "{0.." + str(val_count) + "}.tar")
        test_url = "{}/{}-{}".format(test_base_url, "test", "{0.." + str(test_count) + "}.tar")

        self.train_dataset = (
            wds.Dataset(train_url, handler=wds.warn_and_continue, length=40000 // 40)
            .shuffle(100)
            .decode("pil")
            .rename(image="ppm;jpg;jpeg;png", info="cls")
            .map_dict(image=self.train_transform)
            .to_tuple("image", "info")
            .batched(40)
        )

        self.valid_dataset = (
            wds.Dataset(valid_url, handler=wds.warn_and_continue, length=10000 // 20)
            .shuffle(100)
            .decode("pil")
            .rename(image="ppm", info="cls")
            .map_dict(image=self.valid_transform)
            .to_tuple("image", "info")
            .batched(20)
        )

        self.test_dataset = (
            wds.Dataset(test_url, handler=wds.warn_and_continue, length=10000 // 20)
            .shuffle(100)
            .decode("pil")
            .rename(image="ppm", info="cls")
            .map_dict(image=self.valid_transform)
            .to_tuple("image", "info")
            .batched(20)
        )

    def create_data_loader(self, dataset, batch_size, num_workers):
        return DataLoader(dataset, batch_size=batch_size, num_workers=num_workers)

    def train_dataloader(self):
        """
        :return: output - Train data loader for the given input
        """
        self.train_data_loader = self.create_data_loader(
            self.train_dataset,
            self.args.get("train_batch_size", None),
            self.args.get("train_num_workers", 4),
        )
        return self.train_data_loader

    def val_dataloader(self):
        """
        :return: output - Validation data loader for the given input
        """
        self.val_data_loader = self.create_data_loader(
            self.valid_dataset,
            self.args.get("val_batch_size", None),
            self.args.get("val_num_workers", 4),
        )
        return self.val_data_loader

    def test_dataloader(self):
        """
        :return: output - Test data loader for the given input
        """
        self.test_data_loader = self.create_data_loader(
            self.test_dataset,
            self.args.get("val_batch_size", None),
            self.args.get("val_num_workers", 4),
        )
        return self.test_data_loader
