/*
 * Copyright 2026 The Kubeflow Authors
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 * https://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

import { render, screen, waitFor, within } from '@testing-library/react';
import { vi, Mock } from 'vitest';
import { Api } from 'src/mlmd/library';
import * as MlmdUtils from 'src/mlmd/MlmdUtils';
import {
  Artifact,
  ArtifactType,
  Context,
  Event,
  Execution,
  ExecutionType,
  GetArtifactsByIDResponse,
  GetArtifactTypesResponse,
  GetEventsByExecutionIDsResponse,
  Value,
} from 'src/third_party/mlmd';
import {
  GetExecutionsByIDResponse,
  GetExecutionTypesByIDResponse,
} from 'src/third_party/mlmd/generated/ml_metadata/proto/metadata_store_service_pb';
import { RoutePage, RouteParams } from 'src/components/Router';
import TestUtils, { testBestPractices } from 'src/TestUtils';
import ExecutionDetails, {
  ExecutionDetailsContent,
  parseEventsByType,
} from 'src/pages/ExecutionDetails';
import { CommonTestWrapper } from 'src/TestWrapper';

testBestPractices();

function buildExecution(id = 1): Execution {
  const execution = new Execution();
  execution.setId(id);
  execution.setTypeId(10);
  const nameValue = new Value();
  nameValue.setStringValue('test-execution');
  execution.getCustomPropertiesMap().set('display_name', nameValue);
  return execution;
}

function buildExecutionType(id = 10): ExecutionType {
  const executionType = new ExecutionType();
  executionType.setId(id);
  executionType.setName('system.ContainerExecution');
  return executionType;
}

function buildEvent(type: Event.Type, artifactId: number, name?: string): Event {
  const event = new Event();
  event.setType(type);
  event.setArtifactId(artifactId);
  if (name) {
    const step = new Event.Path.Step();
    step.setKey(name);
    const path = new Event.Path();
    path.addSteps(step);
    event.setPath(path);
  }
  return event;
}

function buildEventsResponse(events: Event[]): GetEventsByExecutionIDsResponse {
  const response = new GetEventsByExecutionIDsResponse();
  for (const event of events) {
    response.addEvents(event);
  }
  return response;
}

function buildExecutionsByIDResponse(executions: Execution[]): GetExecutionsByIDResponse {
  const response = new GetExecutionsByIDResponse();
  response.setExecutionsList(executions);
  return response;
}

function buildExecutionTypesByIDResponse(types: ExecutionType[]): GetExecutionTypesByIDResponse {
  const response = new GetExecutionTypesByIDResponse();
  response.setExecutionTypesList(types);
  return response;
}

describe('parseEventsByType', () => {
  it('returns all empty arrays when response is null', () => {
    const result = parseEventsByType(null);
    expect(result[Event.Type.INPUT]).toEqual([]);
    expect(result[Event.Type.OUTPUT]).toEqual([]);
    expect(result[Event.Type.DECLARED_INPUT]).toEqual([]);
    expect(result[Event.Type.DECLARED_OUTPUT]).toEqual([]);
    expect(result[Event.Type.UNKNOWN]).toEqual([]);
  });

  it('groups events correctly into INPUT/OUTPUT/DECLARED_INPUT/DECLARED_OUTPUT buckets', () => {
    const response = buildEventsResponse([
      buildEvent(Event.Type.INPUT, 1),
      buildEvent(Event.Type.OUTPUT, 2),
      buildEvent(Event.Type.DECLARED_INPUT, 3),
      buildEvent(Event.Type.DECLARED_OUTPUT, 4),
      buildEvent(Event.Type.INPUT, 5),
    ]);

    const result = parseEventsByType(response);
    expect(result[Event.Type.INPUT]).toHaveLength(2);
    expect(result[Event.Type.OUTPUT]).toHaveLength(1);
    expect(result[Event.Type.DECLARED_INPUT]).toHaveLength(1);
    expect(result[Event.Type.DECLARED_OUTPUT]).toHaveLength(1);
    expect(result[Event.Type.UNKNOWN]).toHaveLength(0);
  });

  it('places events with default type (UNKNOWN) into the UNKNOWN bucket', () => {
    const eventDefaultType = new Event();
    eventDefaultType.setArtifactId(100);

    const response = buildEventsResponse([eventDefaultType]);
    const result = parseEventsByType(response);

    expect(result[Event.Type.UNKNOWN]).toHaveLength(1);
  });
});

describe('ExecutionDetailsContent', () => {
  let getExecutionsByIDSpy: Mock;
  let getEventsByExecutionIDsSpy: Mock;
  let getExecutionTypesByIDSpy: Mock;
  let getArtifactTypesSpy: Mock;
  let getArtifactsByIDSpy: Mock;
  let getContextByExecutionSpy: ReturnType<typeof vi.spyOn>;
  let onErrorSpy: Mock;
  let onTitleUpdateSpy: Mock;

  beforeEach(() => {
    onErrorSpy = vi.fn();
    onTitleUpdateSpy = vi.fn();
    getExecutionsByIDSpy = vi.spyOn(Api.getInstance().metadataStoreService, 'getExecutionsByID');
    getEventsByExecutionIDsSpy = vi.spyOn(
      Api.getInstance().metadataStoreService,
      'getEventsByExecutionIDs',
    );
    getExecutionTypesByIDSpy = vi.spyOn(
      Api.getInstance().metadataStoreService,
      'getExecutionTypesByID',
    );
    getArtifactTypesSpy = vi.spyOn(Api.getInstance().metadataStoreService, 'getArtifactTypes');
    getArtifactsByIDSpy = vi.spyOn(Api.getInstance().metadataStoreService, 'getArtifactsByID');
    getContextByExecutionSpy = vi.spyOn(MlmdUtils, 'getContextByExecution');

    getArtifactTypesSpy.mockResolvedValue(new GetArtifactTypesResponse());
    getArtifactsByIDSpy.mockResolvedValue(new GetArtifactsByIDResponse());
    getContextByExecutionSpy.mockReturnValue(new Promise(() => {}));
  });

  function renderContent(id = 1) {
    return render(
      <CommonTestWrapper>
        <ExecutionDetailsContent id={id} onError={onErrorSpy} onTitleUpdate={onTitleUpdateSpy} />
      </CommonTestWrapper>,
    );
  }

  function mockSuccessfulLoad(execution?: Execution, executionType?: ExecutionType) {
    getExecutionsByIDSpy.mockResolvedValue(
      buildExecutionsByIDResponse([execution || buildExecution()]),
    );
    getEventsByExecutionIDsSpy.mockResolvedValue(buildEventsResponse([]));
    getExecutionTypesByIDSpy.mockResolvedValue(
      buildExecutionTypesByIDResponse([executionType || buildExecutionType()]),
    );
  }

  function mockLoadWithEvents(events: Event[], execution?: Execution) {
    getExecutionsByIDSpy.mockResolvedValue(
      buildExecutionsByIDResponse([execution || buildExecution()]),
    );
    getEventsByExecutionIDsSpy.mockResolvedValue(buildEventsResponse(events));
    getExecutionTypesByIDSpy.mockResolvedValue(
      buildExecutionTypesByIDResponse([buildExecutionType()]),
    );
  }

  it('shows CircularProgress while execution data is loading', () => {
    getExecutionsByIDSpy.mockReturnValue(new Promise(() => {}));
    getEventsByExecutionIDsSpy.mockReturnValue(new Promise(() => {}));
    getArtifactTypesSpy.mockReturnValue(new Promise(() => {}));

    renderContent();

    expect(screen.getByRole('progressbar')).toBeInTheDocument();
  });

  it.each([NaN, -1])('shows page error when execution ID is invalid (%s)', async (id) => {
    renderContent(id);

    await waitFor(() => {
      expect(onErrorSpy).toHaveBeenCalledWith(
        expect.stringContaining('Invalid execution id'),
        expect.any(Error),
        'error',
        expect.any(Function),
      );
    });
  });

  it('shows page error when no execution found for the given ID', async () => {
    getExecutionsByIDSpy.mockResolvedValue(buildExecutionsByIDResponse([]));
    getEventsByExecutionIDsSpy.mockResolvedValue(buildEventsResponse([]));

    renderContent(999);

    await waitFor(() => {
      expect(onErrorSpy).toHaveBeenCalledWith(
        expect.stringContaining('No execution identified by id: 999'),
        undefined,
        'error',
        expect.any(Function),
      );
    });
  });

  it('shows page error when multiple executions found', async () => {
    getExecutionsByIDSpy.mockResolvedValue(
      buildExecutionsByIDResponse([buildExecution(1), buildExecution(2)]),
    );
    getEventsByExecutionIDsSpy.mockResolvedValue(buildEventsResponse([]));

    renderContent(1);

    await waitFor(() => {
      expect(onErrorSpy).toHaveBeenCalledWith(
        expect.stringContaining('Found multiple executions with ID: 1'),
        undefined,
        'error',
        expect.any(Function),
      );
    });
  });

  it('shows page error when execution type cannot be found', async () => {
    getExecutionsByIDSpy.mockResolvedValue(buildExecutionsByIDResponse([buildExecution()]));
    getEventsByExecutionIDsSpy.mockResolvedValue(buildEventsResponse([]));
    getExecutionTypesByIDSpy.mockResolvedValue(buildExecutionTypesByIDResponse([]));

    renderContent();

    await waitFor(() => {
      expect(onErrorSpy).toHaveBeenCalledWith(
        expect.stringContaining('Cannot find execution type with id'),
        undefined,
        'error',
        expect.any(Function),
      );
    });
  });

  it('shows page error when multiple execution types are found', async () => {
    getExecutionsByIDSpy.mockResolvedValue(buildExecutionsByIDResponse([buildExecution()]));
    getEventsByExecutionIDsSpy.mockResolvedValue(buildEventsResponse([]));
    getExecutionTypesByIDSpy.mockResolvedValue(
      buildExecutionTypesByIDResponse([buildExecutionType(10), buildExecutionType(11)]),
    );

    renderContent();

    await waitFor(() => {
      expect(onErrorSpy).toHaveBeenCalledWith(
        expect.stringContaining('More than one execution type found with id'),
        undefined,
        'error',
        expect.any(Function),
      );
    });
  });

  it('shows service error when load fails with a ServiceError', async () => {
    getExecutionsByIDSpy.mockRejectedValue({ message: 'GRPC unavailable', code: 14 });
    getEventsByExecutionIDsSpy.mockRejectedValue({ message: 'GRPC unavailable', code: 14 });

    renderContent();

    await waitFor(() => {
      expect(onErrorSpy).toHaveBeenCalledWith(
        expect.stringContaining('GRPC unavailable'),
        undefined,
        'error',
        expect.any(Function),
      );
    });
  });

  it('shows fallback error when load fails with a non-service error', async () => {
    getExecutionsByIDSpy.mockRejectedValue(undefined);
    getEventsByExecutionIDsSpy.mockRejectedValue(undefined);

    renderContent();

    await waitFor(() => {
      expect(onErrorSpy).toHaveBeenCalledWith(
        'Error: failed to load execution details.',
        expect.any(Error),
        'error',
        expect.any(Function),
      );
    });
  });

  it('shows warning when artifact types fetch fails', async () => {
    mockSuccessfulLoad();
    getArtifactTypesSpy.mockRejectedValue(new Error('artifact types unavailable'));

    renderContent();

    await waitFor(() => {
      expect(onErrorSpy).toHaveBeenCalledWith(
        'Failed to fetch artifact types',
        expect.any(Error),
        'warning',
        expect.any(Function),
      );
    });
  });

  it('renders ResourceInfo with execution type name after data loads', async () => {
    mockSuccessfulLoad();

    renderContent();

    await waitFor(() => {
      expect(screen.getByText('Reference')).toBeInTheDocument();
    });
    expect(onTitleUpdateSpy).toHaveBeenCalledWith('test-execution');
  });

  it('uses empty string as type name when execution type name is empty', async () => {
    const emptyNameType = new ExecutionType();
    emptyNameType.setId(10);
    emptyNameType.setName('');
    mockSuccessfulLoad(undefined, emptyNameType);

    renderContent();

    await waitFor(() => {
      expect(screen.getByText('Reference')).toBeInTheDocument();
    });
    expect(screen.queryByText('system.ContainerExecution')).not.toBeInTheDocument();
  });

  it('renders IO section headers when events are present', async () => {
    mockLoadWithEvents([buildEvent(Event.Type.INPUT, 100), buildEvent(Event.Type.OUTPUT, 200)]);

    renderContent();

    await waitFor(() => {
      expect(screen.getByText('Inputs')).toBeInTheDocument();
    });
    expect(screen.getByText('Outputs')).toBeInTheDocument();
  });

  describe('IO sections (via ExecutionDetailsContent)', () => {
    it('renders a table row per event', async () => {
      mockLoadWithEvents([
        buildEvent(Event.Type.INPUT, 10),
        buildEvent(Event.Type.INPUT, 20),
        buildEvent(Event.Type.OUTPUT, 30),
      ]);

      renderContent();

      await waitFor(() => {
        expect(screen.getByText('Inputs')).toBeInTheDocument();
      });

      const inputsSection = screen.getByText('Inputs').closest('section')!;
      const inputRows = within(inputsSection as HTMLElement).getAllByRole('row');
      expect(inputRows).toHaveLength(3); // 1 header + 2 data rows

      const outputsSection = screen.getByText('Outputs').closest('section')!;
      const outputRows = within(outputsSection as HTMLElement).getAllByRole('row');
      expect(outputRows).toHaveLength(2); // 1 header + 1 data row
    });
  });

  it('renders artifact type name in SectionIO table rows', async () => {
    const artifactType = new ArtifactType();
    artifactType.setId(5);
    artifactType.setName('system.Dataset');
    const typesResponse = new GetArtifactTypesResponse();
    typesResponse.setArtifactTypesList([artifactType]);
    getArtifactTypesSpy.mockResolvedValue(typesResponse);

    const artifact = new Artifact();
    artifact.setId(10);
    artifact.setTypeId(5);
    artifact.setUri('gs://bucket/data');
    getArtifactsByIDSpy.mockResolvedValue(
      new GetArtifactsByIDResponse().setArtifactsList([artifact]),
    );

    mockLoadWithEvents([buildEvent(Event.Type.INPUT, 10, 'my-dataset')]);

    renderContent();

    await waitFor(() => {
      expect(screen.getByText('system.Dataset')).toBeInTheDocument();
    });
    expect(screen.getByText('gs://bucket/data')).toBeInTheDocument();
  });

  it('renders empty URI cell when artifact has no URI set', async () => {
    const artifact = new Artifact();
    artifact.setId(10);
    getArtifactsByIDSpy.mockResolvedValue(
      new GetArtifactsByIDResponse().setArtifactsList([artifact]),
    );
    mockLoadWithEvents([buildEvent(Event.Type.INPUT, 10, 'no-uri-artifact')]);

    renderContent();

    await waitFor(() => {
      expect(screen.getByRole('link', { name: 'no-uri-artifact' })).toBeInTheDocument();
    });

    const inputsSection = screen.getByText('Inputs').closest('section')!;
    const dataRows = within(inputsSection as HTMLElement).getAllByRole('row');
    const dataRow = dataRows[1];
    const cells = within(dataRow).getAllByRole('cell');
    expect(cells[3]).toHaveTextContent('');
  });

  describe('artifact linking (via ExecutionDetailsContent)', () => {
    it('renders artifact name as a link when artifact ID exists', async () => {
      const artifact = new Artifact();
      artifact.setId(10);
      artifact.setUri('gs://bucket/path');

      mockLoadWithEvents([buildEvent(Event.Type.INPUT, 10, 'my-artifact')]);
      getArtifactsByIDSpy.mockResolvedValue(
        new GetArtifactsByIDResponse().setArtifactsList([artifact]),
      );

      renderContent();

      await waitFor(() => {
        const link = screen.getByRole('link', { name: 'my-artifact' });
        expect(link).toHaveAttribute('href', '/artifacts/10');
      });
    });

    it('renders plain text when artifact ID is zero', async () => {
      mockLoadWithEvents([buildEvent(Event.Type.OUTPUT, 0)]);

      renderContent();

      await waitFor(() => {
        expect(screen.getByText('Outputs')).toBeInTheDocument();
      });

      const outputsSection = screen.getByText('Outputs').closest('section')!;
      expect(within(outputsSection as HTMLElement).queryAllByRole('link')).toHaveLength(0);
    });
  });

  describe('ExecutionReference', () => {
    it('renders a pipeline run link when context is found', async () => {
      const context = new Context();
      context.setName('run-abc-123');
      getContextByExecutionSpy.mockResolvedValue(context);

      mockSuccessfulLoad();
      renderContent();

      await waitFor(() => {
        expect(screen.getByText('Pipeline Run')).toBeInTheDocument();
      });

      expect(screen.getByRole('link', { name: 'runs/details/run-abc-123' })).toHaveAttribute(
        'href',
        '/runs/details/run-abc-123/execution/1',
      );
    });

    it('renders an original execution link when cached execution ID exists', async () => {
      const execution = buildExecution();
      const cachedIdValue = new Value();
      cachedIdValue.setStringValue('original-exec-42');
      execution.getCustomPropertiesMap().set('cached_execution_id', cachedIdValue);

      mockSuccessfulLoad(execution);
      renderContent();

      await waitFor(() => {
        expect(screen.getByText('Original Execution')).toBeInTheDocument();
      });

      expect(screen.getByRole('link', { name: 'execution/original-exec-42' })).toHaveAttribute(
        'href',
        '/executions/original-exec-42',
      );
    });

    it('renders neither link when no context and no cached ID', async () => {
      getContextByExecutionSpy.mockResolvedValue(null);
      mockSuccessfulLoad();
      renderContent();

      await waitFor(() => {
        expect(screen.getByText('Reference')).toBeInTheDocument();
      });

      expect(screen.queryByText('Pipeline Run')).not.toBeInTheDocument();
      expect(screen.queryByText('Original Execution')).not.toBeInTheDocument();
      await TestUtils.flushPromises();
    });
  });
});

describe('ExecutionDetails (page wrapper)', () => {
  function buildPageProps(overrides?: { updateToolbar?: Mock }) {
    const updateToolbarSpy = overrides?.updateToolbar ?? vi.fn();
    const match = {
      isExact: true,
      path: RoutePage.EXECUTION_DETAILS,
      url: '/executions/1',
      params: { [RouteParams.ID]: '1' },
    } as any;
    return {
      props: TestUtils.generatePageProps(
        ExecutionDetails,
        { pathname: '/executions/1' } as any,
        match,
        vi.fn(),
        vi.fn(),
        vi.fn(),
        updateToolbarSpy,
        vi.fn(),
      ),
      updateToolbarSpy,
    };
  }

  it('sets initial toolbar state with execution ID in title', () => {
    vi.spyOn(Api.getInstance().metadataStoreService, 'getExecutionsByID').mockReturnValue(
      new Promise(() => {}),
    );
    vi.spyOn(Api.getInstance().metadataStoreService, 'getEventsByExecutionIDs').mockReturnValue(
      new Promise(() => {}),
    );
    vi.spyOn(Api.getInstance().metadataStoreService, 'getArtifactTypes').mockReturnValue(
      new Promise(() => {}),
    );

    const { props, updateToolbarSpy } = buildPageProps();

    render(
      <CommonTestWrapper>
        <ExecutionDetails {...props} />
      </CommonTestWrapper>,
    );

    expect(updateToolbarSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        pageTitle: 'Execution #1',
        breadcrumbs: [{ displayName: 'Executions', href: RoutePage.EXECUTIONS }],
      }),
    );
  });

  it('updates toolbar title via onTitleUpdate after successful load', () => {
    const { props, updateToolbarSpy } = buildPageProps();
    const wrapperElement = new ExecutionDetails(props).render() as any;
    const childElement = wrapperElement.props.children;
    childElement.props.onTitleUpdate('test-execution');

    expect(updateToolbarSpy).toHaveBeenCalledWith(
      expect.objectContaining({
        pageTitle: 'test-execution',
      }),
    );
  });
});
