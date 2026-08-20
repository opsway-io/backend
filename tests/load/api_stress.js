import http from 'k6/http';
import { check, sleep } from 'k6';
import { randomString } from 'https://jslib.k6.io/k6-utils/1.2.0/index.js';

export const options = {
    stages: [
        { duration: '5s', target: 5 }, // Ramp up to 5 virtual users
        { duration: '15s', target: 5 }, // Stay at 5 users for 15s
        { duration: '5s', target: 0 },  // Ramp down
    ],
};

const BASE_URL = 'http://localhost:8001/v1';

export function setup() {
    // Authenticate and get token
    const loginRes = http.post(`${BASE_URL}/authentication/login`, JSON.stringify({
        email: 'admin@opsway.eu',
        password: 'pass'
    }), {
        headers: { 'Content-Type': 'application/json' }
    });

    // Check if login is successful. 
    // We expect a cookie to be set, or a token to be returned.
    // Opsway uses cookies (e.g., access_token). k6 automatically manages cookies.
    
    let teamId = 1; // Assuming team ID 1 for opsway team from seed
    
    // We can also extract data if it's in JSON
    if (loginRes.status === 200) {
        try {
            const body = loginRes.json();
            if (body.user && body.user.teams && body.user.teams.length > 0) {
                teamId = body.user.teams[0].id;
            }
        } catch (e) {
            console.error("Failed to parse login response");
        }
    }

    return { teamId };
}

export default function (data) {
    const teamId = data.teamId;

    // 1. Create a monitor
    const monitorPayload = JSON.stringify({
        name: `Load Test Monitor ${randomString(5)}`,
        settings: {
            method: 'GET',
            url: 'https://example.com',
            frequency: 1, // 1 minute
            tls: {
                enabled: true
            }
        }
    });

    const createRes = http.post(`${BASE_URL}/teams/${teamId}/monitors`, monitorPayload, {
        headers: { 'Content-Type': 'application/json' },
    });

    check(createRes, {
        'monitor created successfully': (r) => r.status === 201 || r.status === 200,
    });

    // 2. Fetch all monitors
    const getRes = http.get(`${BASE_URL}/teams/${teamId}/monitors`);
    check(getRes, {
        'fetched monitors': (r) => r.status === 200,
    });

    sleep(1);
}
