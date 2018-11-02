package test

//type RunApiTestSuite struct {
//	suite.Suite
//	namespace        string
//	conn             *grpc.ClientConn
//	experimentClient api.ExperimentServiceClient
//	jobClient        api.JobServiceClient
//	pipelineClient   api.PipelineServiceClient
//	runClient        api.RunServiceClient
//}
//
//// Check the namespace have ML pipeline installed and ready
//func (s *RunApiTestSuite) SetupTest() {
//	err := waitForReady(*namespace, *initializeTimeout)
//	if err != nil {
//		glog.Exitf("Failed to initialize test. Error: %s", err.Error())
//	}
//	s.namespace = *namespace
//	s.conn, err = getRpcConnection(s.namespace)
//	if err != nil {
//		glog.Exitf("Failed to get RPC connection. Error: %s", err.Error())
//	}
//	s.experimentClient = api.NewExperimentServiceClient(s.conn)
//	s.jobClient = api.NewJobServiceClient(s.conn)
//	s.pipelineClient = api.NewPipelineServiceClient(s.conn)
//	s.runClient = api.NewRunServiceClient(s.conn)
//}
//
//func (s *RunApiTestSuite) TearDownTest() {
//	s.conn.Close()
//}
//
//func (s *RunApiTestSuite) TestRunApis() {
//	t := s.T()
//	clientSet, err := getKubernetesClient()
//	if err != nil {
//		t.Fatalf("Can't initialize a Kubernete client. Error: %s", err.Error())
//	}
//
//	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
//	defer cancel()
//
//	/* ---------- Upload a pipeline ---------- */
//	pipelineBody, writer := uploadPipelineFileOrFail("resources/hello-world.yaml")
//	response, err := clientSet.RESTClient().Post().
//		AbsPath(fmt.Sprintf(mlPipelineAPIServerBase, s.namespace, "pipelines/upload")).
//		SetHeader("Content-Type", writer.FormDataContentType()).
//		Body(pipelineBody).Do().Raw()
//	assert.Nil(t, err)
//	var helloWorldPipeline api.Pipeline
//	json.Unmarshal(response, &helloWorldPipeline)
//	assert.Equal(t, "hello-world.yaml", helloWorldPipeline.Name)
//
//	/* ---------- Create a new hello world experiment ---------- */
//	createExperimentRequest := &api.CreateExperimentRequest{Experiment: &api.Experiment{Name: "hello world experiment"}}
//	helloWorldExperiment, err := s.experimentClient.CreateExperiment(ctx, createExperimentRequest)
//	assert.Nil(t, err)
//
//	/* ---------- Create a new hello world run by specifying pipeline ID ---------- */
//	requestStartTime := time.Now().Unix()
//	createRunRequest := &api.CreateRunRequest{Run: &api.Run{
//		Name:        "hello world",
//		Description: "this is hello world",
//		PipelineSpec: &api.PipelineSpec{
//			PipelineId: helloWorldPipeline.Id,
//		},
//		ResourceReferences: []*api.ResourceReference{
//			{Key: &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: helloWorldExperiment.Id}},
//		},
//	}}
//	runDetail, err := s.runClient.CreateRun(ctx, createRunRequest)
//	assert.Nil(t, err)
//	s.checkHelloWorldRunDetail(t, runDetail, helloWorldExperiment.Id, helloWorldPipeline.Id, requestStartTime)
//
//	/* ---------- Get hello world run ---------- */
//	runDetail, err = s.runClient.GetRun(ctx, &api.GetRunRequest{RunId: runDetail.Run.Id})
//	assert.Nil(t, err)
//	s.checkHelloWorldRunDetail(t, runDetail, helloWorldExperiment.Id, helloWorldPipeline.Id, requestStartTime)
//
//	/* ---------- Create a new argument parameter experiment ---------- */
//	createExperimentRequest = &api.CreateExperimentRequest{Experiment: &api.Experiment{Name: "argument parameter experiment"}}
//	argParamsExperiment, err := s.experimentClient.CreateExperiment(ctx, createExperimentRequest)
//	assert.Nil(t, err)
//
//	/* ---------- Create a new argument parameter run by uploading workflow manifest ---------- */
//	requestStartTime = time.Now().Unix()
//	argParamsBytes, err := ioutil.ReadFile("resources/arguments-parameters.yaml")
//	assert.Nil(t, err)
//	argParamsBytes, err = yaml.ToJSON(argParamsBytes)
//	assert.Nil(t, err)
//	createRunRequest = &api.CreateRunRequest{Run: &api.Run{
//		Name:        "argument parameter",
//		Description: "this is argument parameter",
//		PipelineSpec: &api.PipelineSpec{
//			WorkflowManifest: string(argParamsBytes),
//			Parameters: []*api.Parameter{
//				{Name: "param1", Value: "goodbye"},
//				{Name: "param2", Value: "world"},
//			},
//		},
//		ResourceReferences: []*api.ResourceReference{
//			{Key: &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: argParamsExperiment.Id}},
//		},
//	}}
//	runDetail, err = s.runClient.CreateRun(ctx, createRunRequest)
//	assert.Nil(t, err)
//	s.checkArgParamsRunDetail(t, runDetail, argParamsExperiment.Id, requestStartTime)
//
//	/* ---------- List all the runs. Both runs should be returned ---------- */
//	listRunsResponse, err := s.runClient.ListRuns(ctx, &api.ListRunsRequest{})
//	assert.Nil(t, err)
//	assert.Equal(t, len(listRunsResponse.Runs), 2)
//
//	/* ---------- List the runs, paginated, default sort ---------- */
//	listRunsResponse, err = s.runClient.ListRuns(ctx, &api.ListRunsRequest{PageSize: 1})
//	assert.Nil(t, err)
//	assert.Equal(t, len(listRunsResponse.Runs), 1)
//	assert.Equal(t, listRunsResponse.Runs[0].Name, "hello world")
//	listRunsResponse, err = s.runClient.ListRuns(ctx, &api.ListRunsRequest{PageSize: 1, PageToken: listRunsResponse.NextPageToken})
//	assert.Nil(t, err)
//	assert.Equal(t, len(listRunsResponse.Runs), 1)
//	assert.Equal(t, listRunsResponse.Runs[0].Name, "argument parameter")
//
//	/* ---------- List the runs, paginated, sort by name ---------- */
//	listRunsResponse, err = s.runClient.ListRuns(ctx, &api.ListRunsRequest{PageSize: 1, SortBy: "name"})
//	assert.Nil(t, err)
//	assert.Equal(t, len(listRunsResponse.Runs), 1)
//	assert.Equal(t, listRunsResponse.Runs[0].Name, "argument parameter")
//	listRunsResponse, err = s.runClient.ListRuns(ctx, &api.ListRunsRequest{PageSize: 1, SortBy: "name", PageToken: listRunsResponse.NextPageToken})
//	assert.Nil(t, err)
//	assert.Equal(t, len(listRunsResponse.Runs), 1)
//	assert.Equal(t, listRunsResponse.Runs[0].Name, "hello world")
//
//	/* ---------- List the runs, sort by unsupported field ---------- */
//	_, err = s.runClient.ListRuns(ctx, &api.ListRunsRequest{PageSize: 2, SortBy: "description"})
//	assert.NotNil(t, err)
//	assert.Contains(t, err.Error(), "InvalidArgument")
//
//	/* ---------- List runs for hello world experiment. One run should be returned ---------- */
//	listRunsResponse, err = s.runClient.ListRuns(ctx, &api.ListRunsRequest{
//		ResourceReferenceKey: &api.ResourceKey{
//			Type: api.ResourceType_EXPERIMENT, Id: helloWorldExperiment.Id}})
//	assert.Nil(t, err)
//	assert.Equal(t, len(listRunsResponse.Runs), 1)
//	assert.Equal(t, listRunsResponse.Runs[0].Name, "hello world")
//}
//
//func (s *RunApiTestSuite) checkHelloWorldRunDetail(t *testing.T, runDetail *api.RunDetail, experimentId string, pipelineId string, requestStartTime int64) {
//	assert.NotNil(t, runDetail)
//	assert.NotNil(t, runDetail.Run)
//	assert.NotNil(t, runDetail.Run.PipelineSpec)
//	assert.NotNil(t, runDetail.PipelineRuntime)
//	assert.NotNil(t, runDetail.Run.CreatedAt)
//
//	// Check workflow manifest is not empty
//	assert.Contains(t, runDetail.Run.PipelineSpec.WorkflowManifest, "whalesay")
//	// Check runtime workflow manifest is not empty
//	assert.Contains(t, runDetail.PipelineRuntime.WorkflowManifest, "whalesay")
//	assert.True(t, runDetail.Run.CreatedAt.Seconds > requestStartTime)
//
//	runDetail.Run.PipelineSpec.WorkflowManifest = ""
//	runDetail.Run.Status = ""
//	expectedRun := api.Run{
//		Id:          runDetail.Run.Id,
//		Name:        "hello world",
//		Description: "this is hello world",
//		PipelineSpec: &api.PipelineSpec{
//			PipelineId: pipelineId,
//		},
//		ResourceReferences: []*api.ResourceReference{
//			{Key: &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experimentId}},
//		},
//		CreatedAt: &timestamp.Timestamp{Seconds: runDetail.Run.CreatedAt.Seconds},
//	}
//	assert.Equal(t, expectedRun, &runDetail.Run)
//}
//
//func (s *RunApiTestSuite) checkArgParamsRunDetail(t *testing.T, runDetail *api.RunDetail, experimentId string, requestStartTime int64) {
//	argParamsBytes, err := ioutil.ReadFile("resources/arguments-parameters.yaml")
//	assert.Nil(t, err)
//	argParamsBytes, err = yaml.ToJSON(argParamsBytes)
//	assert.Nil(t, err)
//	// Check runtime workflow manifest is not empty
//	assert.Contains(t, runDetail.PipelineRuntime.WorkflowManifest, "arguments-parameters-")
//	assert.True(t, runDetail.Run.CreatedAt.Seconds > requestStartTime)
//
//	runDetail.Run.Status = ""
//	expectedRun := api.Run{
//		Id:          runDetail.Run.Id,
//		Name:        "hello world",
//		Description: "this is hello world",
//		PipelineSpec: &api.PipelineSpec{
//			WorkflowManifest: string(argParamsBytes),
//		},
//		ResourceReferences: []*api.ResourceReference{
//			{Key: &api.ResourceKey{Type: api.ResourceType_EXPERIMENT, Id: experimentId}},
//		},
//		CreatedAt: &timestamp.Timestamp{Seconds: runDetail.Run.CreatedAt.Seconds},
//	}
//	assert.Equal(t, expectedRun, &runDetail.Run)
//}
//
//func TestRunApi(t *testing.T) {
//	//suite.Run(t, new(RunApiTestSuite))
//}
