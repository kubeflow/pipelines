import { Event } from '../generated/src/apis/metadata/metadata_store_pb';
import { GetEventsByArtifactIDsRequest, GetEventsByArtifactIDsResponse } from '../generated/src/apis/metadata/metadata_store_service_pb';
import { Apis } from '../lib/Apis';
import { formatDateString, serviceErrorToString } from './Utils';

export const getArtifactCreationTime = async (artifactId: number): Promise<string> => {
  const eventsRequest = new GetEventsByArtifactIDsRequest();
  if (!artifactId) {
    throw new Error('artifactId is empty');
  }

  eventsRequest.setArtifactIdsList([artifactId]);
  const { error, response } = await Apis.getMetadataServicePromiseClient().getEventsByArtifactIDs(eventsRequest);
  if (error) {
    throw new Error(serviceErrorToString(error));
  }
  const data = (response as GetEventsByArtifactIDsResponse).getEventsList().map(event => ({
    time: event.getMillisecondsSinceEpoch(),
    type: event.getType() || Event.Type.UNKNOWN,
  }));
  // The last output event is the event that produced current artifact.
  const lastOutputEvent = data.reverse().find(event =>
    event.type === Event.Type.DECLARED_OUTPUT || event.type === Event.Type.OUTPUT
  );
  if (lastOutputEvent && lastOutputEvent.time) {
    return formatDateString(new Date(lastOutputEvent.time));
  } else {
    // No valid time found, just return empty
    return '';
  }
};
