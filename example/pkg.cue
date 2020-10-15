package test

import (
    corev1 "k8s.io/api/core/v1"
    rbacv1 "k8s.io/api/rbac/v1"
)

// imported schemas don't specify kind/apiversion
#Namespace: corev1.#Namespace & {
    apiVersion: "v1"
    kind: "Namespace"
}

TestNs: #Namespace & {
    apiVersion: "v1"
    kind: "Namespace"
    metadata: generateName: "test-ns-"
}

NoGenNameServiceAccount: corev1.#ServiceAccount & {
    apiVersion: "v1"
    kind: "ServiceAccount"
    metadata: {
        name: "test"
        namespace: "default"
    }
}

TestServiceAccount: corev1.#ServiceAccount & {
    apiVersion: "v1"
    kind: "ServiceAccount"
    metadata: {
        generateName: "test-sa-"
        namespace: TestNs.metadata.name
    }
}

TestClusterRole: rbacv1.#ClusterRole & {
    apiVersion: "rbac.authorization.k8s.io/v1"
    kind: "ClusterRole"
    metadata: generateName: "test-"
    rules: [
        {
            apiGroups: ["*"]
            resources: ["*"]
            verbs: ["*"]
        },
        {
            nonResourceURLs: ["*"]
            verbs: ["*"]
        }
    ]
}

TestClusterRoleBinding: rbacv1.#ClusterRoleBinding & {
    apiVersion: "rbac.authorization.k8s.io/v1"
    kind: "ClusterRoleBinding"
    metadata: generateName: "test-"
    roleRef: {
        apiGroup: "rbac.authorization.k8s.io"
        kind: "ClusterRole"
        name: TestClusterRole.metadata.name
    }
    subjects: [
        {
            kind: "ServiceAccount"
            name: TestServiceAccount.metadata.name
            namespace: TestNs.metadata.name
        }
    ]
}
