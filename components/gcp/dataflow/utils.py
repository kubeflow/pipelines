import googleapiclient.discovery as discovery

def create_df_client():
    return discovery.build('dataflow', 'v1b3')