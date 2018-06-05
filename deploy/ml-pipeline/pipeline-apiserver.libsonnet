{
  all(namespace, api_image):: [
    $.parts(namespace).serviceAccount,
    $.parts(namespace).roleBinding,
    $.parts(namespace).role,
    $.parts(namespace).service,
    $.parts(namespace).deploy(api_image),
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
  },  // parts
}
