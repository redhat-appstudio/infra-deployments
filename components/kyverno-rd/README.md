Note: chainsaw is not deployed to any "real" clusters using ArgoCD.

Kyverno is a special component that does not share any base resources, thus it does not fully follow the universal component standards outlined in [this document](../../docs/ring-deployments/directory-layout.md). The only ring with a `base/` directory is Ring 0 and there are no manifest or image promotions available for this component.
