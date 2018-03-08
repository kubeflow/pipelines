import * as sinon from 'sinon';
import * as Apis from './apis';

const testPackages = [{
  author: 'user1',
  description: 'An awesome unstructured text pipeline package.',
  id: 1,
  location: 'gcs://pipelines_bucket/structured_text1',
  name: 'Unstructured text',
  parameters: ['x', 'y'],
}];

describe('My API', () => {
  let server: sinon.SinonFakeServer;

  beforeEach(() => {
    server = sinon.fakeServer.create();
  });

  afterEach(() => {
    server.restore();
  });

  it('should request _api/packages', async (done) => {
    server.respondWith('GET', '_api/packages',
      [200, { 'Content-Type': 'application/json' },
        JSON.stringify(testPackages)]);

    try {
      const response = await Apis.getPackages();
      assert.equal(response, JSON.stringify(testPackages));
      done();
    } catch (err) {
      done(err);
    }
    server.respond();
  });
});
