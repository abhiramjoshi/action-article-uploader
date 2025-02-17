const childProcess = require('child_process');
const os = require('os');
const process = require('process');

const VERSION = process.env.INPUT_VERSION
if (VERSION == '') {
  console.error("There was an error getting the action version")
  process.exit(1)
}

function chooseBinary() {
    const platform = os.platform()
    const arch = os.arch()

    if (platform === 'linux' && arch === 'x64') {
        return `action-linux-amd64-${VERSION}`
    }
    if (platform === 'linux' && arch === 'arm64') {
        return `action-linux-arm64-${VERSION}`
    }

    console.error(`Unsupported platform (${platform}) and architecture (${arch})`)
    process.exit(1)
}

function main() {
    const binary = chooseBinary()
    const mainScript = `${__dirname}/bins/${binary}/${binary}`
    console.log(`Using file ${binary}`);
    const spawnSyncReturns = childProcess.spawnSync(mainScript, { stdio: 'inherit' });
    const status = spawnSyncReturns.status
    if (status != 0) {
      console.log(`Failed exit status of ${status}`);  
      console.log(spawnSyncReturns.error)
      process.exit(status);
    }
    process.exit(0);
}

if (require.main === module) {
    main();
}
