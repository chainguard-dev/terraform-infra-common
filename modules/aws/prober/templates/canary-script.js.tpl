const synthetics = require('Synthetics');
const log = require('SyntheticsLogger');
const https = require('https');

const apiCanaryBlueprint = async function () {
    const hostname = '${service_url}'.replace(/^https?:\/\//, '').replace(/\/$/, '');

    log.info(`Checking prober endpoint: https://$${hostname}`);

    // Use executeStep with manual HTTPS request for better control
    await synthetics.executeStep('Verify prober endpoint', async function() {
        return new Promise((resolve, reject) => {
            const options = {
                hostname: hostname,
                port: 443,
                path: '/',
                method: 'GET',
                headers: {
                    'Authorization': '${authorization}'
                }
            };

            const req = https.request(options, (res) => {
                log.info(`Status Code: $${res.statusCode}`);

                // Collect response body
                let body = '';
                res.on('data', (chunk) => {
                    body += chunk;
                });

                res.on('end', () => {
                    if (res.statusCode === 200) {
                        log.info('Successfully verified prober endpoint with status 200');
                        log.info(`Response body: $${body.substring(0, 200)}`);
                        resolve();
                    } else {
                        const errorMsg = `Failed: Expected status code 200, got $${res.statusCode}`;
                        log.error(errorMsg);
                        log.error(`Response body: $${body.substring(0, 200)}`);
                        reject(new Error(errorMsg));
                    }
                });
            });

            req.on('error', (error) => {
                log.error(`Request failed: $${error.message}`);
                reject(error);
            });

            req.setTimeout(10000, () => {
                req.destroy();
                reject(new Error('Request timeout after 10 seconds'));
            });

            req.end();
        });
    });
};

exports.handler = async () => {
    return await apiCanaryBlueprint();
};
