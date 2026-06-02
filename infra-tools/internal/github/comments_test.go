package github

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	gh "github.com/google/go-github/v68/github"
)

type fakeCommentsService struct {
	comments []*gh.IssueComment
	created  []*gh.IssueComment
	edited   map[int64]*gh.IssueComment
	nextID   int64
}

func newFakeCommentsService() *fakeCommentsService {
	return &fakeCommentsService{
		edited: make(map[int64]*gh.IssueComment),
		nextID: 1,
	}
}

func (f *fakeCommentsService) ListComments(_ context.Context, _, _ string, _ int, _ *gh.IssueListCommentsOptions) ([]*gh.IssueComment, *gh.Response, error) {
	return f.comments, &gh.Response{}, nil
}

func (f *fakeCommentsService) CreateComment(_ context.Context, _, _ string, _ int, comment *gh.IssueComment) (*gh.IssueComment, *gh.Response, error) {
	comment.ID = gh.Ptr(f.nextID)
	f.nextID++
	f.created = append(f.created, comment)
	f.comments = append(f.comments, comment)
	return comment, &gh.Response{}, nil
}

func (f *fakeCommentsService) EditComment(_ context.Context, _, _ string, commentID int64, comment *gh.IssueComment) (*gh.IssueComment, *gh.Response, error) {
	f.edited[commentID] = comment
	for i, c := range f.comments {
		if c.GetID() == commentID {
			f.comments[i].Body = comment.Body
		}
	}
	return comment, &gh.Response{}, nil
}

func TestUpsertComment_CreatesNew(t *testing.T) {
	g := NewWithT(t)

	fake := newFakeCommentsService()
	client := &CommentClient{comments: fake, owner: "org", repo: "repo"}

	body := CommentMarker + "\n### Test\nHello"
	err := client.UpsertComment(context.Background(), 42, body)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(fake.created).To(HaveLen(1))
	g.Expect(*fake.created[0].Body).To(Equal(body))
}

func TestUpsertComment_UpdatesExisting(t *testing.T) {
	g := NewWithT(t)

	fake := newFakeCommentsService()
	// Pre-existing comment with the marker
	fake.comments = []*gh.IssueComment{
		{ID: gh.Ptr(int64(99)), Body: gh.Ptr(CommentMarker + "\nold content")},
	}

	client := &CommentClient{comments: fake, owner: "org", repo: "repo"}

	body := CommentMarker + "\n### Updated\nNew content"
	err := client.UpsertComment(context.Background(), 42, body)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(fake.created).To(BeEmpty()) // no new comment created
	g.Expect(fake.edited).To(HaveKey(int64(99)))
	g.Expect(*fake.edited[99].Body).To(Equal(body))
}

func TestUpsertComment_IgnoresUnrelatedComments(t *testing.T) {
	g := NewWithT(t)

	fake := newFakeCommentsService()
	fake.comments = []*gh.IssueComment{
		{ID: gh.Ptr(int64(1)), Body: gh.Ptr("unrelated comment")},
		{ID: gh.Ptr(int64(2)), Body: gh.Ptr("another comment")},
	}

	client := &CommentClient{comments: fake, owner: "org", repo: "repo"}

	body := CommentMarker + "\nnew"
	err := client.UpsertComment(context.Background(), 42, body)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(fake.created).To(HaveLen(1)) // created new, didn't update existing
}

// --- UpsertCommentByMarker ---

func TestUpsertCommentByMarker_CreatesWithCustomMarker(t *testing.T) {
	g := NewWithT(t)

	fake := newFakeCommentsService()
	client := &CommentClient{comments: fake, owner: "org", repo: "repo"}

	const myMarker = "<!-- my-tool-comment -->"
	body := myMarker + "\nhello from my tool"
	err := client.UpsertCommentByMarker(context.Background(), 1, body, myMarker)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(fake.created).To(HaveLen(1))
	g.Expect(*fake.created[0].Body).To(Equal(body))
}

func TestUpsertCommentByMarker_UpdatesExistingWithCustomMarker(t *testing.T) {
	g := NewWithT(t)

	fake := newFakeCommentsService()
	const myMarker = "<!-- my-tool-comment -->"
	fake.comments = []*gh.IssueComment{
		{ID: gh.Ptr(int64(7)), Body: gh.Ptr(myMarker + "\nold content")},
	}
	client := &CommentClient{comments: fake, owner: "org", repo: "repo"}

	updated := myMarker + "\nnew content"
	err := client.UpsertCommentByMarker(context.Background(), 1, updated, myMarker)
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(fake.created).To(BeEmpty())
	g.Expect(fake.edited).To(HaveKey(int64(7)))
	g.Expect(*fake.edited[7].Body).To(Equal(updated))
}

func TestUpsertCommentByMarker_DoesNotMatchDifferentMarker(t *testing.T) {
	g := NewWithT(t)

	// Two tools each post with their own marker — they must not interfere.
	fake := newFakeCommentsService()
	fake.comments = []*gh.IssueComment{
		{ID: gh.Ptr(int64(1)), Body: gh.Ptr("<!-- tool-a-comment -->\ntool A content")},
	}
	client := &CommentClient{comments: fake, owner: "org", repo: "repo"}

	// Tool B posts with its own marker — should create, not update tool A's comment.
	err := client.UpsertCommentByMarker(context.Background(), 1, "<!-- tool-b-comment -->\ntool B", "<!-- tool-b-comment -->")
	g.Expect(err).NotTo(HaveOccurred())
	g.Expect(fake.created).To(HaveLen(1))
	g.Expect(fake.edited).To(BeEmpty())
}

func TestUpsertCommentByMarker_RejectsEmptyMarker(t *testing.T) {
	g := NewWithT(t)

	fake := newFakeCommentsService()
	client := &CommentClient{comments: fake, owner: "org", repo: "repo"}

	err := client.UpsertCommentByMarker(context.Background(), 1, "body", "")
	g.Expect(err).To(HaveOccurred())
	g.Expect(err.Error()).To(ContainSubstring("marker must not be empty"))
}
