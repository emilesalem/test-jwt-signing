const { Worker, isMainThread, workerData, parentPort } = require('worker_threads');

if (isMainThread) {
  // duration of the test
  const elpased = 10;

  // rate of signing requests per milliseconds
  const milliRate = 500;

  const worker = new Worker(__filename);

  let totalSigned = 0;
  worker.on('message', () => {
    totalSigned++;
  });

  let totalSent = 0;

  setInterval(() => {
    totalSent += milliRate;
    for (let i = 0; i < milliRate; i++) {
      worker.postMessage('go');
    }
  }, 1);

  setTimeout(() => {
    console.log(`signed ${totalSigned} times in ${elpased} seconds, requested ${totalSent} signatures`);
    process.exit(0);
  }, elpased * 1000);

  return;
}

const jwt = require('jsonwebtoken');

parentPort.on('message', () => {
  jwt.sign({ iss: 'bar' }, 'shhhhh');
  parentPort.postMessage('done');
});
