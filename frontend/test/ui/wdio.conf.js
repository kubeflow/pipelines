const path = require('path');
const VisualRegressionCompare = require('wdio-visual-regression-service/compare');

const headless = !!process.env.HEADLESS_UI_TESTS;
const singleSuite = process.env.SINGLE_SUITE;

function getScreenshotName(basePath) {
  return function (context) {
    const testName = context.test.title.replace(/ /g, '-');
    const suiteName = path.parse(context.test.file).name;
    const browserViewport = context.meta.viewport;
    const browserWidth = browserViewport.width;
    const browserHeight = browserViewport.height;

    return path.join(basePath, `${suiteName}/${testName}_${browserWidth}x${browserHeight}.png`);
  };
}

exports.config = {
  maxInstances: 10,
  baseUrl: 'http://localhost:3000',
  capabilities: [{
    maxInstances: 5,
    browserName: 'chrome',
    chromeOptions: {
      args: headless ? ['--headless', '--disable-gpu', '--window-size=1024,800'] : [],
    },
  }],
  coloredLogs: true,
  connectionRetryCount: 3,
  connectionRetryTimeout: 90000,
  deprecationWarnings: false,
  framework: 'jasmine',
  jasmineNodeOpts: {
    defaultTimeoutInterval: 10000,
  },
  logLevel: 'silent',
  plugins: {
    'wdio-webcomponents': {},
  },
  reporters: ['spec'],
  screenshotPath: './ui/errorShots/',
  services: ['selenium-standalone', 'visual-regression'],
  specs: [
    singleSuite || './ui/*.spec.js',
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
