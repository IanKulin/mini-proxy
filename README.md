# mini-proxy

**mini-proxy** is a web proxy that only serves a single page or endpoint from somewhere else in the web.

Q. What would I use this for?
A. I have no idea what you might use it for. What I use it for is re-exposing a health check endpoint from an app I have behind basic auth in Nginx Proxy Manager. Instead of writing a complicated exception in the NPM GUI to allow my external monitoring to check it, I proxy just the health check endpoint with mini-proxy inside the same Docker network, then expose this with no authentication.

## Configuration

mini-proxy uses environment variables for configuration:

- `TARGET_HEALTH_URL`: The URL to proxy requests to (default: http://your-app:8080/health)
- `PORT`: The port to listen on (default: 8080)
- `RATE_LIMIT_MS`: Minimum milliseconds between requests (default: 1000ms)
- `RATE_LIMIT_WHITELIST_IP`: IP address to bypass rate limiting (optional)
- `ONLY_TRUST_PROXY_IP`: IP Address of proxy

## Rate Limiting

mini-proxy includes simple rate limiting to prevent abuse. By default, it allows one request per 1000ms. You can configure this with:

- `RATE_LIMIT_MS`: Set to a different value (e.g., 500 for 500ms between requests)
- `RATE_LIMIT_WHITELIST_IP`: Set to a specific IP address to bypass rate limiting for that IP
- `ONLY_TRUST_PROXY_IP`: If the whitelist IP (above) is in the any of the headers, rate limiting is skipped, defining a trusted proxy means it will only be skipped if this proxy is detected AND the whitelisted IP is in the headers

## Example Usage

```yaml
services:
  mini-proxy:
    image: mini-proxy
    ports:
      - "8080:8080"
    environment:
      - TARGET_HEALTH_URL=https://api.example.com/health
      - RATE_LIMIT_MS=500
      - RATE_LIMIT_WHITELIST_IP=192.168.1.100
      - ONLY_ONLY_TRUST_PROXY_IP=127.0.0.1
