# cuebectl

`cuebectl` takes a collection of [cue files](https://cuelang.org/) that describe kubernetes resources, and continually reconciles them with 
cluster state, to allow declarative management of kubernetes resources.

## Project Status

**Exploratory** - 

this is still largely a proof of concept to work out UX and explore features. But it's in a great state
to try out, learn about CUE, and suggest ideas.

## Installation

`cuebectl` can be installed standalone or as a kubectl plugin (`kubectl-cue`).

## Example

Without cuebectl, multiple imperative steps:

```sh
$ cat role.yaml
apiVersion: "rbac.authorization.k8s.io/v1"
kind: "Role"
metadata:
  namespace: mynamespace
  generateName: "test-"
rules:
 - apiGroups: ["*"]
   resources: ["*"]
   verbs: ["*"]


$ kubectl -n mynamespace create role.yaml
$ kubectl -n mynamespace get roles -o name
test-xwtdt
# reference the new role in the rolebinding
$ vim rolebinding.yaml
$ cat rolebinding.yaml
apiVersion: "rbac.authorization.k8s.io/v1"
kind: "RoleBinding"
metadata: generateName: "test-"
roleRef:
  apiGroup: "rbac.authorization.k8s.io"
  kind: "Role"
  name: "test-xwtdt"
subjects:
 - kind: "ServiceAccount"
   name: default
   namespace: mynamespace
 
$ kubectl create rolbinding.yaml
```

With cuebectl, a single declarative application:

```sh
# cat manifests/pkg.cue
package role

import (
    rbacv1 "k8s.io/api/rbac/v1"
)

ns: "mynamespace"

Role: rbacv1.#Role & {
    apiVersion: "rbac.authorization.k8s.io/v1"
    kind: "Role"
    metadata: generateName: "test-"
    metadata: namespace: ns 
    rules: [
        {
            apiGroups: ["*"]
            resources: ["*"]
            verbs: ["*"]
        }
    ]
}

RoleBinding: rbacv1.#RoleBinding & {
    apiVersion: "rbac.authorization.k8s.io/v1"
    kind: "RoleBinding"
    metadata: generateName: "test-"
    metadata: namespace: ns
    roleRef: {
        apiGroup: "rbac.authorization.k8s.io"
        kind: "Role"
        name: Role.metadata.name
    }
    subjects: [
        {
            kind: "ServiceAccount"
            name: "default"
            namespace: ns
        }
    ]
}

$ cuebectl apply manifests
RoleBinding not yet concrete: RoleBinding.roleRef.name: undefined field name (and 2 more errors)
created Role: mynamespace/test-kdt66 (rbac.authorization.k8s.io/v1, Kind=Role)
created RoleBinding: mynamespace/test-72qmg (rbac.authorization.k8s.io/v1, Kind=RoleBinding)
```

See [example](./example/pkg.cue) for a longer example:

```sh
$ cuebectl apply example
TestServiceAccount not yet concrete: TestServiceAccount.metadata.namespace: undefined field name
TestClusterRoleBinding not yet concrete: TestClusterRoleBinding.roleRef.name: undefined field name (and 2 more errors)
created NoGenNameServiceAccount: default/test (/v1, Kind=ServiceAccount)
created TestNs: /test-ns-xwtdt (/v1, Kind=Namespace)
DependentClusterRoleBinding not yet concrete: DependentClusterRoleBinding.roleRef.name: undefined field name (and 1 more errors)
TestClusterRoleBinding not yet concrete: TestClusterRoleBinding.roleRef.name: undefined field name (and 1 more errors)
created TestClusterRole: /test-kdt66 (rbac.authorization.k8s.io/v1, Kind=ClusterRole)
created TestServiceAccount: test-ns-xwtdt/test-sa-cg6lj (/v1, Kind=ServiceAccount)
Operation cannot be fulfilled on serviceaccounts "test": the object has been modified; please apply your changes to the latest version and try again
created DependentClusterRoleBinding: /test-bp2kr (rbac.authorization.k8s.io/v1, Kind=ClusterRoleBinding)
created TestClusterRoleBinding: /test-72qmg (rbac.authorization.k8s.io/v1, Kind=ClusterRoleBinding)
```

## How does it work? 

The CUE instance provided to `cuebectl apply` is continually reconciled with the current state of the cluster. As new values become concrete (hydrated from the cluster), they are created or updated as needed. The sync continues until all top-level fields in the CUE instance are created. If `--watch`/`-w` is specified, syncing continues indefinitely.

![State Diagram](https://mermaid.ink/svg/eyJjb2RlIjoic3RhdGVEaWFncmFtLXYyXG4gICAgc3RhdGUgUHJvY2Vzc0NVRSB7XG4gICAgICAgIGdldFN0YXRlOiBDbHVzdGVyIFN0YXRlXG4gICAgICAgIHVuaWZ5OiBVbmlmeSB3aXRoIENVRVxuICAgICAgICBnZXRTdGF0ZSAtLT4gdW5pZnkgXG4gICAgICAgIHVuaWZ5IC0tPiBnZXRTdGF0ZVxuICAgIH1cbiAgICBzdGF0ZSBTeW5jIHtcbiAgICAgIGM6IGNvbmNyZXRlIHZhbHVlXG4gICAgICBjIC0tPiBDcmVhdGVcbiAgICAgIGMgLS0-IEFwcGx5XG4gICAgfVxuICAgIENyZWF0ZSAtLT4gV2F0Y2g6IHN0YXJ0IHdhdGNoaW5nXG4gICAgc3RhdGUgSW5mb3JtIHtcbiAgICAgIFdhdGNoXG4gICAgfVxuICAgIHVuaWZ5IC0tPiBjOiBlbWl0XG4gICAgV2F0Y2ggLS0-IGdldFN0YXRlXG5cbiIsIm1lcm1haWQiOnsidGhlbWUiOiJkZWZhdWx0IiwidGhlbWVWYXJpYWJsZXMiOnsiYmFja2dyb3VuZCI6IndoaXRlIiwicHJpbWFyeUNvbG9yIjoiI0VDRUNGRiIsInNlY29uZGFyeUNvbG9yIjoiI2ZmZmZkZSIsInRlcnRpYXJ5Q29sb3IiOiJoc2woODAsIDEwMCUsIDk2LjI3NDUwOTgwMzklKSIsInByaW1hcnlCb3JkZXJDb2xvciI6ImhzbCgyNDAsIDYwJSwgODYuMjc0NTA5ODAzOSUpIiwic2Vjb25kYXJ5Qm9yZGVyQ29sb3IiOiJoc2woNjAsIDYwJSwgODMuNTI5NDExNzY0NyUpIiwidGVydGlhcnlCb3JkZXJDb2xvciI6ImhzbCg4MCwgNjAlLCA4Ni4yNzQ1MDk4MDM5JSkiLCJwcmltYXJ5VGV4dENvbG9yIjoiIzEzMTMwMCIsInNlY29uZGFyeVRleHRDb2xvciI6IiMwMDAwMjEiLCJ0ZXJ0aWFyeVRleHRDb2xvciI6InJnYig5LjUwMDAwMDAwMDEsIDkuNTAwMDAwMDAwMSwgOS41MDAwMDAwMDAxKSIsImxpbmVDb2xvciI6IiMzMzMzMzMiLCJ0ZXh0Q29sb3IiOiIjMzMzIiwibWFpbkJrZyI6IiNFQ0VDRkYiLCJzZWNvbmRCa2ciOiIjZmZmZmRlIiwiYm9yZGVyMSI6IiM5MzcwREIiLCJib3JkZXIyIjoiI2FhYWEzMyIsImFycm93aGVhZENvbG9yIjoiIzMzMzMzMyIsImZvbnRGYW1pbHkiOiJcInRyZWJ1Y2hldCBtc1wiLCB2ZXJkYW5hLCBhcmlhbCIsImZvbnRTaXplIjoiMTZweCIsImxhYmVsQmFja2dyb3VuZCI6IiNlOGU4ZTgiLCJub2RlQmtnIjoiI0VDRUNGRiIsIm5vZGVCb3JkZXIiOiIjOTM3MERCIiwiY2x1c3RlckJrZyI6IiNmZmZmZGUiLCJjbHVzdGVyQm9yZGVyIjoiI2FhYWEzMyIsImRlZmF1bHRMaW5rQ29sb3IiOiIjMzMzMzMzIiwidGl0bGVDb2xvciI6IiMzMzMiLCJlZGdlTGFiZWxCYWNrZ3JvdW5kIjoiI2U4ZThlOCIsImFjdG9yQm9yZGVyIjoiaHNsKDI1OS42MjYxNjgyMjQzLCA1OS43NzY1MzYzMTI4JSwgODcuOTAxOTYwNzg0MyUpIiwiYWN0b3JCa2ciOiIjRUNFQ0ZGIiwiYWN0b3JUZXh0Q29sb3IiOiJibGFjayIsImFjdG9yTGluZUNvbG9yIjoiZ3JleSIsInNpZ25hbENvbG9yIjoiIzMzMyIsInNpZ25hbFRleHRDb2xvciI6IiMzMzMiLCJsYWJlbEJveEJrZ0NvbG9yIjoiI0VDRUNGRiIsImxhYmVsQm94Qm9yZGVyQ29sb3IiOiJoc2woMjU5LjYyNjE2ODIyNDMsIDU5Ljc3NjUzNjMxMjglLCA4Ny45MDE5NjA3ODQzJSkiLCJsYWJlbFRleHRDb2xvciI6ImJsYWNrIiwibG9vcFRleHRDb2xvciI6ImJsYWNrIiwibm90ZUJvcmRlckNvbG9yIjoiI2FhYWEzMyIsIm5vdGVCa2dDb2xvciI6IiNmZmY1YWQiLCJub3RlVGV4dENvbG9yIjoiYmxhY2siLCJhY3RpdmF0aW9uQm9yZGVyQ29sb3IiOiIjNjY2IiwiYWN0aXZhdGlvbkJrZ0NvbG9yIjoiI2Y0ZjRmNCIsInNlcXVlbmNlTnVtYmVyQ29sb3IiOiJ3aGl0ZSIsInNlY3Rpb25Ca2dDb2xvciI6InJnYmEoMTAyLCAxMDIsIDI1NSwgMC40OSkiLCJhbHRTZWN0aW9uQmtnQ29sb3IiOiJ3aGl0ZSIsInNlY3Rpb25Ca2dDb2xvcjIiOiIjZmZmNDAwIiwidGFza0JvcmRlckNvbG9yIjoiIzUzNGZiYyIsInRhc2tCa2dDb2xvciI6IiM4YTkwZGQiLCJ0YXNrVGV4dExpZ2h0Q29sb3IiOiJ3aGl0ZSIsInRhc2tUZXh0Q29sb3IiOiJ3aGl0ZSIsInRhc2tUZXh0RGFya0NvbG9yIjoiYmxhY2siLCJ0YXNrVGV4dE91dHNpZGVDb2xvciI6ImJsYWNrIiwidGFza1RleHRDbGlja2FibGVDb2xvciI6IiMwMDMxNjMiLCJhY3RpdmVUYXNrQm9yZGVyQ29sb3IiOiIjNTM0ZmJjIiwiYWN0aXZlVGFza0JrZ0NvbG9yIjoiI2JmYzdmZiIsImdyaWRDb2xvciI6ImxpZ2h0Z3JleSIsImRvbmVUYXNrQmtnQ29sb3IiOiJsaWdodGdyZXkiLCJkb25lVGFza0JvcmRlckNvbG9yIjoiZ3JleSIsImNyaXRCb3JkZXJDb2xvciI6IiNmZjg4ODgiLCJjcml0QmtnQ29sb3IiOiJyZWQiLCJ0b2RheUxpbmVDb2xvciI6InJlZCIsImxhYmVsQ29sb3IiOiJibGFjayIsImVycm9yQmtnQ29sb3IiOiIjNTUyMjIyIiwiZXJyb3JUZXh0Q29sb3IiOiIjNTUyMjIyIiwiY2xhc3NUZXh0IjoiIzEzMTMwMCIsImZpbGxUeXBlMCI6IiNFQ0VDRkYiLCJmaWxsVHlwZTEiOiIjZmZmZmRlIiwiZmlsbFR5cGUyIjoiaHNsKDMwNCwgMTAwJSwgOTYuMjc0NTA5ODAzOSUpIiwiZmlsbFR5cGUzIjoiaHNsKDEyNCwgMTAwJSwgOTMuNTI5NDExNzY0NyUpIiwiZmlsbFR5cGU0IjoiaHNsKDE3NiwgMTAwJSwgOTYuMjc0NTA5ODAzOSUpIiwiZmlsbFR5cGU1IjoiaHNsKC00LCAxMDAlLCA5My41Mjk0MTE3NjQ3JSkiLCJmaWxsVHlwZTYiOiJoc2woOCwgMTAwJSwgOTYuMjc0NTA5ODAzOSUpIiwiZmlsbFR5cGU3IjoiaHNsKDE4OCwgMTAwJSwgOTMuNTI5NDExNzY0NyUpIn19LCJ1cGRhdGVFZGl0b3IiOmZhbHNlfQ)

