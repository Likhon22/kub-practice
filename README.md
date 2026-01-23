# Kubernetes Microservices Project - Complete Learning Journey

A production-grade microservices application demonstrating advanced Kubernetes concepts including StatefulSets, Kustomize, RBAC, NetworkPolicies, HPA, and Ingress.

> **🎓 Learning Project**: This README documents my complete journey from basic Kubernetes deployments to production-ready configurations with security, multi-environment support, and high availability.

---

## Table of Contents

1. [Application Architecture](#application-architecture)
2. [Services Overview](#services-overview)
3. [Project Evolution](#project-evolution)
4. [Kubernetes Architecture](#kubernetes-architecture)
5. [Security Implementation](#security-implementation)
6. [Multi-Environment with Kustomize](#multi-environment-with-kustomize)
7. [Deployment Guide](#deployment-guide)
8. [Accessing the Application](#accessing-the-application)
9. [Troubleshooting](#troubleshooting)
10. [Key Concepts Learned](#key-concepts-learned)
11. [Next Steps](#next-steps)

---

## Application Architecture

```
                     ┌─────────────────┐
                     │   INGRESS       │
                     │  (nginx)        │
                     │  Port: 80       │
                     └────────┬────────┘
                              │
            ┌─────────────────┼─────────────────┐
            │                 │                 │
            ▼                 ▼                 ▼
   ┌────────────────┐ ┌────────────────┐ ┌────────────────┐
   │   FRONTEND     │ │    BACKEND     │ │     AUTH       │
   │  React/Vite    │ │   Go Server    │ │  Go Server     │
   │   Port: 80     │ │   Port: 8080   │ │  Port: 8081    │
   └────────────────┘ └───────┬────────┘ └───────┬────────┘
                              │                  │
                              └──────┬───────────┘
                                     │
                              ┌──────▼──────────────────┐
                              │    MONGODB REPLICA SET  │
                              │   (3 Nodes - rs0)       │
                              │   Port: 27017           │
                              │   - mongo-0 (PRIMARY)   │
                              │   - mongo-1 (SECONDARY) │
                              │   - mongo-2 (SECONDARY) │
                              └─────────────────────────┘
```

**Traffic Flow:**

1. External Request → Ingress (path-based routing)
2. Ingress → Services (ClusterIP)
3. Services → Pods (via labels)
4. Backend/Auth → MongoDB Replica Set

---

## Services Overview

### Backend (Go)

- **Port:** 8080
- **Database:** `backend_db` → `items` collection
- **Routes:** Served at `/api` via Ingress
- **Endpoints:**
  - `GET /api` - Service info
  - `GET /api/health` - Health check with DB status
  - `GET /api/items` - Fetch all items
  - `POST /api/items/create` - Create item

### Auth (Go)

- **Port:** 8081
- **Database:** `auth_db` → `users` collection
- **Routes:** Served at `/auth` via Ingress
- **Endpoints:**
  - `GET /auth` - Service info
  - `GET /auth/health` - Health check with DB status
  - `GET /auth/login` - Simulated login (returns fake JWT)
  - `GET /auth/users` - Fetch all users
  - `POST /auth/users/create` - Create user

### MongoDB Replica Set

- **Port:** 27017
- **Nodes:** 3 (mongo-0, mongo-1, mongo-2)
- **Replica Set:** rs0
- **Storage:** PersistentVolumeClaims (3x 1Gi)
- **Databases:** `backend_db`, `auth_db`

---

## Project Evolution

### Phase 1: Simple Deployment (kubernetes/withoutStatefulset/)

**Initial Setup:**

- MongoDB as a simple Deployment
- Manual PersistentVolume + PersistentVolumeClaim
- Single replica, no high availability
- All resources in one folder

**Structure:**

```
kubernetes/withoutStatefulset/
├── deployment/
│   └── mongo-deployment.yaml  # Single MongoDB instance
├── service/
│   └── mongo-service.yaml     # ClusterIP service
└── volumes/
    ├── mongo-pv.yaml          # Manual PersistentVolume
    └── mongo-pvc.yaml         # PersistentVolumeClaim
```

**Problems Discovered:**

- Single point of failure (no redundancy)
- Data loss if pod crashes before PVC binds
- Not suitable for production
- Manual volume management

### Phase 2: StatefulSet Migration (kubernetes/base/statefulsets/)

**Why StatefulSet?**

- **Stable pod identities**: Pods get persistent names (mongo-0, mongo-1, mongo-2)
- **Ordered deployment**: Pods start/stop in order
- **Automatic PVCs**: Each pod gets its own storage automatically
- **Perfect for databases**: Designed for stateful applications

**MongoDB Replica Set Benefits:**

- **High Availability**: If mongo-0 fails, mongo-1 or mongo-2 becomes primary
- **Data Replication**: Data automatically syncs across all 3 nodes
- **Automatic Failover**: No downtime when a node fails
- **Read Scaling**: Distribute reads across secondaries

**Configuration:**

```yaml
# StatefulSet creates 3 pods with persistent storage
spec:
  replicas: 3
  serviceName: mongo # Headless service for peer discovery
  volumeClaimTemplates: # Auto-creates PVC for each pod
    - metadata:
        name: mongo-storage
      spec:
        accessModes: ["ReadWriteOnce"]
        resources:
          requests:
            storage: 1Gi
```

**Initialization:**

```bash
# Initialize replica set (one-time setup)
kubectl exec -it mongo-0 -n dev -- mongosh --eval '
rs.initiate({
  _id: "rs0",
  members: [
    { _id: 0, host: "mongo-0.mongo.dev.svc.cluster.local:27017" },
    { _id: 1, host: "mongo-1.mongo.dev.svc.cluster.local:27017" },
    { _id: 2, host: "mongo-2.mongo.dev.svc.cluster.local:27017" }
  ]
})
'
```

### Phase 3: Multi-Environment with Kustomize

**The Problem:**

- Development needs fewer resources (dev: 2 replicas)
- Production needs more resources (prod: 5 replicas)
- Different MongoDB connection strings per environment
- Manual changes lead to mistakes

**The Solution: Kustomize**

**Kustomize Concept:**

- **Base**: Common configuration shared by all environments
- **Overlays**: Environment-specific customizations (patches)
- **No templating**: Just YAML patches (simpler than Helm for this use case)

**Directory Structure:**

```
kubernetes/
├── base/                       # Common config (defaults)
│   ├── deployment/             # Frontend, Backend, Auth
│   ├── statefulsets/           # MongoDB
│   ├── services/               # ClusterIP services
│   ├── config/                 # ConfigMaps
│   ├── secrets/                # Secrets
│   ├── hpa/                    # Horizontal Pod Autoscalers
│   ├── rbac/                   # RBAC configs
│   ├── networkpolicies/        # Security policies
│   ├── ingress.yaml            # Path-based routing
│   └── kustomization.yaml      # Base manifest aggregator
│
└── overlays/
    └── dev/                    # Development environment
        ├── kustomization.yaml  # References base + applies patches
        ├── configmap-patch.yaml # MongoDB URLs with .dev namespace
        └── replicas-patch.yaml  # Scale to 2 replicas
```

**How Kustomize Works:**

```bash
# Build and preview (doesn't apply)
kubectl kustomize kubernetes/overlays/dev/

# Apply to cluster
kubectl apply -k kubernetes/overlays/dev/
```

**Example Patch (replicas-patch.yaml):**

```yaml
# Scales all deployments to 2 replicas in dev
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend-deployment
spec:
  replicas: 2 # Overrides base (which has 1)
```

**Example Patch (configmap-patch.yaml):**

```yaml
# Fixes MongoDB DNS for dev namespace
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  MONGO_URL: mongodb://mongo-0.mongo.dev,mongo-1.mongo.dev,mongo-2.mongo.dev:27017/?replicaSet=rs0
  # Base has: mongo-0.mongo, mongo-1.mongo (missing namespace)
  # Dev patch adds: .dev suffix for proper DNS resolution
```

---

## Kubernetes Architecture

### 1. Workloads

#### Deployments (Stateless Applications)

**frontend-deployment.yaml:**

```yaml
replicas: 2 # (in dev, patched from base)
spec:
  template:
    spec:
      serviceAccountName: frontend-sa # RBAC identity
      containers:
        - name: frontend
          image: likhon22/kub_prac_frontend:latest
          resources:
            requests:
              cpu: "10m"
              memory: "32Mi"
            limits:
              cpu: "50m"
              memory: "64Mi"
```

**backend-deployment.yaml:**

- Init container: Waits for MongoDB to be ready before starting
- Liveness/Readiness probes: Health checks
- ConfigMap/Secret references: Environment variables
- ServiceAccount: backend-sa (for RBAC)

**auth-deployment.yaml:**

- Similar to backend
- Different service account: auth-sa
- Different port: 8081

#### StatefulSet (Stateful Application)

**mongo-statefulset.yaml:**

```yaml
spec:
  replicas: 3
  serviceName: mongo # Required for stable network identity
  selector:
    matchLabels:
      app: mongo
  template:
    spec:
      containers:
        - name: mongo
          image: mongo:latest
          command:
            - mongod
            - --replSet
            - rs0 # Replica set name
            - --bind_ip_all
          volumeMounts:
            - name: mongo-storage
              mountPath: /data/db
  volumeClaimTemplates: # Creates PVC for each pod automatically
    - metadata:
        name: mongo-storage
      spec:
        accessModes: ["ReadWriteOnce"]
        resources:
          requests:
            storage: 1Gi
```

**Why StatefulSet vs Deployment:**
| Feature | Deployment | StatefulSet |
|---------|-----------|-------------|
| Pod names | Random (backend-7fffff8b8d-l5jhh) | Stable (mongo-0, mongo-1) |
| Startup order | Parallel | Sequential (0→1→2) |
| Storage | Shared or manual PVC | Automatic PVC per pod |
| Network identity | Random IP | Stable DNS (mongo-0.mongo.dev) |
| Use case | Stateless apps | Databases, queues |

### 2. Networking

#### Services

**ClusterIP Services:**

```yaml
# backend-service.yaml
apiVersion: v1
kind: Service
metadata:
  name: backend-service
spec:
  type: ClusterIP # Internal only
  selector:
    app: backend # Finds pods with this label
  ports:
    - port: 8080
      targetPort: 8080
```

**Headless Service (MongoDB):**

```yaml
# mongo-headless-service.yaml
apiVersion: v1
kind: Service
metadata:
  name: mongo
spec:
  clusterIP: None # Headless!
  selector:
    app: mongo
  ports:
    - port: 27017
```

**Why Headless?**

- Returns pod IPs directly (not a virtual ClusterIP)
- Enables direct peer-to-peer communication
- Required for MongoDB replica set member discovery
- DNS returns: `mongo-0.mongo.dev`, `mongo-1.mongo.dev`, etc.

#### Ingress (External Access)

**ingress.yaml:**

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: app-ingress
spec:
  rules:
    - host: myapp.local # Domain-based routing
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: frontend-service
                port:
                  number: 80
          - path: /api
            pathType: Prefix
            backend:
              service:
                name: backend-service
                port:
                  number: 8080
          - path: /auth
            pathType: Prefix
            backend:
              service:
                name: auth-service
                port:
                  number: 8081
```

**How Path Routing Works:**

1. Request: `http://myapp.local/api/items`
2. Ingress sees path starts with `/api`
3. Routes to `backend-service:8080`
4. Backend receives: `/api/items`

### 3. Configuration & Secrets

**ConfigMap (Non-Sensitive Data):**

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  MONGO_URL: mongodb://mongo-0.mongo.dev,mongo-1.mongo.dev,mongo-2.mongo.dev:27017/?replicaSet=rs0
  LOG_LEVEL: info
```

**Secret (Sensitive Data):**

```yaml
apiVersion: v1
kind: Secret
metadata:
  name: app-secrets
type: Opaque
data:
  MONGO_PASSWORD: base64encodedpassword
```

**Usage in Deployment:**

```yaml
env:
  - name: MONGO_URL
    valueFrom:
      configMapKeyRef:
        name: app-config
        key: MONGO_URL
  - name: MONGO_PASSWORD
    valueFrom:
      secretKeyRef:
        name: app-secrets
        key: MONGO_PASSWORD
```

### 4. Auto-Scaling (HPA)

**backend-hpa.yaml:**

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: backend-hpa
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: backend-deployment
  minReplicas: 2
  maxReplicas: 10
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 70 # Scale when CPU > 70%
```

**How HPA Works:**

1. Metrics Server collects CPU/memory usage from pods
2. HPA checks average CPU every 15 seconds
3. If average > 70%, adds more replicas
4. If average < 70%, removes replicas (min 2, max 10)

**Requirements:**

- Metrics Server addon must be enabled: `minikube addons enable metrics-server`
- Pods must have resource requests defined

---

## Security Implementation

### 1. RBAC (Role-Based Access Control)

**What is RBAC?**

- Controls what pods can do with the **Kubernetes API**
- Not for network traffic (that's NetworkPolicy)
- Based on least-privilege principle

**Components:**

#### ServiceAccounts (Identity)

```yaml
# backend-sa
apiVersion: v1
kind: ServiceAccount
metadata:
  name: backend-sa
```

Each deployment uses its own ServiceAccount:

- `frontend-deployment` uses `frontend-sa`
- `backend-deployment` uses `backend-sa`
- `auth-deployment` uses `auth-sa`

#### Roles (Permissions)

```yaml
# backend-role
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: backend-role
rules:
  - apiGroups: [""] # Core API group
    resources: ["configmaps"]
    resourceNames: ["app-config"] # Only this ConfigMap
    verbs: ["get", "list"] # Read-only
```

**Why backend needs ConfigMap access:**

- Backend reads `MONGO_URL` from ConfigMap at startup
- Needs permission to call Kubernetes API: `GET /api/v1/namespaces/dev/configmaps/app-config`

**Why frontend/auth don't:**

- They get config via environment variables (injected at pod creation)
- Don't need runtime API access

#### RoleBindings (Assignment)

```yaml
# backend-rolebinding
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: backend-rolebinding
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: backend-role
subjects:
  - kind: ServiceAccount
    name: backend-sa
    namespace: dev
```

**Testing RBAC:**

```bash
# Test if backend-sa can get configmaps
kubectl auth can-i get configmaps/app-config \
  --as=system:serviceaccount:dev:backend-sa -n dev
# Output: yes

# Test if backend-sa can get secrets (should fail)
kubectl auth can-i get secrets \
  --as=system:serviceaccount:dev:backend-sa -n dev
# Output: no
```

### 2. NetworkPolicies (Network Segmentation)

**What are NetworkPolicies?**

- Firewall rules for pod-to-pod traffic
- By default, all pods can talk to all pods
- NetworkPolicies are **deny-by-default**: once applied, only explicitly allowed traffic passes

**Why Needed?**
Without NetworkPolicies:

- Frontend can directly access MongoDB ❌ (security risk!)
- Compromised frontend = full database access
- No defense in depth

With NetworkPolicies:

- Frontend → MongoDB: ❌ BLOCKED
- Frontend → Backend: ✅ Allowed
- Backend → MongoDB: ✅ Allowed
- Defense in depth: Multiple security layers

#### CNI Plugin: Calico

**Why Calico?**

- Minikube's default CNI doesn't support NetworkPolicies
- Calico enforces NetworkPolicy rules using iptables
- Required for NetworkPolicies to work

**Installation:**

```bash
minikube delete  # Clean slate
minikube start --cni=calico --memory=4096
```

**Verification:**

```bash
kubectl get pods -n kube-system | grep calico
# calico-kube-controllers-xxx   1/1   Running
# calico-node-xxx               1/1   Running
```

#### NetworkPolicy Examples

**mongo-policy.yaml:**

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: mongo-allow-backend-auth
spec:
  podSelector:
    matchLabels:
      app: mongo # Protect MongoDB pods
  policyTypes:
    - Ingress # Control incoming traffic
  ingress:
    # Rule 1: Allow backend
    - from:
        - podSelector:
            matchLabels:
              app: backend
      ports:
        - protocol: TCP
          port: 27017
    # Rule 2: Allow auth
    - from:
        - podSelector:
            matchLabels:
              app: auth
      ports:
        - protocol: TCP
          port: 27017
    # Rule 3: Allow mongo-to-mongo (for replica set)
    - from:
        - podSelector:
            matchLabels:
              app: mongo
      ports:
        - protocol: TCP
          port: 27017
```

**Effect:**

- Backend → MongoDB: ✅ Allowed
- Auth → MongoDB: ✅ Allowed
- Mongo → Mongo: ✅ Allowed (replica set sync)
- Frontend → MongoDB: ❌ BLOCKED (not in allowed list)

**frontend-policy.yaml:**

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: frontend-allow-ingress
spec:
  podSelector:
    matchLabels:
      app: frontend
  policyTypes:
    - Ingress
  ingress:
    - from:
        - namespaceSelector:
            matchLabels:
              name: ingress-nginx # Only ingress controller
      ports:
        - protocol: TCP
          port: 80
```

**Critical Detail: Namespace Labels**

The ingress-nginx namespace must have the label `name=ingress-nginx`:

```bash
# Check current labels
kubectl get namespace ingress-nginx --show-labels

# Add label if missing
kubectl label namespace ingress-nginx name=ingress-nginx
```

**Why?** NetworkPolicy's `namespaceSelector` matches by labels, not names!

#### Testing NetworkPolicies

**Test 1: Frontend blocked from MongoDB**

```bash
kubectl exec -it deployment/frontend-deployment -n dev -- sh
# Inside pod:
nc -zv mongo-0.mongo.dev 27017
# Expected: Connection timed out ✅
```

**Test 2: Backend allowed to MongoDB**

```bash
kubectl exec -it deployment/backend-deployment -n dev -- sh
# Inside pod:
nc -zv mongo-0.mongo.dev 27017
# Expected: mongo-0.mongo.dev (10.x.x.x:27017) open ✅
```

**Test 3: Ingress can reach frontend**

```bash
kubectl exec -it -n ingress-nginx deployment/ingress-nginx-controller -- /bin/bash
# Inside ingress pod:
curl --max-time 3 http://frontend-service.dev.svc.cluster.local/
# Expected: HTML response ✅
```

---

## Multi-Environment with Kustomize

### Base Configuration (kubernetes/base/)

**Default values (production-ready):**

- Replicas: 1 (minimal)
- No namespace (applied to default or overlays specify)
- Generic MongoDB URLs (mongo-0.mongo, mongo-1.mongo)

**kustomization.yaml:**

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

resources:
  # Order matters! Dependencies first
  - config/app-config.yaml
  - secrets/app-secrets.yaml
  - rbac/serviceaccounts.yaml
  - rbac/roles.yaml
  - rbac/rolebindings.yaml
  - statefulsets/mongo-statefulset.yaml
  - services/mongo-headless-service.yaml
  - deployment/auth-deployment.yaml
  - deployment/backend-deployment.yaml
  - deployment/frontend-deployment.yaml
  - services/auth-service.yaml
  - services/backend-service.yaml
  - services/frontend-service.yaml
  - hpa/auth-hpa.yaml
  - hpa/backend-hpa.yaml
  - ingress.yaml
  - networkpolicies/mongo-policy.yaml
  - networkpolicies/backend-policy.yaml
  - networkpolicies/auth-policy.yaml
  - networkpolicies/frontend-policy.yaml
```

### Development Overlay (kubernetes/overlays/dev/)

**Customizations for dev:**

- Namespace: `dev`
- Replicas: 2 (vs 1 in base)
- MongoDB URLs: with `.dev` namespace suffix
- Labels: `environment: dev`

**kustomization.yaml:**

```yaml
apiVersion: kustomize.config.k8s.io/v1beta1
kind: Kustomization

namespace: dev # Apply all resources to dev namespace

commonLabels:
  environment: dev # Add label to all resources

bases:
  - ../../base # Reference base configuration

patches:
  - path: configmap-patch.yaml
  - path: replicas-patch.yaml
```

**configmap-patch.yaml:**

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: app-config
data:
  # Fix DNS: add .dev namespace
  MONGO_URL: mongodb://mongo-0.mongo.dev,mongo-1.mongo.dev,mongo-2.mongo.dev:27017/?replicaSet=rs0
```

**replicas-patch.yaml:**

```yaml
# Patch all deployments to 2 replicas
apiVersion: apps/v1
kind: Deployment
metadata:
  name: auth-deployment
spec:
  replicas: 2
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend-deployment
spec:
  replicas: 2
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: frontend-deployment
spec:
  replicas: 2
```

### How Kustomize Merges

**Example: backend-deployment**

**Base (kubernetes/base/deployment/backend-deployment.yaml):**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend-deployment
spec:
  replicas: 1 # Default
  template:
    spec:
      serviceAccountName: backend-sa
      containers:
        - name: backend
          image: likhon22/kub_prac_backend:latest
```

**After applying dev overlay:**

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: backend-deployment
  namespace: dev # Added by overlay
  labels:
    environment: dev # Added by overlay
spec:
  replicas: 2 # Patched from 1 → 2
  template:
    metadata:
      labels:
        environment: dev # Added by overlay
    spec:
      serviceAccountName: backend-sa
      containers:
        - name: backend
          image: likhon22/kub_prac_backend:latest
```

### Preview vs Apply

**Preview (dry-run):**

```bash
# Build and show merged YAML (doesn't apply to cluster)
kubectl kustomize kubernetes/overlays/dev/ > preview.yaml
cat preview.yaml
```

**Apply:**

```bash
# Build and apply to cluster
kubectl apply -k kubernetes/overlays/dev/

# Important: Use -k flag, NOT -f!
# -k = kustomize (merges patches)
# -f = raw files (treats patches as complete resources, fails!)
```

---

## Deployment Guide

### Prerequisites

**1. Install Minikube:**

```bash
# Linux
curl -LO https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64
sudo install minikube-linux-amd64 /usr/local/bin/minikube

# Verify
minikube version
```

**2. Install kubectl:**

```bash
# Linux
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Verify
kubectl version --client
```

### Step-by-Step Deployment

**Step 1: Start Minikube with Calico**

```bash
# Delete existing cluster (if any)
minikube delete

# Start fresh with Calico CNI
minikube start --cni=calico --memory=4096 --cpus=2

# Wait for Calico to be ready
kubectl get pods -n kube-system | grep calico
# Wait until both calico pods show Running
```

**Step 2: Enable Required Addons**

```bash
# Ingress controller (for HTTP routing)
minikube addons enable ingress

# Metrics server (for HPA)
minikube addons enable metrics-server

# Verify ingress is ready
kubectl get pods -n ingress-nginx
# Wait until all pods show Running (takes ~1 minute)
```

**Step 3: Add Namespace Label for NetworkPolicies**

```bash
# NetworkPolicies reference this label
kubectl label namespace ingress-nginx name=ingress-nginx

# Verify
kubectl get namespace ingress-nginx --show-labels
# Should show: name=ingress-nginx
```

**Step 4: Create dev Namespace**

```bash
kubectl create namespace dev

# Verify
kubectl get namespaces
```

**Step 5: Deploy Application**

```bash
# Navigate to project root
cd /path/to/Project_1

# Apply dev configuration
kubectl apply -k kubernetes/overlays/dev/

# Wait for pods to be ready (takes 2-3 minutes)
kubectl get pods -n dev -w
# Press Ctrl+C when all pods show Running
```

**Step 6: Initialize MongoDB Replica Set**

```bash
# Wait for all 3 MongoDB pods to be Running
kubectl get pods -n dev -l app=mongo

# Initialize replica set (one-time setup)
kubectl exec -it mongo-0 -n dev -- mongosh --eval '
rs.initiate({
  _id: "rs0",
  members: [
    { _id: 0, host: "mongo-0.mongo.dev.svc.cluster.local:27017" },
    { _id: 1, host: "mongo-1.mongo.dev.svc.cluster.local:27017" },
    { _id: 2, host: "mongo-2.mongo.dev.svc.cluster.local:27017" }
  ]
})
'

# Verify replica set status (wait ~10 seconds)
kubectl exec -it mongo-0 -n dev -- mongosh --eval "rs.status()" --quiet | grep -E "name|stateStr"
# Should show:
# "name" : "mongo-0..." , "stateStr" : "PRIMARY"
# "name" : "mongo-1..." , "stateStr" : "SECONDARY"
# "name" : "mongo-2..." , "stateStr" : "SECONDARY"
```

**Step 7: Restart Backend/Auth (to reconnect to initialized replica set)**

```bash
kubectl rollout restart deployment backend-deployment -n dev
kubectl rollout restart deployment auth-deployment -n dev

# Verify MongoDB connection
kubectl logs -n dev deployment/backend-deployment | grep -i mongo
# Should show: "Connected to MongoDB successfully"
```

**Step 8: Verify All Resources**

```bash
# Check all resources
kubectl get all -n dev

# Check NetworkPolicies
kubectl get networkpolicies -n dev

# Check HPA (should show CPU percentages, not <unknown>)
kubectl get hpa -n dev

# Check Ingress
kubectl get ingress -n dev
```

---

## Accessing the Application

### The Challenge: Minikube with Docker Driver

**Problem:**

- Minikube runs in a Docker container
- Your host machine can't directly reach Minikube's internal IP
- Standard port-forward or tunnel approaches needed

### Solution: Port Forwarding

**Step 1: Update /etc/hosts**

```bash
# Remove any existing entries
sudo sed -i '/myapp.local/d' /etc/hosts

# Add localhost entry
echo "127.0.0.1 myapp.local" | sudo tee -a /etc/hosts

# Verify
cat /etc/hosts | grep myapp
# Should show: 127.0.0.1 myapp.local
```

**Step 2: Port Forward Ingress Controller**

```bash
# Terminal 1 - Keep this running
kubectl port-forward -n ingress-nginx service/ingress-nginx-controller 8080:80

# You'll see:
# Forwarding from 127.0.0.1:8080 -> 80
# Forwarding from [::1]:8080 -> 80
```

**Step 3: Access Application**

```bash
# In a browser:
http://myapp.local:8080/

# Or via curl:
curl http://myapp.local:8080/           # Frontend
curl http://myapp.local:8080/api        # Backend API
curl http://myapp.local:8080/auth       # Auth service
```

### Understanding the Flow

```
Browser: http://myapp.local:8080/api
    ↓
DNS lookup: /etc/hosts → myapp.local = 127.0.0.1
    ↓
Connect to: localhost:8080
    ↓
kubectl port-forward (listening on 8080)
    ↓
Forwards to: ingress-nginx-controller:80 (inside Minikube)
    ↓
Ingress checks: Host=myapp.local, Path=/api
    ↓
Routes to: backend-service:8080
    ↓
Service routes to: backend pod (via label: app=backend)
    ↓
Backend pod responds with JSON
    ↓
Response travels back: Pod → Service → Ingress → kubectl → Browser
```

### Why :8080 in the URL?

- Default HTTP port is 80
- Port 80 might be used by Apache/nginx on your host
- kubectl port-forward listens on 8080 instead
- You must specify `:8080` in the URL

### Alternative: Direct Pod Access (for debugging)

```bash
# Access frontend pod directly
kubectl port-forward -n dev deployment/frontend-deployment 3000:80

# Access backend pod directly
kubectl port-forward -n dev deployment/backend-deployment 8080:8080

# Then:
curl http://localhost:3000/  # Frontend
curl http://localhost:8080/api  # Backend
```

---

## Troubleshooting

### 1. Pods Not Starting

**Symptom:**

```bash
kubectl get pods -n dev
# NAME                       READY   STATUS    RESTARTS
# backend-deployment-xxx     0/1     Pending   0
```

**Diagnosis:**

```bash
# Check events
kubectl describe pod <pod-name> -n dev

# Common issues:
# - Insufficient resources
# - Image pull errors
# - PVC not binding
```

**Solutions:**

```bash
# Check cluster resources
kubectl top nodes

# Check if image exists
docker pull likhon22/kub_prac_backend:latest

# Check PVCs
kubectl get pvc -n dev
```

### 2. Service Endpoints Empty

**Symptom:**

```bash
kubectl get endpoints -n dev
# NAME               ENDPOINTS
# backend-service    <none>
```

**Diagnosis:**
Service selector doesn't match pod labels!

```bash
# Check service selector
kubectl get svc backend-service -n dev -o yaml | grep -A3 selector

# Check pod labels
kubectl get pods -n dev --show-labels

# Labels must match!
```

**Solution:**
Fix selector in service or labels in deployment to match.

### 3. NetworkPolicy Blocking Traffic

**Symptom:**

```bash
# From inside ingress controller pod:
curl http://frontend-service.dev.svc.cluster.local/
# Hangs/times out
```

**Diagnosis:**
NetworkPolicy blocking ingress-nginx namespace!

**Solution:**

```bash
# Add required label to ingress-nginx namespace
kubectl label namespace ingress-nginx name=ingress-nginx --overwrite

# Verify
kubectl get namespace ingress-nginx --show-labels

# Re-test after 10 seconds
```

### 4. MongoDB Replica Set Not Initialized

**Symptom:**

```bash
kubectl logs -n dev deployment/backend-deployment
# Error: no replset config has been received
# Or: ReplicaSetNoPrimary
```

**Solution:**

```bash
# Initialize replica set
kubectl exec -it mongo-0 -n dev -- mongosh --eval '
rs.initiate({
  _id: "rs0",
  members: [
    { _id: 0, host: "mongo-0.mongo.dev.svc.cluster.local:27017" },
    { _id: 1, host: "mongo-1.mongo.dev.svc.cluster.local:27017" },
    { _id: 2, host: "mongo-2.mongo.dev.svc.cluster.local:27017" }
  ]
})
'

# Restart backend/auth to reconnect
kubectl rollout restart deployment backend-deployment -n dev
kubectl rollout restart deployment auth-deployment -n dev
```

### 5. HPA Shows `<unknown>` CPU

**Symptom:**

```bash
kubectl get hpa -n dev
# NAME          TARGETS         MINPODS   MAXPODS
# backend-hpa   <unknown>/70%   2         10
```

**Diagnosis:**
Metrics server not installed or not ready.

**Solution:**

```bash
# Enable metrics server
minikube addons enable metrics-server

# Wait 1-2 minutes, then check
kubectl get hpa -n dev
# Should show actual CPU percentage (e.g., 15%/70%)
```

### 6. Ingress Returns 404

**Symptom:**

```bash
curl http://myapp.local:8080/api
# 404 Not Found
```

**Diagnosis:**
Path routing not configured correctly.

**Solution:**

```bash
# Check ingress rules
kubectl describe ingress app-ingress -n dev

# Check backend service exists
kubectl get svc backend-service -n dev

# Check endpoints
kubectl get endpoints backend-service -n dev
# Should show pod IPs, not <none>
```

### 7. Can't Access Application from Browser

**Symptom:**
Browser shows "Connection refused" or times out.

**Checklist:**

```bash
# 1. Is port-forward running?
ps aux | grep "port-forward"
# Should show: kubectl port-forward ... 8080:80

# 2. Is /etc/hosts configured?
cat /etc/hosts | grep myapp
# Should show: 127.0.0.1 myapp.local

# 3. Are you using :8080 in URL?
# Correct: http://myapp.local:8080/
# Wrong:   http://myapp.local/

# 4. Is ingress controller ready?
kubectl get pods -n ingress-nginx
# All pods should be Running
```

---

## Key Concepts Learned

### 1. Kubernetes Core Concepts

✅ **Deployments**: Stateless applications with rolling updates  
✅ **StatefulSets**: Stateful applications with stable identities  
✅ **Services**: Stable network endpoints (ClusterIP, Headless)  
✅ **Ingress**: HTTP/HTTPS routing to services  
✅ **ConfigMaps**: Non-sensitive configuration  
✅ **Secrets**: Sensitive data (passwords, tokens)  
✅ **PersistentVolumes**: Storage for stateful apps  
✅ **Namespaces**: Resource isolation

### 2. Advanced Kubernetes

✅ **Kustomize**: Multi-environment configuration management  
✅ **RBAC**: Role-Based Access Control for API security  
✅ **NetworkPolicies**: Pod-to-pod network segmentation  
✅ **HPA**: Horizontal Pod Autoscaler for dynamic scaling  
✅ **Liveness/Readiness Probes**: Health checks  
✅ **Init Containers**: Pre-startup tasks  
✅ **Labels & Selectors**: Service discovery mechanism

### 3. MongoDB Concepts

✅ **Replica Set**: High availability with automatic failover  
✅ **Primary/Secondary**: Read-write vs read-only nodes  
✅ **Replication**: Automatic data sync across nodes  
✅ **Headless Service**: Direct pod-to-pod discovery  
✅ **StatefulSet for Databases**: Stable pod identities

### 4. Networking & Security

✅ **CNI Plugins**: Calico for NetworkPolicy enforcement  
✅ **Service Discovery**: DNS-based pod lookup  
✅ **Ingress Controllers**: nginx for HTTP routing  
✅ **Path-Based Routing**: Multiple services on one domain  
✅ **Network Isolation**: Defense in depth with NetworkPolicies

### 5. DevOps Best Practices

✅ **Infrastructure as Code**: Everything in YAML  
✅ **Declarative Configuration**: Desired state, not imperative commands  
✅ **Multi-Environment**: Base + overlays pattern  
✅ **Least Privilege**: RBAC with minimal permissions  
✅ **Defense in Depth**: Multiple security layers (RBAC + NetworkPolicies)  
✅ **Health Checks**: Automatic pod restart on failure  
✅ **Resource Management**: CPU/memory requests & limits

### 6. Troubleshooting Skills

✅ **kubectl debug commands**: describe, logs, exec, get events  
✅ **Service discovery**: Understanding endpoints and selectors  
✅ **Network debugging**: Testing connectivity between pods  
✅ **Label matching**: How selectors find resources  
✅ **DNS resolution**: Kubernetes internal DNS  
✅ **Port forwarding**: Accessing services from localhost

---

## Next Steps

### Immediate Next Steps (This Week)

**1. ✅ Push to GitHub**

- This is a portfolio-worthy project!
- Shows production-grade Kubernetes knowledge
- Demonstrates security best practices

**2. 📊 Add Monitoring (Recommended)**

- Install Prometheus + Grafana using Helm
- Create dashboards for your services
- See metrics that drive your HPA decisions
- Essential for production operations

**3. 🚀 Create Production Overlay**

- `overlays/prod/` with different settings
- Higher replicas (5 instead of 2)
- Increased resource limits
- Production-grade MongoDB configuration

### Medium Term (Next 2-3 Weeks)

**4. 🔄 CI/CD Pipeline**

- GitHub Actions for automated deployments
- Build Docker images on push
- Deploy to dev/prod automatically
- Run tests before deployment

**5. 📦 Learn Helm**

- Use Helm to install third-party apps (like Prometheus)
- Understand when to use Helm vs Kustomize
- Compare templating approaches

**6. 📝 Centralized Logging**

- Install Grafana Loki
- Aggregate logs from all pods
- Search across services
- Log retention policies

### Long Term (When Needed)

**7. 🕸️ Service Mesh (Istio)**

- Mutual TLS between services
- Advanced traffic routing (canary, blue-green)
- Circuit breakers and retries
- Distributed tracing

**8. 📈 Advanced Monitoring**

- Prometheus alerts
- PagerDuty integration
- SLO/SLI monitoring
- Custom metrics

**9. 🔒 Enhanced Security**

- Pod Security Policies/Standards
- Image scanning (Trivy)
- Secrets management (Vault)
- OPA/Gatekeeper for policy enforcement

---

## Project Statistics

**Total Kubernetes Resources:** 30+

- 3 Deployments (Frontend, Backend, Auth)
- 1 StatefulSet (MongoDB)
- 4 Services (3 ClusterIP, 1 Headless)
- 1 Ingress
- 2 HPA
- 3 ServiceAccounts
- 3 Roles
- 3 RoleBindings
- 4 NetworkPolicies
- 1 ConfigMap
- 1 Secret

**Lines of YAML:** 1500+
**Namespaces:** 2 (dev, monitoring ready)
**Security Layers:** 2 (RBAC + NetworkPolicies)
**High Availability:** 3-node MongoDB replica set
**Auto-Scaling:** Dynamic 2-10 replicas based on CPU

---

## Acknowledgments

This project was built as a hands-on learning journey to master production-grade Kubernetes concepts. Special focus on:

- Deep understanding over quick tutorials
- Problem-first, solution-second approach
- Real-world debugging and troubleshooting
- Security from the start, not as an afterthought

**Technologies Used:**

- Kubernetes 1.28+
- Minikube with Calico CNI
- MongoDB 7.0 (Replica Set)
- Go 1.21 (Backend/Auth services)
- React 18 + Vite (Frontend)
- Kustomize (Multi-environment config)
- nginx Ingress Controller

---

## License

MIT License - Feel free to use this for learning!

---

**🎓 Remember:** This README documents my learning journey. If you're using this to learn, don't skip the troubleshooting section - the mistakes and fixes are where the real learning happens!
| ------------------- | ------ | --------------------------------------------------------- |
| `/api` | GET | Returns service info and status message |
| `/api/health` | GET | Returns health status including MongoDB connection status |
| `/api/items` | GET | Fetches all items from `items` collection in MongoDB |
| `/api/items/create` | POST | Creates a new item and saves it to MongoDB |

#### Endpoint Details

**GET /api**

- Returns basic service information
- Response: `{"service": "backend", "message": "Hello from Backend API!", "status": "ok"}`

**GET /api/health**

- Checks service and database health
- Response includes `db_status: "connected"` or `"disconnected"`

**GET /api/items**

- Queries MongoDB `backend_db.items` collection
- Returns all stored items with count
- Response: `{"count": 2, "items": [...]}`

**POST /api/items/create**

- Inserts a new document into `backend_db.items` collection
- Request body: `{"name": "My Item"}`
- MongoDB auto-generates `_id` and code adds `created_at` timestamp
- Response: `{"message": "Item created successfully", "item": {...}}`

---

### Auth (Go)

- **Port:** 8081
- **Stack:** Go HTTP Server + MongoDB
- **Database:** `auth_db`
- **Collection:** `users`

#### Data Model

```json
{
  "id": "MongoDB ObjectID",
  "username": "string",
  "email": "string",
  "created_at": "timestamp"
}
```

#### Endpoints

| Endpoint             | Method | Description                                               |
| -------------------- | ------ | --------------------------------------------------------- |
| `/auth`              | GET    | Returns service info and status message                   |
| `/auth/health`       | GET    | Returns health status including MongoDB connection status |
| `/auth/login`        | GET    | Simulates login, returns a fake JWT token                 |
| `/auth/users`        | GET    | Fetches all users from `users` collection in MongoDB      |
| `/auth/users/create` | POST   | Creates a new user and saves it to MongoDB                |

#### Endpoint Details

**GET /auth**

- Returns basic service information
- Response: `{"service": "auth", "message": "Hello from Auth Service!", "status": "ok"}`

**GET /auth/health**

- Checks service and database health
- Response includes `db_status: "connected"` or `"disconnected"`

**GET /auth/login**

- Simulates authentication (no real validation)
- Response: `{"service": "auth", "action": "login", "status": "simulated_success", "token": "fake-jwt-token-12345"}`

**GET /auth/users**

- Queries MongoDB `auth_db.users` collection
- Returns all stored users with count
- Response: `{"count": 2, "users": [...]}`

**POST /auth/users/create**

- Inserts a new document into `auth_db.users` collection
- Request body: `{"username": "john", "email": "john@example.com"}`
- MongoDB auto-generates `_id` and code adds `created_at` timestamp
- Response: `{"message": "User created successfully", "user": {...}}`

---

### MongoDB

- **Port:** 27017
- **Databases:**
  - `backend_db` - Used by Backend service
  - `auth_db` - Used by Auth service

---

## Project Structure

```
Project_1/
├── frontend/
│   ├── src/
│   │   ├── App.jsx      # Main component with API calls
│   │   ├── main.jsx     # React entry point
│   │   └── index.css    # Styles
│   ├── index.html
│   ├── package.json
│   ├── vite.config.js
│   └── .env
├── backend/
│   ├── main.go          # Go server with MongoDB CRUD
│   ├── go.mod
│   └── .env
├── auth/
│   ├── main.go          # Go server with MongoDB CRUD
│   ├── go.mod
│   └── .env
└── README.md
```

## Environment Variables

### Frontend

| Variable           | Default                 | Description         |
| ------------------ | ----------------------- | ------------------- |
| `VITE_BACKEND_URL` | `http://localhost:8080` | Backend service URL |
| `VITE_AUTH_URL`    | `http://localhost:8081` | Auth service URL    |

### Backend

| Variable    | Default                     | Description            |
| ----------- | --------------------------- | ---------------------- |
| `PORT`      | `8080`                      | Server port            |
| `MONGO_URL` | `mongodb://localhost:27017` | MongoDB connection URL |

### Auth

| Variable    | Default                     | Description            |
| ----------- | --------------------------- | ---------------------- |
| `PORT`      | `8081`                      | Server port            |
| `MONGO_URL` | `mongodb://localhost:27017` | MongoDB connection URL |

## Running Locally

```bash
# 1. Start MongoDB
docker run -d -p 27017:27017 --name mongo mongo:latest

# 2. Frontend (Terminal 1)
cd frontend && npm install && npm run dev

# 3. Backend (Terminal 2)
cd backend && go mod tidy && go run main.go

# 4. Auth (Terminal 3)
cd auth && go mod tidy && go run main.go
```

## API Examples

```bash
# Backend - Create item (saves to MongoDB)
curl -X POST http://localhost:8080/api/items/create \
  -H "Content-Type: application/json" \
  -d '{"name": "Test Item"}'

# Backend - Get all items (reads from MongoDB)
curl http://localhost:8080/api/items

# Auth - Create user (saves to MongoDB)
curl -X POST http://localhost:8081/auth/users/create \
  -H "Content-Type: application/json" \
  -d '{"username": "john", "email": "john@example.com"}'

# Auth - Get all users (reads from MongoDB)
curl http://localhost:8081/auth/users
```

---

## Kubernetes Architecture

This project uses **Kubernetes** for container orchestration, managed via **Kustomize** to handle configuration differences between environments (e.g., Base vs Dev).

### Directory Structure

The `kubernetes/` directory is organized into `base` (common configurations) and `overlays` (environment-specific variations).

```
kubernetes/
├── base/                       # Shared configuration for all environments
│   ├── deployment/             # Stateless application deployments (Frontend, Backend, Auth)
│   ├── statefulsets/           # Stateful application settings (MongoDB)
│   ├── services/               # Internal networking (ClusterIP, Headless Services)
│   ├── ingress.yaml            # External access configuration
│   ├── config/                 # ConfigMaps for non-sensitive data
│   ├── secrets/                # Secrets for sensitive data
│   ├── hpa/                    # Horizontal Pod Autoscalers for auto-scaling
│   ├── networkpolicies/        # Security rules controlling traffic flow
│   ├── rbac/                   # Role-Based Access Control settings
│   └── kustomization.yaml      # Aggregates all base resources
│
├── overlays/
│   └── dev/                    # Development environment customization
│       ├── kustomization.yaml  # Applies patches and namespace
│       ├── configmap-patch.yaml
│       └── replicas-patch.yaml
│
└── withoutStatefulset/         # Alternative configuration excluding MongoDB StatefulSet
```

### Components Detail

#### 1. Workloads

- **Deployments (`deployment/`)**: Manages the stateless application components:
  - `frontend-deployment.yaml`: Runs the React application.
  - `backend-deployment.yaml`: Runs the Backend Go API.
  - `auth-deployment.yaml`: Runs the Auth Go API.
- **StatefulSet (`statefulsets/`)**:
  - `mongo-statefulset.yaml`: Manages the MongoDB database. Unlike Deployments, StatefulSets maintain a sticky identity for each pod (e.g., `mongo-0`, `mongo-1`). It creates a 3-node ReplicaSet (`rs0`) for high availability and uses `PersistentVolumeClaims` to create persistent storage (`/data/db`).

#### 2. Networking

- **Services (`services/`)**: Stable endpoints for pods.
  - `frontend-service`, `backend-service`, `auth-service`: Standard ClusterIPs.
  - `mongo-headless-service`: A Headless service (ClusterIP: None) that allows direct peer-to-peer discovery between MongoDB pods.
- **Ingress (`ingress.yaml`)**: Manages external access to the services (HTTP/HTTPS routing), typically mapping paths or domains to the internal services.
- **Network Policies (`networkpolicies/`)**: Fine-grained traffic control.
  - Ensures tight security by explicitly defining which services can talk to each other (e.g., allowing Frontend to talk to Backend, but restricting direct database access).

#### 3. Configuration & Scaling

- **ConfigMaps & Secrets**: Manage environment variables and sensitive credentials without hardcoding them in the image.
- **HPA (`hpa/`)**: Horizontal Pod Autoscalers automatically increase or decrease the number of Backend and Auth pods based on CPU utilization or other metrics.

### Environments (Overlays)

#### Development (`overlays/dev`)

The `dev` overlay specifically customizes the base configuration for development purposes:

- **Namespace**: Deploys everything into the `dev` namespace to avoid conflicts.
- **Labels**: Adds `environment: dev` labels to all resources.
- **Patches**:
  - `replicas-patch.yaml`: Modifies the number of replicas (likely reducing them to save resources).
  - `configmap-patch.yaml`: Overrides specific configuration values (e.g., logging levels, endpoints) suitable for development.

### Deployment Guide

To deploy the application to your Kubernetes cluster (e.g., Minikube, Docker Desktop, or a cloud provider), use **Kustomize**.

**Deploy Development Environment:**

```bash
# Apply the configuration defined in overlays/dev
kubectl apply -k kubernetes/overlays/dev
```

**Verify Deployment:**

```bash
# Check all resources in the dev namespace
kubectl get all -n dev

# Check persistent volumes for MongoDB
kubectl get pv,pvc -n dev
```

**Clean Up:**

```bash
kubectl delete -k kubernetes/overlays/dev
```
