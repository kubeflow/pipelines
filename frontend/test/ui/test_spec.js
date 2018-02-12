const puppeteer = require('puppeteer');
const fs = require('fs');
const PNG = require('pngjs').PNG;
const pixelmatch = require('pixelmatch');
const os = require('os');

async function waitForCustomElement(page, selector) {
  await page.waitForFunction(`!!(document.querySelector('${selector}') &&
                                 document.querySelector('${selector}').$)`);
}

let browser = null;
let page = null;
const PLATFORM_SUFFIX = os.type().toUpperCase();
const writeDiff = false;
TIMEOUT_INTERVAL = 10 * 1000;

const misMatchThreshold = 0.1;
const goldenPathPrefix = 'test/ui/golden';
const brokenPathPrefix = 'test/ui/broken';

if (!fs.existsSync(brokenPathPrefix)) {
  fs.mkdirSync(brokenPathPrefix);
}

function diffScreenshots(one, two, threshold) {
  const diffPromise = new Promise((resolve, reject) => {
    const doneReading = () => {
      if (++filesRead < 2) return;
      const diff = new PNG({ width: img1.width, height: img1.height });

      const pixelCount =
        pixelmatch(img1.data, img2.data, diff.data, img1.width, img1.height, { threshold });

      if (writeDiff) {
        diff.pack().pipe(fs.createWriteStream('diff.png'));
      }
      resolve(pixelCount);
    }

    const img1 = fs.createReadStream(one).pipe(new PNG()).on('parsed', doneReading);
    const img2 = fs.createReadStream(two).pipe(new PNG()).on('parsed', doneReading);
    let filesRead = 0;
  });
  return diffPromise;
}

async function takeScreenshotsAndDiff(testName) {
  const goldenPath = `${goldenPathPrefix}/${testName}_${PLATFORM_SUFFIX}.png`;
  const brokenPath = `${brokenPathPrefix}/${testName}_${PLATFORM_SUFFIX}.png`;
  if (!fs.existsSync(goldenPath)) {
    console.error(`Error: no golden image for test: ${testName}. Writing broken image anyway.`);
  }
  await page.screenshot({ path: brokenPath });
  const pixels = await diffScreenshots(goldenPath, brokenPath, misMatchThreshold);
  expect(pixels).toBe(0);
}

describe('UI tests', function () {

  beforeAll(async () => {
    browser = await puppeteer.launch();
    page = await browser.newPage();
    page.setViewport({ width: 1024, height: 768 });
    page.goto('http://localhost:3000');
    await waitForCustomElement(page, 'top-bar');
    await waitForCustomElement(page, 'app-shell');
    page.waitFor(1000);
  });

  it('loads templates page', async () => {
    await waitForCustomElement(page, 'top-bar');
    await waitForCustomElement(page, 'app-shell');
    await takeScreenshotsAndDiff('templates');
  });

  it('can interact with runs button with hover', async () => {
    await waitForCustomElement(page, 'top-bar');
    const runsBtn = await page.evaluateHandle(`
      document.querySelector('top-bar').$.runsBtn`);
      
    await runsBtn.hover();
    await page.waitFor(500); // wait for hover effect (0.3s)
    await takeScreenshotsAndDiff('runsBtn-hover');
  });

  it('loads runs page when its button is clicked', async () => {
    await waitForCustomElement(page, 'top-bar');
    const runsBtn = await page.evaluateHandle(`
      document.querySelector('top-bar').$.runsBtn`);
      
    await runsBtn.click();
    await page.waitForFunction(
      `document.querySelector('app-shell')
       .shadowRoot.querySelector('run-list')`);
    await page.waitFor(500);
    await takeScreenshotsAndDiff('runs');
  });

  it('can interact with runs page cards', async () => {
    await waitForCustomElement(page, 'top-bar');
    const card = await page.evaluateHandle(
      `document.querySelector('app-shell')
       .shadowRoot.querySelector('run-list')
       .shadowRoot.querySelector('.container .card')`);
    await card.hover();
    await page.waitFor(500);
    await takeScreenshotsAndDiff('run-card-hover');
  });

  it('loads run details of the clicked run card', async () => {
    await waitForCustomElement(page, 'top-bar');
    const card = await page.evaluateHandle(
      `document.querySelector('app-shell')
       .shadowRoot.querySelector('run-list')
       .shadowRoot.querySelector('.container .card')`);
    await card.click();
    await page.waitFor(500);
    await takeScreenshotsAndDiff('run-details');
  });

  it('navigates back to templates if the topbar logo is clicked', async () => {
    await waitForCustomElement(page, 'top-bar');
    const logo = await page.evaluateHandle(
      `document.querySelector('top-bar').$.logo`);
    await logo.click();
    await page.waitFor(500);
    await takeScreenshotsAndDiff('templates-from-topbar');
  });

  it('loads template details of the clicked template card', async () => {
    await waitForCustomElement(page, 'top-bar');
    const card = await page.evaluateHandle(
      `document.querySelector('app-shell')
       .shadowRoot.querySelector('template-list')
       .shadowRoot.querySelector('.container .card')`);
    await card.click();
    await page.waitFor(500);
    await takeScreenshotsAndDiff('template-details');
  });

  it('loads template configure page of the current template', async () => {
    await waitForCustomElement(page, 'top-bar');
    const configureBtn = await page.evaluateHandle(
      `document.querySelector('app-shell')
       .shadowRoot.querySelector('template-details')
       .shadowRoot.querySelector('.action-button')`);
    await configureBtn.click();
    await page.waitFor(500);
    await takeScreenshotsAndDiff('configure-template');
  });

  afterAll(async () => {
    await browser.close();
  });

});
