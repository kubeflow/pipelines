import { Event } from '../generated/src/apis/metadata/metadata_store_pb';
import { GetEventsByArtifactIDsRequest, GetEventsByArtifactIDsResponse } from '../generated/src/apis/metadata/metadata_store_service_pb';
import { Apis } from '../lib/Apis';
import { formatDateString, serviceErrorToString } from './Utils';

export const getArtifactCreationTime = (artifactId: number): Promise<string> => {
  return new Promise((resolve, reject) => {
    const eventsRequest = new GetEventsByArtifactIDsRequest();
    if (!artifactId) {
      return reject(new Error('artifactId is empty'));
    }

    eventsRequest.setArtifactIdsList([artifactId]);
    Apis.getMetadataServiceClient().getEventsByArtifactIDs(eventsRequest, (err, res) => {
      if (err) {
        return reject(new Error(serviceErrorToString(err)));
      } else {
        const data = (res as GetEventsByArtifactIDsResponse).getEventsList().map(event => ({
          time: event.getMillisecondsSinceEpoch(),
          type: event.getType() || Event.Type.UNKNOWN,
        }));
        // The last output event is the event that produced current artifact.
        const lastOutputEvent = data.reverse().find(event =>
          event.type === Event.Type.DECLARED_OUTPUT || event.type === Event.Type.OUTPUT
        );
        if (lastOutputEvent && lastOutputEvent.time) {
          resolve(formatDateString(new Date(lastOutputEvent.time)));
        } else {
          // No valid time found, just return empty
          resolve('');
        }
      }
    });
  });
};
