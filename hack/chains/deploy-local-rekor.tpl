apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: rekor
spec:
  destination:
    # Define name or server, not both
    name: ''
    namespace: rekor-server
    server: 'https://kubernetes.default.svc'
  source:
    path: helm-charts/rekor
    repoURL: 'https://github.com/sigstore/sigstore-helm-operator'
    targetRevision: HEAD
    helm:
      valueFiles:
        - values.yaml
      values: |-
        server:
          ingress:
            hostname: rekor-server.${domain}
            annotations:
              route.openshift.io/termination: "edge"
  project: default
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true

