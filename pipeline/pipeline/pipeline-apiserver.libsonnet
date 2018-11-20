{
  all(namespace, api_image):: [
    $.parts(namespace).serviceAccount,
    $.parts(namespace).roleBinding,
    $.parts(namespace).role,
    $.parts(namespace).service,
    $.parts(namespace).deploy(api_image),
    $.parts(namespace).pipelineRunnerServiceAccount,
    $.parts(namespace).pipelineRunnerRole,
    $.parts(namespace).pipelineRunnerRoleBinding,
  ],

  parts(namespace):: {
    serviceAccount: {
      apiVersion: "v1",
      kind: "ServiceAccount",
      metadata: {
        name: "ml-pipeline",
        namespace: namespace,
      },
    },  // service account

    roleBinding:: {
      apiVersion: "rbac.authorization.k8s.io/v1beta1",
      kind: "RoleBinding",
      metadata: {
        labels: {
          app: "ml-pipeline",
        },
        name: "ml-pipeline",
        namespace: namespace,
      },
      roleRef: {
        apiGroup: "rbac.authorization.k8s.io",
        kind: "Role",
        name: "ml-pipeline",
      },
      subjects: [
        {
          kind: "ServiceAccount",
          name: "ml-pipeline",
          namespace: namespace,
        },
      ],
    },  // role binding

    role: {
      apiVersion: "rbac.authorization.k8s.io/v1beta1",
      kind: "Role",
      metadata: {
        labels: {
          app: "ml-pipeline",
        },
        name: "ml-pipeline",
        namespace: namespace,
      },
      rules: [
        {
          apiGroups: [
            "argoproj.io",
          ],
          resources: [
            "workflows",
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
          apiGroups: [
            "kubeflow.org",
          ],
          resources: [
            "scheduledworkflows",
          ],
          verbs: [
            "create",
            "get",
            "list",
            "update",
            "patch",
            "delete",
          ],
        },
      ],
    },  // role

    service: {
      apiVersion: "v1",
      kind: "Service",
      metadata: {
        labels: {
          app: "ml-pipeline",
        },
        name: "ml-pipeline",
        namespace: namespace,
      },
      spec: {
        ports: [
          {
            port: 8888,
            targetPort: 8888,
            protocol: "TCP",
            name: "http",
          },
          {
            port: 8887,
            targetPort: 8887,
            protocol: "TCP",
            name: "grpc",
          },
        ],
        selector: {
          app: "ml-pipeline",
        },
      },
      status: {
        loadBalancer: {}
      },
    }, //service

    deploy(image): {
      apiVersion: "apps/v1beta2",
      kind: "Deployment",
      metadata: {
        "labels": {
          "app": "ml-pipeline",
        },
        name: "ml-pipeline",
        namespace: namespace,
      },
      spec: {
        selector: {
          matchLabels: {
            app: "ml-pipeline",
          },
        },
        template: {
          metadata: {
            labels: {
              app: "ml-pipeline",
            },
          },
          spec: {
            containers: [
              {
                name: "ml-pipeline-api-server",
                image: image,
                imagePullPolicy: 'Always',
                ports: [
                    {
                      containerPort: 8888,
                    },
                    {
                      containerPort: 8887,
                    },
                ],
                env: [
                  {
                    name: "POD_NAMESPACE",
                    valueFrom: {
                      fieldRef: {
                        fieldPath: "metadata.namespace",
                      },
                    },
                  },
                ],
              },
            ],
            serviceAccountName: "ml-pipeline",
          },
        },
      },
    }, // deploy

    pipelineRunnerServiceAccount: {
      apiVersion: "v1",
      kind: "ServiceAccount",
      metadata: {
        name: "pipeline-runner",
        namespace: namespace,
      },
    },  // service account

    // Keep in sync with https://github.com/argoproj/argo/blob/master/cmd/argo/commands/const.go#L20
    // Permissions need to be cluster wide for the workflow controller to be able to process workflows
    // in other namespaces. We could potentially use the ConfigMap of the workflow-controller to
    // scope it to a particular namespace in which case we might be able to restrict the permissions
    // to a particular namespace.
    pipelineRunnerRole: {
      apiVersion: "rbac.authorization.k8s.io/v1beta1",
      kind: "ClusterRole",
      metadata: {
        labels: {
          app: "pipeline-runner",
        },
        name: "pipeline-runner",
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

    pipelineRunnerRoleBinding:: {
      apiVersion: "rbac.authorization.k8s.io/v1beta1",
      kind: "ClusterRoleBinding",
      metadata: {
        labels: {
          app: "pipeline-runner",
        },
        name: "pipeline-runner",
        namespace: namespace,
      },
      roleRef: {
        apiGroup: "rbac.authorization.k8s.io",
        kind: "ClusterRole",
        name: "pipeline-runner",
      },
      subjects: [
        {
          kind: "ServiceAccount",
          name: "pipeline-runner",
          namespace: namespace,
        },
      ],
    },  // role binding
  },  // parts
}
