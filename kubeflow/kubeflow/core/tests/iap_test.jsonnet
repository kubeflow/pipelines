local iap = import "../iap.libsonnet";

std.assertEqual(iap.new(
  { namespace: "namespace" },
  { envoyPort: 8080 }
).service, {

  apiVersion: "v1",
  kind: "Service",
  metadata: {
    labels: {
      service: "envoy",
    },
    annotations: {
      "beta.cloud.google.com/backend-config": '{"ports": {"envoy":"envoy-iap"}}',
    },
    name: "envoy",
    namespace: "namespace",
  },
  spec: {
    ports: [
      {
        name: "envoy",
        port: 8080,
        targetPort: 8080,
      },
    ],
    selector: {
      service: "envoy",
    },
    type: "NodePort",
  },
}) &&

std.assertEqual(iap.new(
  { namespace: "namespace" },
  {
    envoyPort: 8080,
    ipName: "ipName",
    hostname: "hostname",
  }
).ingress, {
  apiVersion: "extensions/v1beta1",
  kind: "Ingress",
  metadata: {
    name: "envoy-ingress",
    namespace: "namespace",
    annotations: {
      "kubernetes.io/tls-acme": "true",
      "ingress.kubernetes.io/ssl-redirect": "true",
      "kubernetes.io/ingress.global-static-ip-name": "ipName",
    },
  },
  spec: {
    rules: [
      {
        host: "hostname",
        http: {
          paths: [
            {
              backend: {
                serviceName: "envoy",
                servicePort: 8080,
              },
              path: "/*",
            },
          ],
        },
      },
    ],
  },
}) &&

std.assertEqual(iap.new(
  {
    namespace: "namespace",
  },
  {
    envoyPort: 8080,
    ipName: "ipName",
    hostname: "null",
  }
).ingress, {
  apiVersion: "extensions/v1beta1",
  kind: "Ingress",
  metadata: {
    name: "envoy-ingress",
    namespace: "namespace",
    annotations: {
      "kubernetes.io/tls-acme": "true",
      "ingress.kubernetes.io/ssl-redirect": "true",
      "kubernetes.io/ingress.global-static-ip-name": "ipName",
    },
  },
  spec: {
    rules: [
      {
        http: {
          paths: [
            {
              backend: {
                serviceName: "envoy",
                servicePort: 8080,
              },
              path: "/*",
            },
          ],
        },
      },
    ],
  },
}) &&

std.assertEqual(iap.new(
  {
    namespace: "namespace",
  },
  {
    secretName: "secretName",
    hostname: "hostname",
    issuer: "issuer",
    privateGKECluster: "false",
  }
).certificate, {
  apiVersion: "certmanager.k8s.io/v1alpha1",
  kind: "Certificate",
  metadata: {
    name: "secretName",
    namespace: "namespace",
  },
  spec: {
    secretName: "secretName",
    issuerRef: {
      name: "issuer",
      kind: "Issuer",
    },
    commonName: "hostname",
    dnsNames: [
      "hostname",
    ],
    acme: {
      config: [
        {
          http01: {
            ingress: "envoy-ingress",
          },
          domains: [
            "hostname",
          ],
        },
      ],
    },
  },
}) &&

std.assertEqual(iap.new(
  {
    namespace: "namespace",
  },
  {
  }
).whoamiApp, {
  apiVersion: "extensions/v1beta1",
  kind: "Deployment",
  metadata: {
    name: "whoami-app",
    namespace: "namespace",
  },
  spec: {
    replicas: 1,
    template: {
      metadata: {
        labels: {
          app: "whoami",
        },
      },
      spec: {
        containers: [
          {
            env: [
              {
                name: "PORT",
                value: "8081",
              },
            ],
            image: "gcr.io/cloud-solutions-group/esp-sample-app:1.0.0",
            name: "app",
            ports: [
              {
                containerPort: 8081,
              },
            ],
            readinessProbe: {
              failureThreshold: 2,
              httpGet: {
                path: "/healthz",
                port: 8081,
                scheme: "HTTP",
              },
              periodSeconds: 10,
              successThreshold: 1,
              timeoutSeconds: 5,
            },
          },
        ],
      },
    },
  },
}) &&

std.assertEqual(iap.new(
  {
    namespace: "namespace",
  },
  {
  }
).whoamiService, {
  apiVersion: "v1",
  kind: "Service",
  metadata: {
    labels: {
      app: "whoami",
    },
    name: "whoami-app",
    namespace: "namespace",
  },
  spec: {
    ports: [
      {
        port: 80,
        targetPort: 8081,
      },
    ],
    selector: {
      app: "whoami",
    },
    type: "ClusterIP",
  },
})
