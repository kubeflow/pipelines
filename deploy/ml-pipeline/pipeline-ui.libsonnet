{
  all(namespace, ui_image):: [
    $.parts(namespace).serviceUi,
    $.parts(namespace).deployUi(ui_image),
  ],
  parts(namespace):: {
    serviceUi: {
      apiVersion: "v1",
      kind: "Service",
      metadata: {
        labels: {
          app: "ml-pipeline-ui",
        },
        name: "ml-pipeline-ui",
        namespace: namespace,
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
                imagePullPolicy: "IfNotPresent",
                ports: [
                    {
                      containerPort: 3000,
                    },
                  ],
              },
            ],
          },
        },
      },
    }, // deployUi
  },  // parts
}