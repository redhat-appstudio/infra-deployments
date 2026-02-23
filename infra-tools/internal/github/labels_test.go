package github

import (
	"context"
	"testing"

	gh "github.com/google/go-github/v68/github"
	. "github.com/onsi/gomega"
)

// fakeIssuesService implements IssuesService for testing.
type fakeIssuesService struct {
	// labels simulates the current labels on a PR.
	labels []*gh.Label
	// removed tracks which labels were removed.
	removed []string
	// added tracks which labels were added.
	added []string
}

func (f *fakeIssuesService) ListLabelsByIssue(_ context.Context, _, _ string, _ int, _ *gh.ListOptions) ([]*gh.Label, *gh.Response, error) {
	return f.labels, nil, nil
}

func (f *fakeIssuesService) RemoveLabelForIssue(_ context.Context, _, _ string, _ int, label string) (*gh.Response, error) {
	f.removed = append(f.removed, label)
	return nil, nil
}

func (f *fakeIssuesService) AddLabelsToIssue(_ context.Context, _, _ string, _ int, labels []string) ([]*gh.Label, *gh.Response, error) {
	f.added = append(f.added, labels...)
	return nil, nil, nil
}

func (f *fakeIssuesService) GetLabel(_ context.Context, _, _, _ string) (*gh.Label, *gh.Response, error) {
	return &gh.Label{}, nil, nil
}

func (f *fakeIssuesService) CreateLabel(_ context.Context, _, _ string, _ *gh.Label) (*gh.Label, *gh.Response, error) {
	return &gh.Label{}, nil, nil
}

func TestIsManagedLabel(t *testing.T) {
	tests := []struct {
		name  string
		label string
		want  bool
	}{
		{"environment label", "environment/production", true},
		{"environment staging", "environment/staging", true},
		{"environment development", "environment/development", true},
		{"cluster label", "cluster/kflux-ocp-p01", true},
		{"bug label", "bug", false},
		{"priority label", "priority/high", false},
		{"approved label", "approved", false},
		{"hold-production label", HoldProductionLabel, true},
		{"infra prefix", "infra/something", true},
		{"empty label", "", false},
		{"partial prefix", "environ", false},
		{"similar prefix", "environments/production", false},
		{"cluster-like", "clusters/foo", false},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			g := NewWithT(t)
			g.Expect(isManagedLabel(tc.label)).To(Equal(tc.want))
		})
	}
}

func TestSyncLabels_OnlyRemovesManagedLabels(t *testing.T) {
	g := NewWithT(t)

	fake := &fakeIssuesService{
		labels: []*gh.Label{
			{Name: gh.Ptr("environment/production")},
			{Name: gh.Ptr("environment/staging")},
			{Name: gh.Ptr("cluster/kflux-ocp-p01")},
			{Name: gh.Ptr("bug")},
			{Name: gh.Ptr("approved")},
			{Name: gh.Ptr("priority/critical")},
		},
	}

	client := &Client{issues: fake, owner: "o", repo: "r"}

	// Desired: only environment/development — stale managed labels should be
	// removed, but bug/approved/priority must stay.
	err := client.SyncLabels(context.Background(), 1, []string{"environment/development"})
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(fake.removed).To(ConsistOf(
		"environment/production",
		"environment/staging",
		"cluster/kflux-ocp-p01",
	))

	// Non-managed labels must never appear in the removed list.
	for _, label := range fake.removed {
		g.Expect(isManagedLabel(label)).To(BeTrue(), "non-managed label %q was removed", label)
	}
}

func TestSyncLabels_NoRemovalWhenAllDesired(t *testing.T) {
	g := NewWithT(t)

	fake := &fakeIssuesService{
		labels: []*gh.Label{
			{Name: gh.Ptr("environment/production")},
			{Name: gh.Ptr("bug")},
		},
	}

	client := &Client{issues: fake, owner: "o", repo: "r"}

	err := client.SyncLabels(context.Background(), 1, []string{"environment/production"})
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(fake.removed).To(BeEmpty())
	g.Expect(fake.added).To(BeEmpty())
}

func TestSyncLabels_HoldLabelAddedAndRemoved(t *testing.T) {
	t.Run("adds hold label when production is desired", func(t *testing.T) {
		g := NewWithT(t)

		fake := &fakeIssuesService{
			labels: []*gh.Label{
				{Name: gh.Ptr("environment/development")},
			},
		}

		client := &Client{issues: fake, owner: "o", repo: "r"}
		err := client.SyncLabels(context.Background(), 1, []string{
			"environment/production",
			HoldProductionLabel,
		})
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(fake.removed).To(ConsistOf("environment/development"))
		g.Expect(fake.added).To(ConsistOf("environment/production", HoldProductionLabel))
	})

	t.Run("removes hold label when production is no longer desired", func(t *testing.T) {
		g := NewWithT(t)

		fake := &fakeIssuesService{
			labels: []*gh.Label{
				{Name: gh.Ptr("environment/production")},
				{Name: gh.Ptr(HoldProductionLabel)},
				{Name: gh.Ptr("approved")}, // non-managed — must survive
			},
		}

		client := &Client{issues: fake, owner: "o", repo: "r"}
		err := client.SyncLabels(context.Background(), 1, []string{
			"environment/development",
		})
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(fake.removed).To(ConsistOf("environment/production", HoldProductionLabel))
		g.Expect(fake.added).To(ConsistOf("environment/development"))
	})
}

func TestSyncLabels_AddsNewLabels(t *testing.T) {
	g := NewWithT(t)

	fake := &fakeIssuesService{
		labels: []*gh.Label{
			{Name: gh.Ptr("bug")},
		},
	}

	client := &Client{issues: fake, owner: "o", repo: "r"}

	err := client.SyncLabels(context.Background(), 1, []string{"environment/staging"})
	g.Expect(err).NotTo(HaveOccurred())

	g.Expect(fake.removed).To(BeEmpty())
	g.Expect(fake.added).To(ConsistOf("environment/staging"))
}
