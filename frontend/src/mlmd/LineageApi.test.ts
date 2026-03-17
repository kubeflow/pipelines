import { describe, expect, it, vi, beforeEach } from 'vitest';
import { getArtifactTypes, getExecutionTypes, getArtifactCreationTime } from './LineageApi';
import {
  ArtifactType,
  Event,
  ExecutionType,
  GetArtifactTypesResponse,
  GetEventsByArtifactIDsResponse,
  GetExecutionTypesResponse,
  MetadataStoreServicePromiseClient,
} from 'src/third_party/mlmd';

function buildMockMetadataStoreService(): MetadataStoreServicePromiseClient {
  return {
    getArtifactTypes: vi.fn(),
    getExecutionTypes: vi.fn(),
    getEventsByArtifactIDs: vi.fn(),
  } as unknown as MetadataStoreServicePromiseClient;
}

const SAMPLE_TIME_MS = 1_560_300_108_000;
const EARLIER_TIME_MS = 1_560_200_000_000;

function buildEvent(type: Event.Type, timeMs: number): Event {
  const event = new Event();
  event.setType(type);
  event.setMillisecondsSinceEpoch(timeMs);
  return event;
}

function mockEventsResponse(service: MetadataStoreServicePromiseClient, events: Event[]): void {
  const response = new GetEventsByArtifactIDsResponse();
  response.setEventsList(events);
  vi.mocked(service.getEventsByArtifactIDs).mockResolvedValue(response);
}

describe('LineageApi', () => {
  let metadataStoreService: MetadataStoreServicePromiseClient;

  beforeEach(() => {
    metadataStoreService = buildMockMetadataStoreService();
  });

  describe('getArtifactTypes', () => {
    it('returns populated Map when service responds with types', async () => {
      const type1 = new ArtifactType();
      type1.setId(1);
      type1.setName('Model');
      const type2 = new ArtifactType();
      type2.setId(2);
      type2.setName('Dataset');

      const response = new GetArtifactTypesResponse();
      response.setArtifactTypesList([type1, type2]);
      vi.mocked(metadataStoreService.getArtifactTypes).mockResolvedValue(response);

      const result = await getArtifactTypes(metadataStoreService);
      expect(result.size).toBe(2);
      expect(result.get(1)!.getName()).toBe('Model');
      expect(result.get(2)!.getName()).toBe('Dataset');
    });

    it('returns empty Map and calls errorCallback when response is null', async () => {
      vi.mocked(metadataStoreService.getArtifactTypes).mockResolvedValue(
        null as unknown as GetArtifactTypesResponse,
      );
      const errorCallback = vi.fn();

      const result = await getArtifactTypes(metadataStoreService, errorCallback);
      expect(result.size).toBe(0);
      expect(errorCallback).toHaveBeenCalledWith(
        'Unable to retrieve Artifact Types, some features may not work.',
      );
    });

    it('returns empty Map without throwing when no errorCallback provided', async () => {
      vi.mocked(metadataStoreService.getArtifactTypes).mockResolvedValue(
        null as unknown as GetArtifactTypesResponse,
      );

      const result = await getArtifactTypes(metadataStoreService);
      expect(result.size).toBe(0);
    });
  });

  describe('getExecutionTypes', () => {
    it('returns populated Map when service responds with types', async () => {
      const type1 = new ExecutionType();
      type1.setId(10);
      type1.setName('Train');
      const type2 = new ExecutionType();
      type2.setId(20);
      type2.setName('Evaluate');

      const response = new GetExecutionTypesResponse();
      response.setExecutionTypesList([type1, type2]);
      vi.mocked(metadataStoreService.getExecutionTypes).mockResolvedValue(response);

      const result = await getExecutionTypes(metadataStoreService);
      expect(result.size).toBe(2);
      expect(result.get(10)!.getName()).toBe('Train');
      expect(result.get(20)!.getName()).toBe('Evaluate');
    });

    it('returns empty Map and calls errorCallback when response is null', async () => {
      vi.mocked(metadataStoreService.getExecutionTypes).mockResolvedValue(
        null as unknown as GetExecutionTypesResponse,
      );
      const errorCallback = vi.fn();

      const result = await getExecutionTypes(metadataStoreService, errorCallback);
      expect(result.size).toBe(0);
      expect(errorCallback).toHaveBeenCalledWith(
        'Unable to retrieve Execution Types, some features may not work.',
      );
    });

    it('returns empty Map without throwing when no errorCallback provided', async () => {
      vi.mocked(metadataStoreService.getExecutionTypes).mockResolvedValue(
        null as unknown as GetExecutionTypesResponse,
      );

      const result = await getExecutionTypes(metadataStoreService);
      expect(result.size).toBe(0);
    });
  });

  describe('getArtifactCreationTime', () => {
    it('throws when artifactId is 0', async () => {
      await expect(getArtifactCreationTime(0, metadataStoreService)).rejects.toThrow(
        'artifactId is empty',
      );
    });

    it('returns empty string and calls errorCallback when event response is null', async () => {
      vi.mocked(metadataStoreService.getEventsByArtifactIDs).mockResolvedValue(
        null as unknown as GetEventsByArtifactIDsResponse,
      );
      const errorCallback = vi.fn();

      const result = await getArtifactCreationTime(42, metadataStoreService, errorCallback);
      expect(result).toBe('');
      expect(errorCallback).toHaveBeenCalledWith('Unable to retrieve Events for artifactId: 42');
    });

    it('returns empty string without throwing when null response and no errorCallback', async () => {
      vi.mocked(metadataStoreService.getEventsByArtifactIDs).mockResolvedValue(
        null as unknown as GetEventsByArtifactIDsResponse,
      );

      const result = await getArtifactCreationTime(42, metadataStoreService);
      expect(result).toBe('');
    });

    it('returns formatted date of the last OUTPUT event', async () => {
      mockEventsResponse(metadataStoreService, [
        buildEvent(Event.Type.INPUT, EARLIER_TIME_MS),
        buildEvent(Event.Type.OUTPUT, SAMPLE_TIME_MS),
      ]);

      const result = await getArtifactCreationTime(1, metadataStoreService);
      expect(result).toBe(new Date(SAMPLE_TIME_MS).toLocaleString());
    });

    it('returns formatted date of DECLARED_OUTPUT when no OUTPUT event exists', async () => {
      mockEventsResponse(metadataStoreService, [
        buildEvent(Event.Type.INPUT, EARLIER_TIME_MS),
        buildEvent(Event.Type.DECLARED_OUTPUT, SAMPLE_TIME_MS),
      ]);

      const result = await getArtifactCreationTime(1, metadataStoreService);
      expect(result).toBe(new Date(SAMPLE_TIME_MS).toLocaleString());
    });

    it('returns empty string when no output events exist at all', async () => {
      mockEventsResponse(metadataStoreService, [buildEvent(Event.Type.INPUT, EARLIER_TIME_MS)]);

      const result = await getArtifactCreationTime(1, metadataStoreService);
      expect(result).toBe('');
    });
  });
});
