/**
 * Errorium Frontend API Client
 * Zero external dependencies.
 * Provides authenticated HTTP requests, JWT token management, automatic 401 refresh,
 * and SSE ReadableStream handling.
 */

(function () {
    'use strict';

    const BASE_URL = '/api/v1';
    const TOKENS_KEY = 'errorium_tokens';

    /**
     * Retrieve stored tokens from localStorage.
     * @returns {{ access_token: string, refresh_token: string } | null}
     */
    function getTokens() {
        try {
            const raw = localStorage.getItem(TOKENS_KEY);
            return raw ? JSON.parse(raw) : null;
        } catch (e) {
            return null;
        }
    }

    /**
     * Store tokens in localStorage.
     * @param {{ access_token: string, refresh_token: string }} tokens
     */
    function setTokens(tokens) {
        try {
            if (tokens && tokens.access_token) {
                localStorage.setItem(TOKENS_KEY, JSON.stringify({
                    access_token: tokens.access_token,
                    refresh_token: tokens.refresh_token || ''
                }));
            } else {
                clearTokens();
            }
        } catch (e) {}
    }

    /**
     * Remove tokens from localStorage.
     */
    function clearTokens() {
        try {
            localStorage.removeItem(TOKENS_KEY);
        } catch (e) {}
    }

    /**
     * Format full API endpoint URL.
     * @param {string} path
     * @returns {string}
     */
    function buildUrl(path) {
        if (path.startsWith('http://') || path.startsWith('https://')) {
            return path;
        }
        if (path.startsWith(BASE_URL)) {
            return path;
        }
        const clean = path.startsWith('/') ? path : '/' + path;
        return BASE_URL + clean;
    }

    /**
     * Execute an HTTP request with JSON parsing, Bearer auth and automatic 401 token refresh.
     * @param {string} method HTTP method ('GET', 'POST', 'PATCH', 'DELETE', etc.)
     * @param {string} path API path (e.g. '/auth/login', '/users/me')
     * @param {Object} [options]
     * @param {Object|string|FormData} [options.body]
     * @param {boolean} [options.auth=true] Whether to attach Bearer token
     * @param {boolean} [options.multipart=false] Whether sending multipart/form-data
     * @param {Object} [options.headers] Additional headers
     * @param {AbortSignal} [options.signal] AbortController signal
     * @param {boolean} [options._isRetry=false] Internal retry guard
     * @returns {Promise<any>}
     */
    async function request(method, path, options) {
        const opts = options || {};
        const methodUpper = (method || 'GET').toUpperCase();
        const url = buildUrl(path);
        const headers = Object.assign({}, opts.headers || {});

        const useAuth = opts.auth !== false;
        const tokens = getTokens();

        if (useAuth && tokens && tokens.access_token) {
            headers['Authorization'] = 'Bearer ' + tokens.access_token;
        }

        let body = opts.body;
        if (!opts.multipart && body !== undefined && body !== null && typeof body !== 'string') {
            headers['Content-Type'] = 'application/json';
            body = JSON.stringify(body);
        }

        const fetchInit = {
            method: methodUpper,
            headers: headers
        };

        if (methodUpper !== 'GET' && methodUpper !== 'HEAD' && body !== undefined) {
            fetchInit.body = body;
        }

        if (opts.signal) {
            fetchInit.signal = opts.signal;
        }

        let resp;
        try {
            resp = await fetch(url, fetchInit);
        } catch (netErr) {
            return Promise.reject({
                status: 0,
                error: netErr.message || 'Network error'
            });
        }

        // Handle 401 Unauthorized with token refresh (exactly one attempt)
        if (resp.status === 401 && useAuth && !opts._isRetry) {
            if (tokens && tokens.refresh_token) {
                let refreshOk = false;
                let newTokens = null;

                try {
                    const refreshResp = await fetch(buildUrl('/auth/refresh'), {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({ refresh_token: tokens.refresh_token })
                    });
                    if (refreshResp.ok) {
                        newTokens = await refreshResp.json();
                        if (newTokens && newTokens.access_token) {
                            setTokens(newTokens);
                            refreshOk = true;
                        }
                    }
                } catch (e) {
                    refreshOk = false;
                }

                if (refreshOk) {
                    const retryOpts = Object.assign({}, opts, { _isRetry: true });
                    return request(method, path, retryOpts);
                }
            }

            // Refresh failed or no refresh_token present
            clearTokens();
            if (typeof window !== 'undefined' && typeof window.dispatchEvent === 'function') {
                window.dispatchEvent(new CustomEvent('errorium:logout'));
            }
        }

        // 204 No Content
        if (resp.status === 204) {
            return null;
        }

        // Try reading body (JSON or text)
        const contentType = resp.headers.get('content-type') || '';
        let data = null;
        if (contentType.includes('application/json')) {
            try {
                data = await resp.json();
            } catch (jsonErr) {
                data = null;
            }
        } else {
            try {
                data = await resp.text();
            } catch (textErr) {
                data = null;
            }
        }

        if (!resp.ok) {
            let errorText = resp.statusText;
            if (data && typeof data === 'object') {
                errorText = data.error || data.message || errorText;
            } else if (typeof data === 'string' && data.length > 0) {
                errorText = data;
            }
            return Promise.reject({
                status: resp.status,
                error: errorText,
                data: data
            });
        }

        return data;
    }

    /**
     * Helper for Server-Sent Events (SSE) streaming via fetch and ReadableStream.
     * Works with POST /api/v1/search/rag and POST /api/v1/chats/{id}/messages.
     * @param {string} path API path
     * @param {Object} body Request payload
     * @param {Function} onEvent Callback function(eventName, data)
     * @param {Object} [options]
     * @param {boolean} [options.auth=true]
     * @param {string} [options.method='POST']
     * @param {AbortSignal} [options.signal]
     * @returns {Promise<void>}
     */
    async function stream(path, body, onEvent, options) {
        const opts = options || {};
        const method = (opts.method || 'POST').toUpperCase();
        const url = buildUrl(path);
        const headers = Object.assign({
            'Accept': 'text/event-stream',
            'Content-Type': 'application/json'
        }, opts.headers || {});

        const useAuth = opts.auth !== false;
        const tokens = getTokens();
        if (useAuth && tokens && tokens.access_token) {
            headers['Authorization'] = 'Bearer ' + tokens.access_token;
        }

        const fetchInit = {
            method: method,
            headers: headers
        };

        if (body !== undefined && body !== null) {
            fetchInit.body = typeof body === 'string' ? body : JSON.stringify(body);
        }

        if (opts.signal) {
            fetchInit.signal = opts.signal;
        }

        const resp = await fetch(url, fetchInit);

        if (!resp.ok) {
            let errMsg = resp.statusText;
            try {
                const errData = await resp.json();
                errMsg = errData.error || errData.message || errMsg;
            } catch (_) {}
            throw {
                status: resp.status,
                error: errMsg
            };
        }

        if (!resp.body || !resp.body.getReader) {
            throw new Error('ReadableStream is not supported by the environment');
        }

        const reader = resp.body.getReader();
        const decoder = new TextDecoder('utf-8');
        let buffer = '';
        let currentEvent = 'message';
        let currentData = '';

        try {
            while (true) {
                const { value, done } = await reader.read();
                if (done) break;

                buffer += decoder.decode(value, { stream: true });
                const lines = buffer.split('\n');
                // Keep the trailing incomplete line in buffer
                buffer = lines.pop() || '';

                for (let i = 0; i < lines.length; i++) {
                    const rawLine = lines[i];
                    const line = rawLine.trimEnd();

                    if (line === '') {
                        // Empty line signals event dispatch in SSE
                        if (currentData !== '') {
                            let parsed = currentData;
                            try {
                                parsed = JSON.parse(currentData);
                            } catch (_) {}
                            if (typeof onEvent === 'function') {
                                onEvent(currentEvent, parsed);
                            }
                            currentEvent = 'message';
                            currentData = '';
                        }
                    } else if (line.startsWith('event:')) {
                        currentEvent = line.slice(6).trim();
                    } else if (line.startsWith('data:')) {
                        const dataSlice = line.slice(5).trim();
                        currentData = currentData ? (currentData + '\n' + dataSlice) : dataSlice;
                    }
                }
            }

            // Flush remaining buffer if ending with non-empty event
            if (currentData !== '') {
                let parsed = currentData;
                try {
                    parsed = JSON.parse(currentData);
                } catch (_) {}
                if (typeof onEvent === 'function') {
                    onEvent(currentEvent, parsed);
                }
            }
        } finally {
            reader.releaseLock();
        }
    }

    /**
     * Initiate OAuth2/SSO login redirect for the specified provider.
     * @param {string} [provider='yandex']
     */
    function ssoLogin(provider) {
        const prov = (provider || 'yandex').toLowerCase();
        window.location.href = buildUrl(`/auth/sso/login/${prov}`);
    }

    // Export API object to window.ErroriumAPI
    window.ErroriumAPI = {
        BASE_URL: BASE_URL,
        TOKENS_KEY: TOKENS_KEY,
        getTokens: getTokens,
        setTokens: setTokens,
        clearTokens: clearTokens,
        request: request,
        stream: stream,
        ssoLogin: ssoLogin,
        adminGetStats: function () { return this.request('GET', '/admin/stats'); },
        adminListUsers: function (q) { return this.request('GET', '/admin/users' + (q ? ('?search=' + encodeURIComponent(q)) : '')); },
        adminPatchUser: function (id, body) { return this.request('PATCH', '/admin/users/' + encodeURIComponent(id), { body: body }); },
        adminGetRagConfig: function () { return this.request('GET', '/admin/rag/config'); },
        adminPutRagConfig: function (body) { return this.request('PUT', '/admin/rag/config', { body: body }); },
        adminReloadIndex: function () { return this.request('POST', '/admin/index/reload'); }
    };
})();
