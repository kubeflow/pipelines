import kfp
class TestClient:
    def __init__(self,
        pipeline_url='https://storage.googleapis.com/test-pipeline-version/ff0869f54fd62ca6b4f5c00c13a3977505d6152b/pipeline.zip', 
        create_pipeline_num = 5, 
        create_pipeline_version_num = 5,
        test_num = 0):
        self.pipeline_url = pipeline_url
        self.create_pipeline_num = create_pipeline_num
        self.create_pipeline_version_num = create_pipeline_version_num
        self.test_num=test_num

    def test_create_pipeline(
        self,
        host = '',
        ):
        import kfp
        if host:
            client = kfp.Client(host)
        else:
            client = kfp.Client()
        #create pipeline
        pipeline_ids = []
        for idx in range(self.create_pipeline_num):
            body = {
                'name': 'test_' +str(self.test_num) + '_pipeline_' + str(idx),
                'url':  {"pipeline_url": self.pipeline_url}
            }
            try:
                response = client.pipelines.create_pipeline(body)
                print('pipeline #' + str(idx) + ' created!')
                pipeline_ids.append(response.id)
            except:
                print('Pipeline creation failed at ' + str(idx) + ' pipeline')
                return pipeline_ids

        print('Successfully created {} pipelines!'.format(self.create_pipeline_num))
        self.test_num+=1
        return pipeline_ids

    def test_create_pipeline_versions(
        self,
        pipeline_ids,
        host = '',
        ):

        import kfp
        if host:
            client = kfp.Client(host)
        else:
            client = kfp.Client()
        #create pipeline
        for pipeline_idx, pipeline_id in enumerate(pipeline_ids):
            for version_idx in range(self.create_pipeline_version_num):
                version_name = 'test_' + str(self.test_num) + '_pipeline_' + pipeline_id + '_version_' + str(version_idx)
                version_body = {
                    "name": version_name, \
                    "package_url": {"pipeline_url": self.pipeline_url}, \
                    "resource_references": [{"key": {"id": pipeline_id, "type":3}, "relationship":1}]
                }
                try:
                    response = client.pipelines.create_pipeline_version(version_body)
                    print('version #' + str(version_idx) + ' for pipeline ' + pipeline_id + ' created!')
                except:
                    print('Pipeline creation failed at pipeline #' + pipeline_id + ' version #' + str(version_idx))
                    return
        
        print('Successfully created {} versions each for {} pipelines!'.format(self.create_pipeline_version_num, len(pipeline_ids)))
        return pipeline_ids
            
if __name__ == '__main__':
    import argparse
    parser = argparse.ArgumentParser()
    parser.add_argument('--create_pipeline_num', type = str)
    parser.add_argument('--create_pipeline_version_num', type = str)

    args = parser.parse_args()

    test_client = TestClient(
        create_pipeline_num = args.create_pipeline_num,
        create_pipeline_version_num = args.create_pipeline_version_num
    )
    pipeline_ids = test_client.test_create_pipeline()
    test_client.test_create_pipeline_versions(pipeline_ids)