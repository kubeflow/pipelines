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
  reporters: ['dot'],
  screenshotPath: './ui/visual-regression/errorShots/',
  services: ['selenium-standalone', 'visual-regression'],
  specs: [
    singleSuite || './ui/visual-regression/*.spec.js',
  ],
  suites: {
    // These suites are separated to avoid race conditions between the e2e tests (which can modify
    // backend state) and the visual regression tests. Because of this, only the visual regression
    // tests will be run if `wdio ui/wdio.conf.js` is called. To run mock e2e tests, use:
    // `wdio ui/wdio.conf.js --suite mockEndToEnd`.
    mockEndToEnd: [
      './ui/end-to-end/*.spec.js',
    ],
    visualRegression: [
      './ui/visual-regression/*.spec.js',
    ],
  },
  sync: true,
  visualRegression: {
    compare: new VisualRegressionCompare.LocalCompare({
      referenceName: getScreenshotName(path.join(process.cwd(), 'ui/visual-regression/screenshots/reference')),
      screenshotName: getScreenshotName(path.join(process.cwd(), 'ui/visual-regression/screenshots/screen')),
      diffName: getScreenshotName(path.join(process.cwd(), 'ui/visual-regression/screenshots/diff')),
      misMatchTolerance: 0.01,
    }),
    viewports: [{ width: 1024, height: 768 }],
  },
  waitforTimeout: 10000,
}
