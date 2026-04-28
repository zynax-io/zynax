# Helm Chart Patterns — Reference

> Canonical templates for Zynax service Helm charts (M6+).
> Chart-testing (`ct lint`) enforces the required-resource list.
> Infrastructure rules live in `infra/AGENTS.md`.

---

## Required Chart Structure

Every service chart MUST include all of the following:

```
helm/zynax-<service>/
├── Chart.yaml
├── values.yaml              ← Defaults. No secrets.
├── values-production.yaml   ← Production overrides. No secrets.
└── templates/
    ├── _helpers.tpl
    ├── deployment.yaml      ← Required
    ├── service.yaml         ← Required (ClusterIP for gRPC)
    ├── serviceaccount.yaml  ← Required
    ├── hpa.yaml             ← Required
    ├── pdb.yaml             ← Required
    ├── networkpolicy.yaml   ← Required
    ├── configmap.yaml       ← Non-secret config only
    ├── NOTES.txt
    └── tests/
        └── test-connection.yaml
```

---

## Deployment Template

```yaml
# templates/deployment.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: {{ include "zynax.fullname" . }}
  labels: {{ include "zynax.labels" . | nindent 4 }}
spec:
  replicas: {{ .Values.replicaCount }}
  selector:
    matchLabels: {{ include "zynax.selectorLabels" . | nindent 6 }}
  strategy:
    type: RollingUpdate
    rollingUpdate:
      maxSurge: 1
      maxUnavailable: 0
  template:
    metadata:
      labels: {{ include "zynax.selectorLabels" . | nindent 8 }}
      annotations:
        prometheus.io/scrape: "true"
        prometheus.io/port: "9090"
        prometheus.io/path: "/metrics"
    spec:
      serviceAccountName: {{ include "zynax.serviceAccountName" . }}
      securityContext:
        runAsNonRoot: true
        runAsUser: 1001
        fsGroup: 1001
        seccompProfile:
          type: RuntimeDefault
      containers:
        - name: {{ .Chart.Name }}
          image: "{{ .Values.image.repository }}:{{ .Values.image.tag }}"
          imagePullPolicy: {{ .Values.image.pullPolicy }}
          securityContext:
            allowPrivilegeEscalation: false
            readOnlyRootFilesystem: true
            capabilities:
              drop: ["ALL"]
          ports:
            - name: grpc
              containerPort: 50051
            - name: metrics
              containerPort: 9090
            - name: health
              containerPort: 8080
          livenessProbe:
            httpGet: { path: /healthz, port: health }
            initialDelaySeconds: 10
            periodSeconds: 15
            failureThreshold: 3
          readinessProbe:
            httpGet: { path: /readyz, port: health }
            initialDelaySeconds: 5
            periodSeconds: 10
            failureThreshold: 3
          startupProbe:
            httpGet: { path: /startupz, port: health }
            failureThreshold: 30
            periodSeconds: 5
          resources: {{ toYaml .Values.resources | nindent 12 }}
          env:
            - name: ZYNAX_SERVICE_NAME
              value: {{ .Chart.Name }}
            - name: ZYNAX_LOG_LEVEL
              value: {{ .Values.logLevel | quote }}
          envFrom:
            - secretRef:
                name: {{ include "zynax.fullname" . }}-secrets
                optional: false
          volumeMounts:
            - name: tmp
              mountPath: /tmp
      volumes:
        - name: tmp
          emptyDir: {}
      topologySpreadConstraints:
        - maxSkew: 1
          topologyKey: kubernetes.io/hostname
          whenUnsatisfiable: DoNotSchedule
          labelSelector:
            matchLabels: {{ include "zynax.selectorLabels" . | nindent 14 }}
```

---

## HPA Template

```yaml
# templates/hpa.yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: {{ include "zynax.fullname" . }}
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: {{ include "zynax.fullname" . }}
  minReplicas: {{ .Values.autoscaling.minReplicas }}
  maxReplicas: {{ .Values.autoscaling.maxReplicas }}
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70
    - type: Resource
      resource:
        name: memory
        target:
          type: Utilization
          averageUtilization: 80
```

---

## NetworkPolicy Template (default deny-all, explicit allow)

```yaml
# templates/networkpolicy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: {{ include "zynax.fullname" . }}
spec:
  podSelector:
    matchLabels: {{ include "zynax.selectorLabels" . | nindent 6 }}
  policyTypes: [Ingress, Egress]
  ingress:
    - from:
        - podSelector:
            matchLabels:
              app.kubernetes.io/part-of: zynax
      ports:
        - protocol: TCP
          port: 50051
    - from: []   # Prometheus scraping
      ports:
        - protocol: TCP
          port: 9090
  egress:
    - to:
        - podSelector:
            matchLabels:
              app.kubernetes.io/part-of: zynax
    - to: []     # DNS
      ports:
        - protocol: UDP
          port: 53
```

---

## PodDisruptionBudget

```yaml
# templates/pdb.yaml
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: {{ include "zynax.fullname" . }}
spec:
  minAvailable: 1
  selector:
    matchLabels: {{ include "zynax.selectorLabels" . | nindent 6 }}
```

---

## values.yaml Defaults

```yaml
replicaCount: 1

image:
  repository: ghcr.io/zynax-io/zynax-<service>
  tag: "latest"          # Override with specific SHA in production
  pullPolicy: IfNotPresent

resources:
  requests:
    cpu: "100m"
    memory: "128Mi"
  limits:
    cpu: "500m"
    memory: "512Mi"

autoscaling:
  minReplicas: 1
  maxReplicas: 10

logLevel: INFO

serviceAccount:
  create: true
  annotations: {}
```
