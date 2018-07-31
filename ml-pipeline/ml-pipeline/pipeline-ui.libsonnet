{
  all(namespace, ui_image):: [
    $.parts(namespace).serviceAccount,
    $.parts(namespace).serviceUi,
    $.parts(namespace).roleBinding,
    $.parts(namespace).role,
    $.parts(namespace).deployUi(ui_image),
  ],
  parts(namespace):: {
    serviceAccount: {
      apiVersion: "v1",
      kind: "ServiceAccount",
      metadata: {
        name: "ml-pipeline-ui",
        namespace: namespace,
      },
    },  // service account

    serviceUi: {
      apiVersion: "v1",
      kind: "Service",
      metadata: {
        labels: {
          app: "ml-pipeline-ui",
        },
        name: "ml-pipeline-ui",
        namespace: namespace,
        annotations: {
          "getambassador.io/config":
            std.join("\n", [
              "---",
              "apiVersion: ambassador/v0",
              "kind:  Mapping",
              "name: pipelineui-mapping",
              "prefix: /pipeline",
              "rewrite: /pipeline",
              "service: ml-pipeline-ui." + namespace,
            ]),
        },  //annotations
      },
      spec: {
        ports: [
          {
            port: 80,
            targetPort: 3000,
          },
        ],
        selector: {
          app: "ml-pipeline-ui",
        },
      },
      status: {
        loadBalancer: {}
      },
    }, //serviceUi

    roleBinding:: {
      apiVersion: "rbac.authorization.k8s.io/v1beta1",
      kind: "RoleBinding",
      metadata: {
        labels: {
          app: "ml-pipeline-ui",
        },
        name: "ml-pipeline-ui",
        namespace: namespace,
      },
      roleRef: {
        apiGroup: "rbac.authorization.k8s.io",
        kind: "Role",
        name: "ml-pipeline-ui",
      },
      subjects: [
        {
          kind: "ServiceAccount",
          name: "ml-pipeline-ui",
          namespace: namespace,
        },
      ],
    },  // role binding

    role: {
      apiVersion: "rbac.authorization.k8s.io/v1beta1",
      kind: "Role",
      metadata: {
        labels: {
          app: "ml-pipeline-ui",
        },
        name: "ml-pipeline-ui",
        namespace: namespace,
      },
      rules: [
        {
          apiGroups: [
            "argoproj.io",
          ],
          resources: [
            "pods",
            "pods/log",
          ],
          verbs: [
            "create",
            "get",
          ],
        },
      ],
    },  // role

    deployUi(image): {
      apiVersion: "apps/v1beta2",
      kind: "Deployment",
      metadata: {
        "labels": {
          "app": "ml-pipeline-ui",
        },
        name: "ml-pipeline-ui",
        namespace: namespace,
      },
      spec: {
        selector: {
          matchLabels: {
            app: "ml-pipeline-ui",
          },
        },
        template: {
          metadata: {
            labels: {
              app: "ml-pipeline-ui",
            },
          },
          spec: {
            containers: [
              {
                name: "ml-pipeline-ui",
                image: image,
                imagePullPolicy: "Always",
                ports: [{
                  containerPort: 3000,
                }],
              },
            ],
            serviceAccountName: "ml-pipeline-ui",
          },
        },
      },
    }, // deployUi
  },  // parts
}
