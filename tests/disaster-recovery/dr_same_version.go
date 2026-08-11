// dr_same_version.go implements the same-version DR (Disaster Recovery) test
// scenario, exercising a full backup/restore cycle on the current Konflux version.
//
// The test creates two tenants (SVTenant1 = KokoHazamar, SVTenant2 = MosheKipod),
// backs them up, simulates a disaster by deleting their namespaces, restores
// from backup using both SOP methods (Velero CLI and oc command), rotates
// ServiceAccount tokens, and verifies structural and functional integrity.
package disaster_recovery

import (
	"context"
	"sync"

	"github.com/konflux-ci/e2e-tests/pkg/framework"
	. "github.com/onsi/ginkgo/v2" //nolint:staticcheck
	. "github.com/onsi/gomega"    //nolint:staticcheck
)

func defineSameVersionSpecs() {
	Context("DR Same-Version Backup/Restore", Ordered, func() {
		defer GinkgoRecover()

		var fw *framework.Framework

		AfterEach(framework.ReportFailure(&fw))

		svTenants := []Tenant{SVTenant1, SVTenant2}

		BeforeAll(func() {
			var err error
			fw, err = framework.NewFramework("dr-sv")
			Expect(err).ShouldNot(HaveOccurred(), "failed to create framework")

			validateDREnvironment(fw)

			for i := range svTenants {
				forkRepoForTenant(fw, &svTenants[i])
			}
		})

		// Phase 1: Tenant creation and initial pipeline execution.
		// PaC config PRs must be merged before waiting for pipeline chains because
		// releases only trigger for push-event builds (merge commits on the default
		// branch), not pull-request-event builds.
		When("creating tenants and running initial pipelines", func() {
			It("should create both tenants concurrently", func() {
				var wg sync.WaitGroup
				for _, t := range svTenants {
					wg.Add(1)
					go func() {
						defer GinkgoRecover()
						defer wg.Done()
						createTenant(fw, t)
					}()
				}
				wg.Wait()
			})

			It("should merge PaC configuration PRs on forked repos", func() {
				for _, t := range svTenants {
					mergePaCConfigPRs(fw, t)
				}
			})

			It("should wait for all pipeline chains to succeed", func() {
				waitForPipelineChains(context.Background(), fw, svTenants, nil, nil)
			})
		})

		// Phase 2: Back up tenant data via Velero.
		When("backing up tenant data", func() {
			It("should create backup CRs for both tenants concurrently", func() {
				var wg sync.WaitGroup
				for _, t := range svTenants {
					wg.Add(1)
					go func() {
						defer GinkgoRecover()
						defer wg.Done()
						createBackup(fw, t)
					}()
				}
				wg.Wait()
			})
		})

		// Phase 3: Simulate disaster by deleting tenant namespaces.
		//
		// Real etcd loss destroys all resources instantly — no graceful
		// deletion, no finalizer processing. `oc delete project` triggers
		// graceful deletion instead, invoking every controller finalizer
		// (application-service, image-controller, integration-service,
		// release-service, pipelines-as-code). Any of these can stall
		// namespace deletion past the 10-minute timeout.
		//
		// Strip all finalizers before deletion to match real-disaster
		// semantics. See KFLUXINFRA-3954, STONEBLD-3714.
		When("simulating disaster by deleting namespaces", func() {
			It("should strip finalizers and delete namespaces atomically", func() {
				stripAndDeleteNamespaces(fw, svTenants)
			})
		})

		// Phase 4: Restore tenants from backup using both SOP methods.
		When("restoring from backup", func() {
			It("should restore tenant-1 (KokoHazamar) via velero CLI method", func() {
				restoreFromBackup(fw, SVTenant1, RestoreMethodVeleroCLI)
			})

			It("should restore tenant-2 (MosheKipod) via oc command method", func() {
				restoreFromBackup(fw, SVTenant2, RestoreMethodOCCommand)
			})
		})

		// Phase 5: Post-restore recovery — rotate stale SA tokens and
		// verify PaC Repositories survived the backup/restore cycle.
		When("performing post-restore recovery", func() {
			It("should rotate SA tokens on both tenants", func() {
				for _, t := range svTenants {
					rotateSATokens(context.Background(), fw, t.Namespace)
				}
			})

			It("should verify PaC Repositories exist on both tenants", func() {
				for _, t := range svTenants {
					verifyPaCRepositories(fw, t)
				}
			})
		})

		// Phase 6: Verify restored tenants are structurally and functionally intact.
		When("verifying restored tenants", func() {
			It("should confirm structural integrity of both tenants", func() {
				for _, t := range svTenants {
					verifyResources(fw, t)
				}
			})

			It("should confirm push secrets contain valid credentials before triggering builds", func() {
				waitForPushSecretReadiness(fw, svTenants)
			})

			It("should link pull secrets to pipeline SA for EC verify tasks", func() {
				ensurePullSecretsOnSA(fw, svTenants)
			})

			It("should confirm functional pipeline execution after restore", func() {
				triggerBuildsAndVerify(context.Background(), fw, svTenants)
			})
		})

		AfterAll(func() {
			cleanupForks(fw, svTenants)
			if CurrentSpecReport().Failed() {
				collectFailureArtifacts(fw, svTenants)
			} else {
				cleanupTestResources(fw, svTenants)
			}
			cleanupDanglingNamespaces(context.Background(), fw)
		})
	})
}
