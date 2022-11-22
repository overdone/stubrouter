# StubRouter
Reverse proxy for testing client side. It allows you to test some client app (UI ot something else)
without real backend server. Furthermore, you can redirect some queries to real backend and some queries to stubs. 
To all backend requests will be added Authorization header with valid JWT token(optional).

### config.toml
**server**
- host - Proxy listen host
- port - Proxy listen port

**session**
- duration - Session duration in **time.Duration** format(example: "1m", "3h30m")
- idle_timeout - Time in **time.Duration** format after that session will be expired
- cookie_name - Session cookie name
- token_secret - String key for signing JWT

**targets** - string to string map(path to host (real backend)).

    For example: "/app": "http://localhost:8888".
    Path /static and /stubapi not allowed because it uses by stubtouter ui

**stubs** - config stubs
- **storage**
  - **type** - Storage type **file** or **db** (Redis)
  - **path** - path to file or db
  - **cache** - memory stub cache settings
    - **enabled** - enabled or not
    - **expiration_interval** - Time in **time.Duration** format after that value will be expired
    - **cleanup_interval** - Time in **time.Duration** format after that value will be removed


## Usage
- Add target to config. Example "app1": "http://server:9090"
- Run proxy on some host and port. Example localhost:8080 
- All request to localhost:8080/app1 will be proxifyed to http://server:9090
- All request with stub config will be responded with stubs
- You can configure stubs in UI http://localhost:8080