import * as sinon from 'sinon';
import * as Apis from './apis';

const testTemplates = [{
  author: 'user1',
  description: 'An awesome unstructured text pipeline template.',
  id: 1,
  location: 'gcs://pipelines_bucket/structured_text1',
  name: 'Unstructured text',
  parameters: ['x', 'y'],
  sharedWith: 'team1',
  tags: [
    'text',
    'experiment'
  ],
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
