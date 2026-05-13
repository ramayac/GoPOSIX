# Kubernetes

Run KoreGo as a sidecar container sharing a Unix socket via `emptyDir` volume.

## Pod Manifest (Minimal)

```yaml
apiVersion: v1
kind: Pod
metadata:
  name: app-with-korego
spec:
  containers:
    - name: app
      image: my-app:latest
      volumeMounts:
        - name: korego-socket
          mountPath: /var/run/korego
      env:
        - name: KOREGO_SOCKET
          value: /var/run/korego/korego.sock

    - name: korego
      image: ghcr.io/ramayac/korego:latest
      command: ["korego", "daemon", "-s", "/var/run/korego/korego.sock"]
      securityContext:
        runAsUser: 65534
        readOnlyRootFilesystem: true
      volumeMounts:
        - name: korego-socket
          mountPath: /var/run/korego

  volumes:
    - name: korego-socket
      emptyDir: {}
```

## Init Container Pattern

For workloads that need file inspection before the app starts, run KoreGo as an init container:

```yaml
spec:
  initContainers:
    - name: setup
      image: ghcr.io/ramayac/korego:latest
      command: ["korego", "ls", "--json", "/config"]
      volumeMounts:
        - name: config
          mountPath: /config
  containers:
    - name: app
      # ...
```

## DaemonSet Pattern

For node-level daemons that serve multiple pods:

```yaml
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: korego-node
spec:
  selector:
    matchLabels:
      app: korego-node
  template:
    spec:
      hostNetwork: true
      containers:
        - name: korego
          image: ghcr.io/ramayac/korego:latest
          command: ["korego", "daemon", "-s", "/var/run/korego.sock"]
          securityContext:
            privileged: false
            readOnlyRootFilesystem: true
          volumeMounts:
            - name: socket-dir
              mountPath: /var/run
      volumes:
        - name: socket-dir
          hostPath:
            path: /var/run
            type: Directory
```

> **Warning:** HostPath volumes bypass pod isolation. Only use the DaemonSet pattern in trusted clusters.

## Resource Limits

```yaml
resources:
  requests:
    memory: "32Mi"
    cpu: "50m"
  limits:
    memory: "128Mi"
    cpu: "500m"
```

The daemon is lightweight — 32 Mi is sufficient for most workloads.

## Security Context

```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 65534
  readOnlyRootFilesystem: true
  capabilities:
    drop: ["ALL"]
  allowPrivilegeEscalation: false
```

## Readiness Probe

```yaml
readinessProbe:
  exec:
    command:
      - sh
      - -c
      - |
        echo '{"jsonrpc":"2.0","method":"korego.ping","id":1}' | nc -U /var/run/korego/korego.sock
  initialDelaySeconds: 2
  periodSeconds: 5
```
