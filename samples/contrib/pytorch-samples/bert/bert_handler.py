import json
import logging
import os
from captum.attr import IntegratedGradients
from captum.attr import visualization
import numpy as np
import torch
from transformers import BertTokenizer
from ts.torch_handler.base_handler import BaseHandler
from bert_train import BertNewsClassifier
from wrapper import AGNewsmodelWrapper
import torch.nn.functional as F

logger = logging.getLogger(__name__)


class NewsClassifierHandler(BaseHandler):
    """
    NewsClassifierHandler class. This handler takes a review / sentence
    and returns the label as either world / sports / business /sci-tech
    """

    def __init__(self):
        self.model = None
        self.mapping = None
        self.device = None
        self.initialized = False
        self.class_mapping_file = None
        self.VOCAB_FILE = None

    def initialize(self, ctx):
        """
        First try to load torchscript else load eager mode state_dict based model

        :param ctx: System properties
        """

        properties = ctx.system_properties
        self.device = torch.device(
            "cuda:" + str(properties.get("gpu_id")) if torch.cuda.is_available() else "cpu"
        )
        model_dir = properties.get("model_dir")

        # Read model serialize/pt file
        model_pt_path = os.path.join(model_dir, "bert.pth")
        # Read model definition file
        model_def_path = os.path.join(model_dir, "bert_train.py")
        if not os.path.isfile(model_def_path):
            raise RuntimeError("Missing the model definition file")

        self.VOCAB_FILE = os.path.join(model_dir, "bert-base-uncased-vocab.txt")
        if not os.path.isfile(self.VOCAB_FILE):
            raise RuntimeError("Missing the vocab file")

        self.class_mapping_file = os.path.join(model_dir, "index_to_name.json")

        state_dict = torch.load(model_pt_path, map_location=self.device)
        self.model = BertNewsClassifier()
        self.model.load_state_dict(state_dict, strict=False)
        self.model.to(self.device)
        self.model.eval()

        logger.debug("Model file %s loaded successfully", model_pt_path)
        self.initialized = True

    def preprocess(self, data):
        """
        Receives text in form of json and converts it into an encoding for the inference stage

        :param data: Input to be passed through the layers for prediction

        :return: output - preprocessed encoding
        """

        text = data[0].get("data")
        if text is None:
            text = data[0].get("body")

        self.text = text
        self.tokenizer = BertTokenizer(self.VOCAB_FILE)  # .from_pretrained("bert-base-cased")
        self.input_ids = torch.tensor([self.tokenizer.encode(self.text, add_special_tokens=True)])
        return self.input_ids

    def inference(self, input_ids):
        """
        Predict the class  for a review / sentence whether
        it is belong to world / sports / business /sci-tech
        :param encoding: Input encoding to be passed through the layers for prediction

        :return: output - predicted output
        """
        inputs = self.input_ids.to(self.device)
        self.outputs = self.model.forward(inputs)
        self.out = np.argmax(self.outputs.cpu().detach())
        return [self.out.item()]

    def postprocess(self, inference_output):
        """
        Does postprocess after inference to be returned to user

        :param inference_output: Output of inference

        :return: output - Output after post processing
        """
        if os.path.exists(self.class_mapping_file):
            with open(self.class_mapping_file) as json_file:
                data = json.load(json_file)
            inference_output = json.dumps(data[str(inference_output[0])])
            return [inference_output]

        return inference_output

    def add_attributions_to_visualizer(
        self,
        attributions,
        tokens,
        pred_prob,
        pred_class,
        true_class,
        attr_class,
        delta,
        vis_data_records,
    ):
        attributions = attributions.sum(dim=2).squeeze(0)
        attributions = attributions / torch.norm(attributions)
        attributions = attributions.cpu().detach().numpy()

        # storing couple samples in an array for visualization purposes
        vis_data_records.append(
            visualization.VisualizationDataRecord(
                attributions,
                pred_prob,
                pred_class,
                true_class,
                attr_class,
                attributions.sum(),
                tokens,
                delta,
            )
        )

    def score_func(self, o):
        output = F.softmax(o, dim=1)
        pre_pro = np.argmax(output.cpu().detach())
        return pre_pro

    def summarize_attributions(self, attributions):
        """Summarises the attribution across multiple runs
        Args:
            attributions ([list): attributions from the Integrated Gradients
        Returns:
            list : Returns the attributions after normalizing them.
        """
        attributions = attributions.sum(dim=-1).squeeze(0)
        attributions = attributions / torch.norm(attributions)
        return attributions

    def explain_handle(self, model_wraper, text, target=1):
        """Captum explanations handler
        Args:
            data_preprocess (Torch Tensor): Preprocessed data to be used for captum
            raw_data (list): The unprocessed data to get target from the request
        Returns:
            dict : A dictionary response with the explanations response."""
        vis_data_records_base = []
        model_wrapper = AGNewsmodelWrapper(self.model)
        tokenizer = BertTokenizer(self.VOCAB_FILE)
        model_wrapper.eval()
        model_wrapper.zero_grad()
        input_ids = torch.tensor([tokenizer.encode(self.text, add_special_tokens=True)])
        input_embedding_test = model_wrapper.model.bert_model.embeddings(input_ids)
        preds = model_wrapper(input_embedding_test)
        out = np.argmax(preds.cpu().detach(), axis=1)
        out = out.item()
        ig_1 = IntegratedGradients(model_wrapper)
        attributions, delta = ig_1.attribute(
            input_embedding_test, n_steps=500, return_convergence_delta=True, target=1
        )
        tokens = tokenizer.convert_ids_to_tokens(input_ids[0].numpy().tolist())
        feature_imp_dict = {}
        feature_imp_dict["words"] = tokens
        attributions_sum = self.summarize_attributions(attributions)
        feature_imp_dict["importances"] = attributions_sum.tolist()
        feature_imp_dict["delta"] = delta[0].tolist()
        return [feature_imp_dict]
