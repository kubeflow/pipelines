const { Storage } = require('@google-cloud/storage');

async function main() {
  try {
    const storage = new Storage();

    const [repoTokenRaw] = await storage
      .bucket('ml-pipeline-test-keys')
      .file('coveralls_repo_token')
      .download();
    const repoToken = repoTokenRaw.toString().trim();
    if (!repoToken) {
      throw new Error('Repo token is empty!');
    }
    console.log(repoToken);
  } catch (err) {
    err.stack && console.error(err.stack);
    console.error('Error occured when fetching coveralls.io repo token: ', err.message);
    process.exit(1);
  }
}

main();
