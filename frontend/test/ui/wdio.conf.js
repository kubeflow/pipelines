const path = require('path');
const VisualRegressionCompare = require('wdio-visual-regression-service/compare');

function getScreenshotName(basePath) {
  return function (context) {
    const type = context.type;
    const testName = context.test.title;
    const browserVersion = parseInt(context.browser.version, 10);
    const browserName = context.browser.name;
    const browserViewport = context.meta.viewport;
    const browserWidth = browserViewport.width;
    const browserHeight = browserViewport.height;

    return path.join(basePath, `${testName}_${type}_${browserName}_v${browserVersion}_${browserWidth}x${browserHeight}.png`);
  };
}

exports.config = {
  maxInstances: 10,
  baseUrl: 'http://localhost:3000',
  capabilities: [{
    maxInstances: 5,
    browserName: 'chrome'
  }],
  coloredLogs: true,
  connectionRetryCount: 3,
  connectionRetryTimeout: 90000,
  deprecationWarnings: true,
  framework: 'jasmine',
  jasmineNodeOpts: {
    defaultTimeoutInterval: 10000,
  },
  logLevel: 'silent',
  plugins: {
    'wdio-webcomponents': {},
  },
  reporters: ['dot'],
  screenshotPath: './ui/errorShots/',
  services: ['selenium-standalone', 'visual-regression'],
  specs: [
    './ui/*.spec.js'
  ],
  sync: true,
  visualRegression: {
    compare: new VisualRegressionCompare.LocalCompare({
      referenceName: getScreenshotName(path.join(process.cwd(), 'ui/screenshots/reference')),
      screenshotName: getScreenshotName(path.join(process.cwd(), 'ui/screenshots/screen')),
      diffName: getScreenshotName(path.join(process.cwd(), 'ui/screenshots/diff')),
      misMatchTolerance: 0.01,
    }),
    viewports: [{ width: 1024, height: 768 }],
  },
  waitforTimeout: 10000,
}
