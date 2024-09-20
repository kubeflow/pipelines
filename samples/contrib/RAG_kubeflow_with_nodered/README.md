# RAG Component

This is a Kubeflow pipeline component that retrieves answers from a provided document using a Retrieval-Augmented Generation (RAG) approach. The component utilizes various libraries such as `langchain`, `pinecone`, and `transformers` to process a PDF document, create embeddings, and generate answers based on the retrieved content.

## Prework

### Installation

```
!pip install kfp
```

### Pinecone 

In this example, we use pinecone to create a vector database. Create an account and create an index, enter the index name, dimensions, enter 384 and then create the index

![](https://github.com/sefgsefg/RAG_kubeflow/blob/main/Pinecone_create_index.png)

Then create an API key(token)

![](https://github.com/sefgsefg/RAG_kubeflow/blob/main/Pinecone_create_APIkey.png)
---

### Huggingface

If the model you use have to sigh the agreement(like meta LLAMA models), you will have to create a huggingface API key.
Go to huggingface setting and find Access Tokens, then click "New token" to create a key for `READ`

![](https://github.com/sefgsefg/RAG_kubeflow/blob/main/huggingface_token.png)
---

### Adjust the parameters of the code

Now go to input your parameters in code.
Type these parameters. For PDF or model you can use "EX" as suggested

![](https://github.com/sefgsefg/RAG_kubeflow/blob/main/Code_para.png)

And there is a parameter call `load_in_8bit` in function `AutoModelForCausalLM.from_pretrained`, if your platform doesn't use GPU, just comment it.

Reference
---
https://github.com/MuhammadMoinFaisal/LargeLanguageModelsProjects/tree/main/Chat_with_Multiple_PDF_llama2_Pinecone
