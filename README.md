# Dockerly

A minimal container runtime built from scratch in Go. Dockerly pulls real Docker images from Docker Hub, extracts filesystem layers, and runs an isolated shell using Linux namespaces and cgroups — demonstrating the core mechanics behind container engines like Docker and containerd.

## How It Works

Dockerly follows the same fundamental steps that Docker uses internally:

1. **Authenticate** with the Docker Hub registry using token-based auth
2. **Fetch the image manifest** from the Docker Registry v2 API, with support for multi-architecture manifest list resolution (selects linux/amd64)
3. **Download image layers** referenced in the manifest
4. **Extract layers** to construct a root filesystem
5. **Spawn a shell** inside isolated Linux namespaces (PID, UTS, IPC, NET, Mount, User)
6. **Apply cgroup limits** for CPU and memory to the containerized process
7. **Pivot root** into the new filesystem so the container sees only its own files


## Prerequisites

- Go 1.21 or later
- Linux (this project uses Linux-specific syscalls: namespaces, cgroups, pivot_root)
- For macOS/Windows: Docker Desktop (to build and run inside a Linux container)

## Usage

### Building on Linux

```bash
make build
./dockerly alpine
```

### Building and Running via Docker (macOS / Windows)

Since Dockerly uses Linux kernel primitives, it must run on Linux. You can use Docker to provide a Linux environment:

```bash
# Build the binary
docker run --rm -v $(pwd):/app -w /app golang:1.21 go build -o dockerly main.go rootfs.go cgroup.go

# Run with an image (e.g., alpine)
docker run --rm -it --privileged -v $(pwd):/app -w /app golang:1.21 ./dockerly alpine
```

Once inside the container shell, you can verify the isolation:

```bash
whoami          # root (mapped via user namespace)
hostname        # dockerly
echo $$         # PID 1 in the new PID namespace
ls /            # Alpine's filesystem, not the host
ps aux          # Only processes in this namespace
```

### Trying Different Images

```bash
./dockerly busybox
./dockerly ubuntu
```

## Technical Details

### Namespaces

Dockerly creates a new process with the following Linux namespaces:

- **CLONE_NEWPID** - Isolated process ID space (container shell becomes PID 1)
- **CLONE_NEWUTS** - Separate hostname (set to "dockerly")
- **CLONE_NEWIPC** - Isolated inter-process communication
- **CLONE_NEWNET** - Isolated network stack
- **CLONE_NEWNS** - Private mount namespace
- **CLONE_NEWUSER** - User namespace with UID/GID mapping

### Cgroups

Resource limits are applied via cgroup v1:

- Memory limit: 500MB
- CPU shares: 512

### Filesystem

The root filesystem is constructed by:

1. Downloading and extracting image layers to `/tmp/dockerly/rootfs`
2. Bind-mounting the new root onto itself
3. Using `pivot_root` to swap the filesystem root
4. Mounting a fresh `/proc` for the new PID namespace

## License

MIT
