{
  parts(namespace):: {
    all:: [
      $.parts(namespace).pvc,
      $.parts(namespace).service,
      $.parts(namespace).deploy,
    ],

    pvc: {
      apiVersion: "v1",
      kind: "PersistentVolumeClaim",
      metadata: {
        name: "minio-pv-claim",
        namespace: namespace,
      },
      spec: {
        accessModes: [
          "ReadWriteOnce",
        ],
        resources: {
          requests: {
            storage: "10Gi",
          },
        },
      },
    }, //pvc

    service: {
      apiVersion: "v1",
      kind: "Service",
      metadata: {
        name: "minio-service",
        namespace: namespace,
      },
      spec: {
        ports: [
          {
            port: 9000,
            targetPort: 9000,
            protocol: "TCP",
          },
        ],
        selector: {
          app: "minio",
        },
      },
      status: {
        loadBalancer: {}
      },
    }, //service

    deploy: {
      apiVersion: "apps/v1beta1",
      kind: "Deployment",
      metadata: {
        name: "minio",
        namespace: namespace,
      },
      spec: {
        strategy: {
          type: "Recreate"
        },
        template: {
          metadata: {
            labels: {
              app: "minio",
            },
          },
          spec: {
            volumes: [
              {
                name: "data",
                persistentVolumeClaim: {
                  claimName: "minio-pv-claim",
                },
              },
            ],
            containers: [
              {
                name: "minio",
                volumeMounts: [
                  {
                    name: "data",
                    mountPath: "/data"
                  },
                ],
                image: "minio/minio:RELEASE.2018-02-09T22-40-05Z",
                args: [
                  "server",
                  "/data",
                ],
                env: [
                  {
                    name: "MINIO_ACCESS_KEY",
                    value: "minio",
                  },
                  {
                    name: "MINIO_SECRET_KEY",
                    value: "minio123",
                  },
                ],
                ports: [
                  {
                    containerPort: 9000,
                  },
                ],
              },
            ],
          },
        },
      },
    }, // deploy
  },  // parts
}