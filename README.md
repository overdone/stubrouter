# StubRouter
Reverse proxy for testing client side. It allows you to test some client app (UI ot something else)
without real backend server. Furthermore, you can redirect some queries to real backend and some queries to stubs. 
To all backend requests will be added Authorization header with valid JWT token(optional).

## All program options
```
  -h, --host=                     Listen host address (default: 0.0.0.0)
  -p, --port=                     Listen host port (default: 3333)
      --sess-duration=            Session duration in time.Duration format
                                  (default: 24h)
      --sess-idle=                Session idle in time.Duration format
                                  (default: 0h)
      --sess-cookie-name=         Session cookie name (default: sessid)
      --auth-enabled              Enable auth
      --auth-user-field=          Auth user field in JWT token
  -t, --target=                   Target pair target_path:target_host
      --stub-type=                Stub storage type: file, redis (default: file)
      --stub-path=                Stub storage path: FS path, redis connect
                                  string (default: .)
      --stub-cache-enabled        Cache stub in memory
      --stub-expiration-interval= Stub lifetime in cache (default: 30m)
      --stub-cleanup-interval=    Remove stub from cache after (default: 60m)
```


## Usage scenario
- Run proxy on some host and port and targets

``` stubrouter  -h localhost -p 8080 -t "/app1:http://server:9090"```
- All request to localhost:8080/app1 will be proxifyed to http://server:9090
- All request with stub config will be responded with stubs
- You can configure stubs in UI http://localhost:8080