package services

import (
	"bytes"
	"context"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/storage"
	"github.com/google/uuid"
)

// memStore keeps files in memory, for the tests.
type memStore struct{ files map[string][]byte }

func (m *memStore) Put(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	b, err := io.ReadAll(r)
	m.files[key] = b
	return err
}

func (m *memStore) Open(_ context.Context, key string) (io.ReadCloser, error) {
	b, ok := m.files[key]
	if !ok {
		return nil, storage.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

func (m *memStore) Delete(_ context.Context, key string) error {
	delete(m.files, key)
	return nil
}

func (m *memStore) Kind() string { return "memory" }

// Group summary's pictures are kept on their first Download or Share, with the entry's and the
// group's titles; the other action marks the same picture; History lists them in the person's HQs.
func TestEntryImages(t *testing.T) {
	f := newWLFixture(t)
	f.db.Model(&models.User{}).Where("id = ?", f.actor).Update("default_branch_id", f.hq[0].ID)
	defer f.db.Model(&models.User{}).Where("id = ?", f.actor).Update("default_branch_id", nil)
	g, err := f.svc.CreateGroup(f.ctx, GroupInput{Title: "Image group", Agents: []GroupAgentInput{{PlatformID: f.p1.ID.String(), CommissionRate: "10"}}, StartsOn: dayString(f.week)}, all, f.actor)
	if err != nil {
		t.Fatal(err)
	}
	sheet, _, err := f.save(f.week, 0, canAll, "", row(f.p1, "1000", "200"))
	if err != nil {
		t.Fatal(err)
	}
	entry := sheet.Entry.ID
	png := []byte("\x89PNG\r\n\x1a\npicture")
	thumb := []byte("\xff\xd8\xffsmall")
	input := func(action string, group *uuid.UUID, withThumb bool) EntryImageInput {
		in := EntryImageInput{EntryID: entry, GroupID: group, Action: action, Image: EntryImageFile{ContentType: "image/png", Size: int64(len(png)), Body: bytes.NewReader(png)}}
		if withThumb {
			in.Thumb = &EntryImageFile{ContentType: "image/jpeg", Size: int64(len(thumb)), Body: bytes.NewReader(thumb)}
		}
		return in
	}
	read := func(rc io.ReadCloser) []byte {
		defer rc.Close()
		b, _ := io.ReadAll(rc)
		return b
	}

	// No storage: nothing is kept.
	if _, err := f.svc.SaveEntryImage(f.ctx, input(ImageDownload, &g.ID, true), all, f.actor); !errors.Is(err, ErrNoImageStorage) {
		t.Fatalf("without storage: %v", err)
	}
	store := &memStore{files: map[string][]byte{}}
	f.svc.SetStore(store)
	defer f.svc.SetStore(nil)

	if _, err := f.svc.SaveEntryImage(f.ctx, input("print", &g.ID, true), all, f.actor); err == nil {
		t.Fatal("an unknown action was accepted")
	}
	// Another HQ's person can't keep a picture of this entry, nor a group that isn't the entry's.
	if _, err := f.svc.SaveEntryImage(f.ctx, input(ImageDownload, &g.ID, true), f.scope(1), f.actor); !errors.Is(err, ErrEntryNotFound) {
		t.Fatalf("other HQ: %v", err)
	}
	stranger := uuid.New()
	if _, err := f.svc.SaveEntryImage(f.ctx, input(ImageDownload, &stranger, true), all, f.actor); !errors.Is(err, ErrGroupNotFound) {
		t.Fatalf("unknown group: %v", err)
	}
	if len(store.files) != 0 {
		t.Fatalf("refused pictures were stored: %v", store.files)
	}

	first, err := f.svc.SaveEntryImage(f.ctx, input(ImageDownload, &g.ID, true), all, f.actor)
	if err != nil {
		t.Fatal(err)
	}
	if first.EntryTitle != "TWL week" || first.GroupTitle != "Image group" || first.BranchCode != "TWL1" || first.CreatedBy != "TWL Tester" ||
		first.DownloadedAt == nil || first.SharedAt != nil || !first.HasThumb || first.WeekStart != dayString(f.week) || first.Size != int64(len(png)) {
		t.Fatalf("kept picture: %+v", first)
	}
	if len(store.files) != 2 {
		t.Fatalf("want the picture and its small copy stored, got %d files", len(store.files))
	}
	for key := range store.files {
		if !strings.HasPrefix(key, "winlose-images/"+f.hq[0].ID.String()+"/"+first.ID.String()) {
			t.Fatalf("stored under %q", key)
		}
	}

	// Sharing the same picture marks it (once); someone else, or another HQ, can't.
	shared, err := f.svc.MarkEntryImage(f.ctx, first.ID, ImageShare, all, f.actor)
	if err != nil {
		t.Fatal(err)
	}
	if shared.SharedAt == nil || shared.DownloadedAt == nil {
		t.Fatalf("after sharing: %+v", shared)
	}
	again, err := f.svc.MarkEntryImage(f.ctx, first.ID, ImageShare, all, f.actor)
	if err != nil || !again.SharedAt.Equal(*shared.SharedAt) {
		t.Fatalf("sharing again changed the time: %v %+v", err, again)
	}
	other := models.User{Username: "twl_image_other", Email: "twl_image_other@example.test", PasswordHash: "x", FullName: "TWL Other", IsActive: true}
	if err := f.db.Where(models.User{Username: other.Username}).FirstOrCreate(&other).Error; err != nil {
		t.Fatal(err)
	}
	if _, err := f.svc.MarkEntryImage(f.ctx, first.ID, ImageDownload, all, other.ID); !errors.Is(err, ErrEntryImageNotFound) {
		t.Fatalf("someone else marked it: %v", err)
	}
	if _, err := f.svc.MarkEntryImage(f.ctx, first.ID, ImageDownload, f.scope(1), f.actor); !errors.Is(err, ErrEntryImageNotFound) {
		t.Fatalf("marked from another HQ: %v", err)
	}

	// A picture without a group or a small copy (shared first).
	second, err := f.svc.SaveEntryImage(f.ctx, input(ImageShare, nil, false), all, f.actor)
	if err != nil {
		t.Fatal(err)
	}
	if second.GroupTitle != "" || second.GroupID != nil || second.HasThumb || second.SharedAt == nil || second.DownloadedAt != nil {
		t.Fatalf("second picture: %+v", second)
	}

	// Each picture has its group's reactions in the entry; one of no group the entry's own.
	if _, _, _, _, err := f.svc.ReactToGroup(f.ctx, entry, g.ID, models.ReactionSent, all, f.actor); err != nil {
		t.Fatal(err)
	}
	if _, _, _, err := f.svc.ReactToEntry(f.ctx, entry, "yes", all, f.actor); err != nil {
		t.Fatal(err)
	}
	kinds := func(list []EntryReaction) string {
		var out []string
		for _, r := range list {
			out = append(out, r.Reaction)
		}
		return strings.Join(out, " ")
	}
	if rows, _, err := f.svc.ListEntryImagesPage(f.ctx, EntryImageFilter{}, all); err != nil || len(rows) != 2 ||
		kinds(rows[1].Reactions) != models.ReactionSent || kinds(rows[0].Reactions) != "yes" {
		t.Fatalf("reactions: %v %+v", err, rows)
	}

	// History: newest first, by words in either title, only the person's HQs.
	list := func(filter EntryImageFilter, scope BranchScope) []EntryImageRow {
		t.Helper()
		rows, total, err := f.svc.ListEntryImagesPage(f.ctx, filter, scope)
		if err != nil {
			t.Fatal(err)
		}
		if int(total) != len(rows) {
			t.Fatalf("total %d for %d rows", total, len(rows))
		}
		return rows
	}
	if rows := list(EntryImageFilter{}, all); len(rows) != 2 || rows[0].ID != second.ID || rows[1].ID != first.ID {
		t.Fatalf("history: %+v", rows)
	}
	if rows := list(EntryImageFilter{Search: "image gr"}, all); len(rows) != 1 || rows[0].ID != first.ID {
		t.Fatalf("search by group: %+v", rows)
	}
	if rows := list(EntryImageFilter{Search: "TWL week"}, all); len(rows) != 2 {
		t.Fatalf("search by entry: %+v", rows)
	}
	if rows := list(EntryImageFilter{BranchID: &f.hq[1].ID}, all); len(rows) != 0 {
		t.Fatalf("another HQ's history: %+v", rows)
	}
	if rows := list(EntryImageFilter{}, f.scope(1)); len(rows) != 0 {
		t.Fatalf("seen from another HQ: %+v", rows)
	}
	if _, _, err := f.svc.ListEntryImagesPage(f.ctx, EntryImageFilter{BranchID: &f.hq[0].ID}, f.scope(1)); !errors.Is(err, ErrBranchAccessDenied) {
		t.Fatalf("asked for an HQ outside the scope: %v", err)
	}
	if rows, total, err := f.svc.ListEntryImagesPage(f.ctx, EntryImageFilter{Page: 2, PageSize: 1}, all); err != nil || total != 2 || len(rows) != 1 || rows[0].ID != first.ID {
		t.Fatalf("page 2: %v %d %+v", err, total, rows)
	}

	// The files: the small copy, or the picture when there is none; not from another HQ.
	rc, contentType, err := f.svc.OpenEntryImage(f.ctx, first.ID, true, all)
	if err != nil || contentType != "image/jpeg" || !bytes.Equal(read(rc), thumb) {
		t.Fatalf("small copy: %v %s", err, contentType)
	}
	rc, contentType, err = f.svc.OpenEntryImage(f.ctx, first.ID, false, all)
	if err != nil || contentType != "image/png" || !bytes.Equal(read(rc), png) {
		t.Fatalf("picture: %v %s", err, contentType)
	}
	rc, contentType, err = f.svc.OpenEntryImage(f.ctx, second.ID, true, all)
	if err != nil || contentType != "image/png" || !bytes.Equal(read(rc), png) {
		t.Fatalf("no small copy: %v %s", err, contentType)
	}
	if _, _, err := f.svc.OpenEntryImage(f.ctx, first.ID, false, f.scope(1)); !errors.Is(err, ErrEntryImageNotFound) {
		t.Fatalf("opened from another HQ: %v", err)
	}

	// Deleting the entry keeps its pictures in History, with its title.
	if _, _, err := f.svc.DeleteEntry(f.ctx, entry, "test cleanup", all); err != nil {
		t.Fatal(err)
	}
	rows := list(EntryImageFilter{}, all)
	if len(rows) != 2 || rows[0].EntryID != nil || rows[1].EntryTitle != "TWL week" || rows[1].GroupTitle != "Image group" ||
		len(rows[0].Reactions) != 0 || len(rows[1].Reactions) != 0 {
		t.Fatalf("after deleting the entry: %+v", rows)
	}
}
