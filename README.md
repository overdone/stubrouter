# StubRouter
Reverse proxy for testing client side. It allows you to test some client app (UI ot something else)
without real backend server. Furthermore, you can redirect some queries to real backend and some queries to stubs. 
To all backend requests will be added Authorization header with valid JWT token(optional).

## Application options
```
  -t, --target=                          Target pair target_path:target_host

server:
  -h, --server.host=                     Listen host address (default: 0.0.0.0)
  -p, --server.port=                     Listen host port (default: 3333)

session:
      --session.duration=                Session duration in time.Duration format (default: 24h)
      --session.idle-timeout=            Session idle in time.Duration format (default: 0h)
      --session.cookie-name=             Session cookie name (default: sessid)

auth:
      --auth.enabled                     Enable auth
      --auth.user-field=                 Auth user field in JWT token

stubs:
      --stubs.type=                      Stub storage type: file, redis (default: file)
      --stubs.path=                      Stub storage path: FS path, redis connect string (default: .)

cache:
      --stubs.cache.enabled              Cache stub in memory
      --stubs.cache.expiration-interval= Stub lifetime in cache (default: 30m)
      --stubs.cache.cleanup-interval=    Remove stub from cache after (default: 60m)

Help Options:
  -h, --help                             Show this help message
```


## Usage scenario
- Run proxy on some host and port and targets

``` stubrouter  -h localhost -p 8080 -t "/app1:http://server:9090"```
- All request to localhost:8080/app1 will be proxifyed to http://server:9090
- All request with stub config will be responded with stubs
- You can configure stubs in UI http://localhost:8080