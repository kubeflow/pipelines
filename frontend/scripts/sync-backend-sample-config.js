// This should be run in pipelines/frontend folder.

const PATH_BACKEND_CONFIG = '../backend/src/apiserver/config/sample_config.json';
const PATH_FRONTEND_CONFIG = 'src/config/sample_config_from_backend.json';
var myArgs = process.argv.slice(2);
if (myArgs.includes('--check')) {
  console.log('Checking frontend sample config...');
  try {
    const childProcess = require('child_process');
    const output = childProcess
      .execSync(`diff ${PATH_BACKEND_CONFIG} ${PATH_FRONTEND_CONFIG}`)
      .toString();
    console.log(output);
    console.log(`${PATH_FRONTEND_CONFIG} is up-to-date with ${PATH_BACKEND_CONFIG}.`);
  } catch (err) {
    if (err.stdout) {
      console.error(err.stdout.toString());
    }
    if (err.stderr) {
      console.error(err.stderr.toString());
    }
    console.error(err.message);
    console.error(
      `ERROR: ${PATH_FRONTEND_CONFIG} is not in sync with ${PATH_BACKEND_CONFIG}, please update ${PATH_FRONTEND_CONFIG} by running the following command:`,
    );
    console.error('npm run sync-backend-sample-config');
    process.exit(err.status);
  }
} else {
  const fs = require('fs');
  fs.copyFileSync(PATH_BACKEND_CONFIG, PATH_FRONTEND_CONFIG);
  console.log(`Synced ${PATH_BACKEND_CONFIG} to ${PATH_FRONTEND_CONFIG}`);
}
