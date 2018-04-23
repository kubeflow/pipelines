{
  all(namespace, scheduler_image):: [
    $.parts(namespace).serviceAccount,
    $.parts(namespace).roleBinding,
    $.parts(namespace).role,
    $.parts(namespace).deploy(scheduler_image),
  ],

  parts(namespace):: {
    serviceAccount: {
      apiVersion: "v1",
      kind: "ServiceAccount",
      metadata: {
        name: "ml-pipeline-scheduler",
        namespace: namespace,
      },
    },  // service account

    roleBinding:: {
      apiVersion: "rbac.authorization.k8s.io/v1beta1",
      kind: "ClusterRoleBinding",
      metadata: {
        labels: {
          app: "ml-pipeline-scheduler",
        },
        name: "ml-pipeline-scheduler",
        namespace: namespace,
      },
      roleRef: {
        apiGroup: "rbac.authorization.k8s.io",
        kind: "ClusterRole",
        name: "ml-pipeline-scheduler",
      },
      subjects: [
        {
          kind: "ServiceAccount",
          name: "ml-pipeline-scheduler",
          namespace: namespace,
        },
      ],
    },  // role binding

    role: {
      apiVersion: "rbac.authorization.k8s.io/v1beta1",
      kind: "ClusterRole",
      metadata: {
        labels: {
          app: "ml-pipeline-scheduler",
        },
        name: "ml-pipeline-scheduler",
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

    deploy(image): {
      apiVersion: "apps/v1beta2",
      kind: "Deployment",
      metadata: {
        "labels": {
          "app": "ml-pipeline-scheduler",
        },
        name: "ml-pipeline-scheduler",
        namespace: namespace,
      },
      spec: {
        selector: {
          matchLabels: {
            app: "ml-pipeline-scheduler",
          },
        },
        template: {
          metadata: {
            labels: {
              app: "ml-pipeline-scheduler",
            },
          },
          spec: {
            containers: [
              {
                name: "ml-pipeline-scheduler",
                image: image,
                imagePullPolicy: 'IfNotPresent',
              },
            ],
            serviceAccountName: "ml-pipeline-scheduler",
          },
        },
      },
    }, // deploy
  },  // parts
}