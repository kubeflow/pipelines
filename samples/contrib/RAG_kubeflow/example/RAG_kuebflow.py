from kfp import dsl

@dsl.component(base_image="python:3.10.12",packages_to_install = [
    'requests',
    'langchain',
    'langchain-community',
    'pypdf',
    'pinecone-client',
    'langchain_pinecone',
    'sentence_transformers',
    'googletrans==3.1.0a0',
    'transformers',
    'torch',
    'bitsandbytes',
    'accelerate',
    'urllib3==1.26.15'
])
def RAG(questions:str):
    import torch
    import requests
    from langchain.text_splitter import RecursiveCharacterTextSplitter
    from langchain_community.vectorstores import Pinecone
    from langchain_community.document_loaders import PyPDFLoader, OnlinePDFLoader
    import os
    import sys
    from pinecone import Pinecone
    from langchain_pinecone import PineconeVectorStore
    from langchain.embeddings import HuggingFaceEmbeddings
    
    import pinecone
    from transformers import AutoTokenizer, AutoModelForCausalLM
    from transformers import pipeline
    from langchain import HuggingFacePipeline, PromptTemplate
    from langchain.chains import RetrievalQA
    from huggingface_hub import login

    
    url = '<change_yours>'#Ex: "https://arxiv.org/pdf/2207.02696.pdf", a yolov7 pdf
    data_name = '<change_yours>'#Ex: "/yolov7.pdf"

    huggingface_token = '<change_yours>'
    embedding_model = '<change_yours>'#Ex: "sentence-transformers/all-MiniLM-L6-v2"
    LLM_model = '<change_yours>'#Ex: "openai-community/gpt2" or "meta-llama/Llama-2-7b-chat-hf"
    Pinecone_token = '<change_yours>'
    Pinecone_index = '<change_yours>'
    
    
    r = requests.get(url, stream=True)
    with open(data_name, 'wb') as fd:
        fd.write(r.content)
    loader = PyPDFLoader(data_name)
    data = loader.load()
    
    text_splitter = RecursiveCharacterTextSplitter(chunk_size=500, chunk_overlap=20)
    docs = text_splitter.split_documents(data)
    
    embeddings = HuggingFaceEmbeddings(model_name=embedding_model)
    PINE_KEY = os.environ.get("PINECONE_API_KEY", Pinecone_token)
    pc = Pinecone(api_key=Pinecone_token)
    os.environ["PINECONE_API_KEY"] = Pinecone_token
    index = pc.Index(Pinecone_index)
    index_name = Pinecone_index
    docsearch = PineconeVectorStore.from_documents(docs, embedding=embeddings, index_name=index_name)

    
    login(huggingface_token)
    
    
    tokenizer = AutoTokenizer.from_pretrained(LLM_model)
    model = AutoModelForCausalLM.from_pretrained(LLM_model,
                        device_map='auto',
                        torch_dtype=torch.float16,
                        use_auth_token=True,
                        load_in_8bit=True # If you don't use GPU, comment this parameter
                         )
    model = pipeline("text-generation",
                model=model,
                tokenizer= tokenizer,
                torch_dtype=torch.bfloat16,
                device_map="auto",
                max_new_tokens = 512,
                do_sample=True,
                top_k=30,
                num_return_sequences=3,
                eos_token_id=tokenizer.eos_token_id
                )
    
    llm=HuggingFacePipeline(pipeline=model, model_kwargs={'temperature':0.1})
    
    SYSTEM_PROMPT = """Use the following pieces of context to answer the question at the end.
    If you don't know the answer, just say that you don't know, don't try to make up an answer."""
    B_INST, E_INST = "[INST]", "[/INST]"
    B_SYS, E_SYS = "<<SYS>>\n", "\n<</SYS>>\n\n"
    
    SYSTEM_PROMPT = B_SYS + SYSTEM_PROMPT + E_SYS
    instruction = """
    {context}

    Question: {question}
    """
    template = B_INST + SYSTEM_PROMPT + instruction + E_INST
    
    prompt = PromptTemplate(template=template, input_variables=["context", "question"])
    
    qa_chain = RetrievalQA.from_chain_type(
    llm=llm,
    chain_type="stuff",
    retriever=docsearch.as_retriever(search_kwargs={"k":3}),
    return_source_documents=True,
    chain_type_kwargs={"prompt": prompt},)
    
    question = questions
    result = qa_chain(question)
    
    print(result['result'])

@dsl.pipeline
def rag_pipeline(recipient: str):
    rag_task = RAG(questions = recipient).set_gpu_limit(1)

from kfp import compiler

compiler.Compiler().compile(rag_pipeline, 'RAG_pipeline.yaml')
