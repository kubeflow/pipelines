{
  parts(namespace):: {
    all:: [
      $.parts(namespace).crd,
      $.parts(namespace).config,
      $.parts(namespace).deploy,
      $.parts(namespace).deployUi,
      $.parts(namespace).uiService,
      $.parts(namespace).serviceAccount,
      $.parts(namespace).role,
      $.parts(namespace).roleBinding,
      $.parts(namespace).uiServiceAccount,
      $.parts(namespace).uiRole,
      $.parts(namespace).uiRoleBinding,
    ],
    crd: {
      apiVersion: "apiextensions.k8s.io/v1beta1",
      kind: "CustomResourceDefinition",
      metadata: {
        name: "workflows.argoproj.io",
      },
      spec: {
        group: "argoproj.io",
        names: {
          kind: "Workflow",
          listKind: "WorkflowList",
          plural: "workflows",
          shortNames: [
            "wf",
          ],
          singular: "workflow",
        },
        scope: "Namespaced",
        version: "v1alpha1",
      },
    },  // crd

    // Deploy the controller
    deploy: {
      apiVersion: "extensions/v1beta1",
      kind: "Deployment",
      labels: {
        app: "workflow-controller",
      },
      metadata: {
        name: "workflow-controller",
        namespace: namespace,
      },
      spec: {
        progressDeadlineSeconds: 600,
        replicas: 1,
        revisionHistoryLimit: 10,
        selector: {
          matchLabels: {
            app: "workflow-controller",
          },
        },
        strategy: {
          rollingUpdate: {
            maxSurge: "25%",
            maxUnavailable: "25%",
          },
          type: "RollingUpdate",
        },
        template: {
          metadata: {
            creationTimestamp: null,
            labels: {
              app: "workflow-controller",
            },
          },
          spec: {
            containers: [
              {
                args: [
                  "--configmap",
                  "workflow-controller-configmap",
                ],
                command: [
                  "workflow-controller",
                ],
                env: [
                  {
                    name: "ARGO_NAMESPACE",
                    valueFrom: {
                      fieldRef: {
                        apiVersion: "v1",
                        fieldPath: "metadata.namespace",
                      },
                    },
                  },
                ],
                image: "argoproj/workflow-controller:v2.2.0",
                imagePullPolicy: "IfNotPresent",
                name: "workflow-controller",
                resources: {},
                terminationMessagePath: "/dev/termination-log",
                terminationMessagePolicy: "File",
              },
            ],
            dnsPolicy: "ClusterFirst",
            restartPolicy: "Always",
            schedulerName: "default-scheduler",
            securityContext: {},
            serviceAccount: "argo",
            serviceAccountName: "argo",
            terminationGracePeriodSeconds: 30,
          },
        },
      },
    },  // deploy


    deployUi: {
      apiVersion: "extensions/v1beta1",
      kind: "Deployment",
      metadata: {
        labels: {
          app: "argo-ui",
        },
        name: "argo-ui",
        namespace: namespace,
      },
      spec: {
        progressDeadlineSeconds: 600,
        replicas: 1,
        revisionHistoryLimit: 10,
        selector: {
          matchLabels: {
            app: "argo-ui",
          },
        },
        strategy: {
          rollingUpdate: {
            maxSurge: "25%",
            maxUnavailable: "25%",
          },
          type: "RollingUpdate",
        },
        template: {
          metadata: {
            creationTimestamp: null,
            labels: {
              app: "argo-ui",
            },
          },
          spec: {
            containers: [
              {
                env: [
                  {
                    name: "ARGO_NAMESPACE",
                    valueFrom: {
                      fieldRef: {
                        apiVersion: "v1",
                        fieldPath: "metadata.namespace",
                      },
                    },
                  },
                  {
                    name: "IN_CLUSTER",
                    value: "true",
                  },
                ],
                image: "argoproj/argoui:v2.2.0",
                imagePullPolicy: "IfNotPresent",
                name: "argo-ui",
                resources: {},
                terminationMessagePath: "/dev/termination-log",
                terminationMessagePolicy: "File",
              },
            ],
            dnsPolicy: "ClusterFirst",
            restartPolicy: "Always",
            schedulerName: "default-scheduler",
            securityContext: {},
            serviceAccount: "argo-ui",
            serviceAccountName: "argo-ui",
            terminationGracePeriodSeconds: 30,
            readinessProbe: {
              httpGet: {
                path: "/",
                port: 8001,
              },
            },
          },
        },
      },
    },  // deployUi

    uiService: {
      apiVersion: "v1",
      kind: "Service",
      metadata: {
        labels: {
          app: "argo-ui",
        },
        name: "argo-ui",
        namespace: namespace,
      },
      spec: {
        ports: [
          {
            port: 80,
            targetPort: 8001,
          },
        ],
        selector: {
          app: "argo-ui",
        },
        sessionAffinity: "None",
        type: "NodePort",
      },
    },

    config: {
      apiVersion: "v1",
      // The commented out section creates a default artifact repository
      // This section is not deleted because we might need it in the future.
      // And it takes time to get this string right.
      //data: {
      //  config: "executorImage: argoproj/argoexec:v2.2.0\nartifactRepository:\n s3:
      //  \n  bucket: mlpipeline\n  endpoint: minio-service.kubeflow:9000\n  insecure: true
      //  \n  accessKeySecret:\n   name: mlpipeline-minio-artifact\n   key: accesskey\n  secretKeySecret:
      //  \n   name: mlpipeline-minio-artifact\n   key: secretkey"
      //},
      data: {
        config: "executorImage: argoproj/argoexec:v2.2.0"
      },
      kind: "ConfigMap",
      metadata: {
        name: "workflow-controller-configmap",
        namespace: namespace,
      },
    },

    serviceAccount: {
      apiVersion: "v1",
      kind: "ServiceAccount",
      metadata: {
        name: "argo",
        namespace: namespace,
      },
    },  // service account

    // Keep in sync with https://github.com/argoproj/argo/blob/master/cmd/argo/commands/const.go#L20
    // Permissions need to be cluster wide for the workflow controller to be able to process workflows
    // in other namespaces. We could potentially use the ConfigMap of the workflow-controller to
    // scope it to a particular namespace in which case we might be able to restrict the permissions
    // to a particular namespace.
    role: {
      apiVersion: "rbac.authorization.k8s.io/v1beta1",
      kind: "ClusterRole",
      metadata: {
        labels: {
          app: "argo",
        },
        name: "argo",
        namespace: namespace,
      },
      rules: [
        {
          apiGroups: [""],
          resources: [
            "pods",
            "pods/exec",
          ],
          verbs: [
            "create",
            "get",
            "list",
            "watch",
            "update",
            "patch",
          ],
        },
        {
          apiGroups: [""],
          resources: [
            "secrets",
          ],
          verbs: [
            "get",
          ],
        },
        {
          apiGroups: [""],
          resources: [
            "configmaps",
          ],
          verbs: [
            "get",
            "watch",
            "list",
          ],
        },
        {
          apiGroups: [
            "",
          ],
          resources: [
            "persistentvolumeclaims",
          ],
          verbs: [
            "create",
            "delete",
          ],
        },
        {
          apiGroups: [
            "argoproj.io",
          ],
          resources: [
            "workflows",
          ],
          verbs: [
            "get",
            "list",
            "watch",
            "update",
            "patch",
          ],
        },
      ],
    },  // operator-role

    roleBinding:: {
      apiVersion: "rbac.authorization.k8s.io/v1beta1",
      kind: "ClusterRoleBinding",
      metadata: {
        labels: {
          app: "argo",
        },
        name: "argo",
        namespace: namespace,
      },
      roleRef: {
        apiGroup: "rbac.authorization.k8s.io",
        kind: "ClusterRole",
        name: "argo",
      },
      subjects: [
        {
          kind: "ServiceAccount",
          name: "argo",
          namespace: namespace,
        },
      ],
    },  // role binding

    uiServiceAccount: {
      apiVersion: "v1",
      kind: "ServiceAccount",
      metadata: {
        name: "argo-ui",
        namespace: namespace,
      },
    },  // service account

    // Keep in sync with https://github.com/argoproj/argo/blob/master/cmd/argo/commands/const.go#L44
    // Permissions need to be cluster wide for the workflow controller to be able to process workflows
    // in other namespaces. We could potentially use the ConfigMap of the workflow-controller to
    // scope it to a particular namespace in which case we might be able to restrict the permissions
    // to a particular namespace.
    uiRole: {
      apiVersion: "rbac.authorization.k8s.io/v1beta1",
      kind: "ClusterRole",
      metadata: {
        labels: {
          app: "argo",
        },
        name: "argo-ui",
        namespace: namespace,
      },
      rules: [
        {
          apiGroups: [""],
          resources: [
            "pods",
            "pods/exec",
            "pods/log",
          ],
          verbs: [
            "get",
            "list",
            "watch",
          ],
        },
        {
          apiGroups: [
            "argoproj.io",
          ],
          resources: [
            "workflows",
          ],
          verbs: [
            "get",
            "list",
            "watch",
          ],
        },
      ],
    },  // operator-role

    uiRoleBinding:: {
      apiVersion: "rbac.authorization.k8s.io/v1beta1",
      kind: "ClusterRoleBinding",
      metadata: {
        labels: {
          app: "argo-ui",
        },
        name: "argo-ui",
        namespace: namespace,
      },
      roleRef: {
        apiGroup: "rbac.authorization.k8s.io",
        kind: "ClusterRole",
        name: "argo-ui",
      },
      subjects: [
        {
          kind: "ServiceAccount",
          name: "argo-ui",
          namespace: namespace,
        },
      ],
    },  // role binding
  },  // parts
}
