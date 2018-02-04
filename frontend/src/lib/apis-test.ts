import * as Apis from './apis';
import * as sinon from 'sinon';

const testTemplates = [{
  "id": 1,
  "name": "Unstructured text",
  "author": "user1",
  "tags": [
    "text",
    "experiment"
  ],
  "description": "An awesome unstructured text pipeline template.",
  "location": "gcs://pipelines_bucket/structured_text1",
  "parameters": ["x", "y"],
  "sharedWith": "team1"
}];

describe('My API', () => {
  let server: sinon.SinonFakeServer;

  beforeEach(() => {
    server = sinon.fakeServer.create();
  });

  afterEach(() => {
    server.restore();
  });

  it('should request _templates', async (done) => {
    server.respondWith('GET', '_templates',
      [200, { 'Content-Type': 'application/json' },
        JSON.stringify(testTemplates)]);

    try {
      const response = await Apis.getTemplates();
      assert.equal(response, JSON.stringify(testTemplates));
      done();
    } catch (err) {
      done(err);
    }
    server.respond();
  });
});