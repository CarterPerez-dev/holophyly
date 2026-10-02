<!-- ©AngelaMos | 2026 -->
<!-- README.md -->

```json
██╗  ██╗ ██████╗ ██╗      ██████╗ ██████╗ ██╗  ██╗██╗   ██╗██╗     ██╗   ██╗
██║  ██║██╔═══██╗██║     ██╔═══██╗██╔══██╗██║  ██║╚██╗ ██╔╝██║     ╚██╗ ██╔╝
███████║██║   ██║██║     ██║   ██║██████╔╝███████║ ╚████╔╝ ██║      ╚████╔╝
██╔══██║██║   ██║██║     ██║   ██║██╔═══╝ ██╔══██║  ╚██╔╝  ██║       ╚██╔╝
██║  ██║╚██████╔╝███████╗╚██████╔╝██║     ██║  ██║   ██║   ███████╗   ██║
╚═╝  ╚═╝ ╚═════╝ ╚══════╝ ╚═════╝ ╚═╝     ╚═╝  ╚═╝   ╚═╝   ╚══════╝   ╚═╝
```

[![Repo](https://img.shields.io/badge/github-holophyly-red?style=flat&logo=github)](https://github.com/CarterPerez-dev/holophyly)
[![Go](https://img.shields.io/badge/Go-1.24-00ADD8?style=flat&logo=go&logoColor=white)](https://go.dev)
[![Go Reference](https://img.shields.io/badge/pkg.go.dev-reference-007d9c?style=flat&logo=go&logoColor=white)](https://pkg.go.dev/github.com/carterperez-dev/holophyly)
[![API first](https://img.shields.io/badge/API-first-4B7BEC?style=flat)](#the-api)
[![Socket](https://img.shields.io/badge/docker.sock-proxied-6D4AFF?style=flat)](#the-socket-is-the-whole-security-story)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)

> Holophyly is a Docker Compose project manager that walks your development directories, finds every compose file you have ever written, groups them into projects, and gives you one HTTP API to start, stop, restart and watch all of them. It streams live container stats and logs over a WebSocket, tells you which compose stack is holding the port you are trying to bind, and refuses outright to stop anything you have marked protected. It is API-first on purpose: a single static Go binary with an embedded status page, built to be driven by whatever front end you already have.

## Why this exists

If you run more than about five Compose projects on one machine, the hard part stops being Docker and starts being bookkeeping. Which of these forty directories has a compose file. Which stack is up right now. What is on port 47312. Is this one the dev stack or the production stack with the live Cloudflare tunnel, because those two look identical in `docker ps` and only one of them has paying users behind it.

Docker Desktop answers some of that and is not available headless. Portainer answers more of it and wants to own your whole deployment story. Holophyly does one thing: it treats a directory tree full of compose files as the unit of work, and exposes it as an API clean enough to build anything on top of.

The safety feature came out of that last question. A project whose name or path matches a protection pattern, or that the scanner recognizes as carrying a Cloudflare tunnel or an edge proxy, cannot be stopped through the API at all. Not with a confirmation dialog. The endpoint refuses.

## How it works

```
  ~/dev, ~/projects
        |
        v
   scanner  ----  walks the tree, skips node_modules/.git/vendor/.venv
        |          finds compose.yml, compose.yaml, docker-compose.y{a,}ml
        v
   project manager  ----  one project per compose file, with:
        |                   environment class (dev / staging / prod)
        |                   tunnel + reverse-proxy detection
        |                   protection verdict
        v
   docker client  ----  containers, stats, logs, system, prune
        |
        +--> chi router  --> JSON over /api
        +--> hub         --> live stats over /ws/stats
        +--> embedded UI --> /
```

The scanner classifies each project by name and path. Tokens like `dev`, `local`, `staging` and `debug` read one way; `prod`, `production`, `live` and `release` read another. Separately it looks for the things that mean "do not touch": `cloudflared`, `cf-tunnel`, `argo`, `traefik`, `nginx-proxy`, `gateway`, `ingress`. A project can also be pinned protected explicitly, by exact path or by glob, and that state persists.

## Quick Start

```sh
go install github.com/carterperez-dev/holophyly/cmd/server@latest
holophyly
```

Or run it the way it is meant to run, which is in a container that never sees the real Docker socket:

```sh
just up
```

`compose.yml` puts a `docker-socket-proxy` in front of the socket with `EXEC: 0` and `SWARM: 0`, on an internal bridge network, and points holophyly at it over TCP. The API container has no socket mount at all.

Host port, data directory and scan paths are all environment. Nothing assumes a well-known port is free:

```sh
HOLOPHYLY_SERVER_PORT=51337 \
HOLOPHYLY_SERVER_HOST=127.0.0.1 \
HOLOPHYLY_SCANNER_PATHS=$HOME/dev,$HOME/projects \
holophyly
```

Configuration resolves environment over file over defaults, through koanf, with every `HOLOPHYLY_`-prefixed variable mapping onto the YAML path.

## The API

| Method | Path | What it does |
|---|---|---|
| `GET` | `/health` | liveness |
| `GET` | `/ready` | readiness, including Docker reachability |
| `GET` | `/api/projects` | every discovered project with state, class and protection |
| `GET` | `/api/projects/{id}` | one project, with its services |
| `POST` | `/api/projects/{id}/start` | compose up |
| `POST` | `/api/projects/{id}/stop` | compose down, refused if protected |
| `POST` | `/api/projects/{id}/restart` | down then up |
| `POST` | `/api/projects/{id}/protect` | pin or unpin protection |
| `PUT` | `/api/projects/{id}/name` | set a display name |
| `PUT` | `/api/projects/{id}/hidden` | hide it from listings |
| `GET` | `/api/projects/{id}/stats` | CPU, memory, network, block IO per container |
| `GET` | `/api/containers/{id}/logs` | log tail |
| `GET` | `/api/system/info` | daemon info |
| `GET` | `/api/system/storage` | images, volumes, build cache, reclaimable |
| `POST` | `/api/system/prune` | reclaim |
| `GET` | `/api/system/port/{port}` | what is holding this port |
| `GET` | `/ws/stats` | live stats stream |

Every request carries a `X-Request-ID` through the logging middleware. The chain is request id, real IP, structured `slog` logging, panic recovery and compression, with CORS defaulting to localhost on any port rather than a wildcard.

The port lookup is the endpoint that gets used most. You go to bring something up, the bind fails, and instead of grepping `docker ps` you ask holophyly which project owns it.

## The socket is the whole security story

Anything that can reach `/var/run/docker.sock` is root on the host. That is not a holophyly property, it is a Docker property, and it is why the shipped topology does not hand the socket to the API.

- `compose.yml` runs `tecnativa/docker-socket-proxy` with the socket read-only and a deliberate allowlist: containers, images, info, networks, volumes, POST and build on; `EXEC` and `SWARM` off. No `docker exec` reaches the daemon through this path.
- The two containers share an internal bridge network. The proxy is not published.
- The server binds `127.0.0.1` by default. Binding `0.0.0.0` is something you opt into by setting it, and the compose file only does that because it is behind the bridge.
- Scan paths are mounted read-only.

If you run the binary directly on the host it will use your socket, because that is what you asked for. The container path is the one that is defensible, so it is the one `just up` gives you.

## Commands

```sh
just run           # go run with live reload of config
just build         # ./bin/holophyly
just build-prod    # CGO_ENABLED=0, -w -s, static
just up            # the proxied compose topology
just logs          # follow it
just lint          # golangci-lint
just check         # lint + go vet
just test-race     # the suite under the race detector
just ci            # lint + test-race
just tidy          # go mod tidy + verify
```

`just` with no argument lists every recipe grouped by area.

## Code shape

```
holophyly/
├── cmd/server/main.go        # wiring and graceful shutdown
├── internal/
│   ├── config/               # koanf: env > file > defaults
│   ├── scanner/              # tree walk, compose discovery, classification
│   │   ├── finder.go         # the walk, with the exclusion set
│   │   └── patterns.go       # env class, tunnel and proxy detection
│   ├── project/              # the unit of work
│   │   ├── manager.go        # lifecycle, state, display metadata
│   │   └── protected.go      # exact-path and glob protection verdicts
│   ├── docker/               # client, containers, compose, stats, logs, system
│   ├── websocket/            # hub, client, protocol
│   ├── api/                  # chi routes, handlers, middleware
│   ├── store/                # persisted display names, hidden flags, pins
│   └── model/types.go        # the wire types
├── web/                      # embedded status page (go:embed)
├── compose.yml               # socket-proxy topology
└── Dockerfile                # multi-stage, static binary
```

## What it does not do

- **It is not a deployment tool.** It starts and stops compose projects that already exist on disk. It does not write compose files, build registries or push images.
- **It does not do remote hosts.** One daemon, the one it is pointed at. Multi-host is a different program.
- **It has no authentication.** It binds loopback and expects to sit behind something that does auth, which is why CORS and origins are configurable and the default is localhost only. Do not publish it.
- **The embedded UI is a status page, not the product.** The API is the product. The real interface lives in a separate workspace front end.

## License

[MIT](LICENSE).
