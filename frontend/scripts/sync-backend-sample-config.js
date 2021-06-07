// This should be run in pipelines/frontend folder.

const PATH_BACKEND_CONFIG = '../../backend/src/apiserver/config/sample_config.json';
const PATH_FRONTEND_CONFIG = 'src/config/sample_config_from_backend.json';

const fs = require('fs');
const backendConfig = require(PATH_BACKEND_CONFIG);
const frontendConfig = backendConfig.map(sample => sample.name);
const content = JSON.stringify(frontendConfig, null, 2) + '\n';
fs.writeFileSync(PATH_FRONTEND_CONFIG, content);
console.log(`Synced ${PATH_BACKEND_CONFIG} to ${PATH_FRONTEND_CONFIG}`);
