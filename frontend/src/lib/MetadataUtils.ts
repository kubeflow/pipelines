import { ArtifactType, Event } from '../generated/src/apis/metadata/metadata_store_pb';
import {
  GetArtifactTypesRequest,
  GetEventsByArtifactIDsRequest,
  GetEventsByArtifactIDsResponse,
} from '../generated/src/apis/metadata/metadata_store_service_pb';
import { Apis } from './Apis';
import { formatDateString, serviceErrorToString } from './Utils';

export type EventTypes = Event.TypeMap[keyof Event.TypeMap];

export const getArtifactCreationTime = async (artifactId: number): Promise<string> => {
  const eventsRequest = new GetEventsByArtifactIDsRequest();
  if (!artifactId) {
    throw new Error('artifactId is empty');
  }

  eventsRequest.setArtifactIdsList([artifactId]);
  const { error, response } = await Apis.getMetadataServicePromiseClient().getEventsByArtifactIDs(
    eventsRequest,
  );
  if (error) {
    throw new Error(serviceErrorToString(error));
  }
  const data = (response as GetEventsByArtifactIDsResponse).getEventsList().map(event => ({
    time: event.getMillisecondsSinceEpoch(),
    type: event.getType() || Event.Type.UNKNOWN,
  }));
  // The last output event is the event that produced current artifact.
  const lastOutputEvent = data
    .reverse()
    .find(event => event.type === Event.Type.DECLARED_OUTPUT || event.type === Event.Type.OUTPUT);
  if (lastOutputEvent && lastOutputEvent.time) {
    return formatDateString(new Date(lastOutputEvent.time));
  } else {
    // No valid time found, just return empty
    return '';
  }
};

export const getArtifactTypeMap = async (): Promise<Map<number, ArtifactType>> => {
  const map = new Map<number, ArtifactType>();
  const { response, error } = await Apis.getMetadataServicePromiseClient().getArtifactTypes(
    new GetArtifactTypesRequest(),
  );
  if (error) {
    throw new Error(serviceErrorToString(error));
  }

  ((response && response.getArtifactTypesList()) || []).forEach(artifactType => {
    const id = artifactType.getId();
    if (id) {
      map.set(id, artifactType);
    }
  });
  return map;
};
