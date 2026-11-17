package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/storage"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// Group summary's pictures (Image): the first Download or Share to Telegram of a picture keeps it
// on the server with the data entry's and the group's titles, for RP's History; doing the other
// with the same picture marks it as that too.

// MaxEntryImageBytes caps a kept picture (a big group at 2x is a few MB); MaxEntryThumbBytes its
// small copy for History's list.
const (
	MaxEntryImageBytes = 15 << 20
	MaxEntryThumbBytes = 1 << 20
)

// EntryImageTypes are the picture formats kept.
var EntryImageTypes = map[string]string{"image/png": "png", "image/jpeg": "jpg"}

// What was done with a picture.
const (
	ImageDownload = "download"
	ImageShare    = "share"
)

var (
	ErrEntryImageNotFound = errors.New("image not found")
	ErrNoImageStorage     = errors.New("file storage is not set up")
)

// SetStore sets where Group summary's pictures are kept (History).
func (s *PlatformService) SetStore(store storage.Store) { s.store = store }

// EntryImageFile is one uploaded file (the picture or its small copy). ContentType is from the
// bytes, not the name.
type EntryImageFile struct {
	ContentType string
	Size        int64
	Body        io.Reader
}

// EntryImageInput: a picture of the entry (in the group, when GroupID is set) and what was done
// with it. Thumb may be nil.
type EntryImageInput struct {
	EntryID uuid.UUID
	GroupID *uuid.UUID
	Action  string
	Image   EntryImageFile
	Thumb   *EntryImageFile
}

// EntryImageRow is a kept picture in History.
type EntryImageRow struct {
	ID           uuid.UUID  `json:"id"`
	EntryID      *uuid.UUID `json:"entry_id"`
	EntryTitle   string     `json:"entry_title"`
	WeekStart    string     `json:"week_start"`
	WeekEnd      string     `json:"week_end"`
	GroupID      *uuid.UUID `json:"group_id"`
	GroupTitle   string     `json:"group_title"`
	BranchID     uuid.UUID  `json:"branch_id"`
	BranchCode   string     `json:"branch_code"`
	BranchName   string     `json:"branch_name"`
	Size         int64      `json:"size"`
	HasThumb     bool       `json:"has_thumb"`
	DownloadedAt *time.Time `json:"downloaded_at"`
	SharedAt     *time.Time `json:"shared_at"`
	CreatedAt    time.Time  `json:"created_at"`
	CreatedBy    string     `json:"created_by"`
	// Reactions: the ✅ / ❌ / 📤 on the picture's group in its data entry (the entry's own when it
	// isn't of one group; none once the entry is deleted).
	Reactions []EntryReaction `json:"reactions"`
}

// EntryImageFilter narrows History: the HQ, words in the data entry's or the group's title, and the
// days it was kept on (YYYY-MM-DD, either may be empty).
type EntryImageFilter struct {
	BranchID               *uuid.UUID
	Search                 string
	CreatedFrom, CreatedTo string
	// PageSize 0: the newest 500; otherwise one page of that size.
	Page, PageSize int
}

func checkImageAction(action string) error {
	if action != ImageDownload && action != ImageShare {
		return fieldError("action", "must be download or share")
	}
	return nil
}

// imageActionColumn is the time column an action sets.
func imageActionColumn(action string) string {
	if action == ImageShare {
		return "shared_at"
	}
	return "downloaded_at"
}

// SaveEntryImage keeps a picture of an entry (one the person can see) and what was done with it.
func (s *PlatformService) SaveEntryImage(ctx context.Context, in EntryImageInput, scope BranchScope, actor uuid.UUID) (*EntryImageRow, error) {
	if s.store == nil {
		return nil, ErrNoImageStorage
	}
	if err := checkImageAction(in.Action); err != nil {
		return nil, err
	}
	ext, ok := EntryImageTypes[in.Image.ContentType]
	if !ok {
		return nil, fieldError("file", "use a PNG or JPG picture")
	}
	if in.Thumb != nil && in.Thumb.ContentType != "image/jpeg" && in.Thumb.ContentType != "image/png" {
		in.Thumb = nil // only a picture is kept as the small copy
	}
	db := s.db.WithContext(ctx)
	entry, err := findEntry(db, in.EntryID.String(), scope, false)
	if err != nil {
		return nil, err
	}
	var group models.AssignmentGroup
	if in.GroupID != nil {
		if err := entryGroup(db, entry, *in.GroupID, &group); err != nil {
			return nil, err
		}
	}
	id := uuid.New()
	img := models.WeeklyEntryImage{
		ID: id, EntryID: &entry.ID, BranchID: entry.BranchID, EntryTitle: entry.Title, WeekStart: entry.WeekStart,
		StorageKey:  fmt.Sprintf("winlose-images/%s/%s.%s", entry.BranchID, id, ext),
		ContentType: in.Image.ContentType, Size: in.Image.Size, CreatedByID: actor,
	}
	if in.GroupID != nil {
		img.GroupID, img.GroupTitle = &group.ID, group.Title
	}
	// The database keeps milliseconds: the times returned now are the ones History reads later.
	now := time.Now().Truncate(time.Millisecond)
	img.CreatedAt, img.UpdatedAt = now, now
	if in.Action == ImageShare {
		img.SharedAt = &now
	} else {
		img.DownloadedAt = &now
	}
	if err := s.store.Put(ctx, img.StorageKey, in.Image.Body, in.Image.Size, in.Image.ContentType); err != nil {
		return nil, fmt.Errorf("could not store the image: %w", err)
	}
	// The small copy is only for the list: without it the list shows the picture itself.
	if in.Thumb != nil {
		thumbExt := EntryImageTypes[in.Thumb.ContentType]
		key := fmt.Sprintf("winlose-images/%s/%s-thumb.%s", entry.BranchID, id, thumbExt)
		if err := s.store.Put(ctx, key, in.Thumb.Body, in.Thumb.Size, in.Thumb.ContentType); err == nil {
			img.ThumbKey = key
		}
	}
	if err := db.Create(&img).Error; err != nil {
		s.removeImageFiles(ctx, &img)
		return nil, err
	}
	rows, err := s.imageRows(ctx, []models.WeeklyEntryImage{img})
	if err != nil {
		return nil, err
	}
	return &rows[0], nil
}

func (s *PlatformService) removeImageFiles(ctx context.Context, img *models.WeeklyEntryImage) {
	if s.store == nil {
		return
	}
	for _, key := range []string{img.StorageKey, img.ThumbKey} {
		if key != "" {
			_ = s.store.Delete(ctx, key)
		}
	}
}

// findImage reads a kept picture in the person's HQs.
func (s *PlatformService) findImage(ctx context.Context, id uuid.UUID, scope BranchScope) (*models.WeeklyEntryImage, error) {
	var img models.WeeklyEntryImage
	if err := s.db.WithContext(ctx).First(&img, "id = ?", id).Error; err != nil || !scope.Has(img.BranchID) {
		return nil, ErrEntryImageNotFound
	}
	return &img, nil
}

// MarkEntryImage notes that the person who kept a picture also downloaded or shared it.
func (s *PlatformService) MarkEntryImage(ctx context.Context, id uuid.UUID, action string, scope BranchScope, actor uuid.UUID) (*EntryImageRow, error) {
	if err := checkImageAction(action); err != nil {
		return nil, err
	}
	img, err := s.findImage(ctx, id, scope)
	if err != nil {
		return nil, err
	}
	if img.CreatedByID != actor {
		return nil, ErrEntryImageNotFound // someone else's picture
	}
	column := imageActionColumn(action)
	if (action == ImageShare && img.SharedAt == nil) || (action == ImageDownload && img.DownloadedAt == nil) {
		now := time.Now().Truncate(time.Millisecond)
		if err := s.db.WithContext(ctx).Model(img).Where(column+" IS NULL").Update(column, now).Error; err != nil {
			return nil, err
		}
		if action == ImageShare {
			img.SharedAt = &now
		} else {
			img.DownloadedAt = &now
		}
	}
	rows, err := s.imageRows(ctx, []models.WeeklyEntryImage{*img})
	if err != nil {
		return nil, err
	}
	return &rows[0], nil
}

// ListEntryImagesPage lists the kept pictures, newest first (one page when f.PageSize is set), with
// how many match in all.
func (s *PlatformService) ListEntryImagesPage(ctx context.Context, f EntryImageFilter, scope BranchScope) ([]EntryImageRow, int64, error) {
	q := scope.where(s.db.WithContext(ctx).Model(&models.WeeklyEntryImage{}), "branch_id")
	if f.BranchID != nil {
		if !scope.Has(*f.BranchID) {
			return nil, 0, ErrBranchAccessDenied
		}
		q = q.Where("branch_id = ?", *f.BranchID)
	}
	if search := strings.TrimSpace(f.Search); search != "" {
		like := likePattern(search)
		q = q.Where("(entry_title LIKE ? OR group_title LIKE ?)", like, like)
	}
	// Days are the business's days (its time zone), from midnight to midnight.
	if strings.TrimSpace(f.CreatedFrom) != "" {
		day, err := s.parseDay("created_from", f.CreatedFrom)
		if err != nil {
			return nil, 0, err
		}
		q = q.Where("created_at >= ?", time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, s.loc))
	}
	if strings.TrimSpace(f.CreatedTo) != "" {
		day, err := s.parseDay("created_to", f.CreatedTo)
		if err != nil {
			return nil, 0, err
		}
		q = q.Where("created_at < ?", time.Date(day.Year(), day.Month(), day.Day()+1, 0, 0, 0, 0, s.loc))
	}
	var total int64
	if err := q.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	q = q.Order("created_at DESC, id").Limit(500)
	if f.PageSize > 0 {
		q = q.Offset((max(f.Page, 1) - 1) * f.PageSize).Limit(f.PageSize)
	}
	var list []models.WeeklyEntryImage
	if err := q.Find(&list).Error; err != nil {
		return nil, 0, err
	}
	rows, err := s.imageRows(ctx, list)
	return rows, total, err
}

func (s *PlatformService) imageRows(ctx context.Context, list []models.WeeklyEntryImage) ([]EntryImageRow, error) {
	var branchIDs, users []uuid.UUID
	for _, img := range list {
		branchIDs, users = append(branchIDs, img.BranchID), append(users, img.CreatedByID)
	}
	branches, err := s.branchesIn(ctx, branchIDs)
	if err != nil {
		return nil, err
	}
	names := s.userNames(ctx, users)
	reactions, err := s.imageReactions(ctx, list)
	if err != nil {
		return nil, err
	}
	out := make([]EntryImageRow, len(list))
	for i, img := range list {
		out[i] = EntryImageRow{
			ID: img.ID, EntryID: img.EntryID, EntryTitle: img.EntryTitle,
			WeekStart: dayString(img.WeekStart), WeekEnd: dayString(img.WeekStart.AddDate(0, 0, 6)),
			GroupID: img.GroupID, GroupTitle: img.GroupTitle,
			BranchID: img.BranchID, BranchCode: branches[img.BranchID].Code, BranchName: branches[img.BranchID].Name,
			Size: img.Size, HasThumb: img.ThumbKey != "", DownloadedAt: img.DownloadedAt, SharedAt: img.SharedAt,
			CreatedAt: img.CreatedAt, CreatedBy: names[img.CreatedByID], Reactions: orNone(reactions[img.ID]),
		}
	}
	return out, nil
}

// imageReactions reads each picture's reactions (see EntryImageRow.Reactions), by picture: one
// read per data entry with group pictures, one for the entries' own.
func (s *PlatformService) imageReactions(ctx context.Context, list []models.WeeklyEntryImage) (map[uuid.UUID][]EntryReaction, error) {
	out := map[uuid.UUID][]EntryReaction{}
	groupsOf := map[uuid.UUID][]uuid.UUID{} // entry → its pictures' groups
	var entries []uuid.UUID                 // entries with a picture of no group
	for _, img := range list {
		switch {
		case img.EntryID == nil:
		case img.GroupID != nil:
			groupsOf[*img.EntryID] = append(groupsOf[*img.EntryID], *img.GroupID)
		default:
			entries = append(entries, *img.EntryID)
		}
	}
	byGroup := map[uuid.UUID]map[uuid.UUID][]EntryReaction{}
	for entry, groups := range groupsOf {
		got, err := s.reactionsWhere(ctx, groupReactions, groups, map[string]interface{}{"entry_id": entry})
		if err != nil {
			return nil, err
		}
		byGroup[entry] = got
	}
	byEntry, err := s.reactionsOf(ctx, entryReactions, entries)
	if err != nil {
		return nil, err
	}
	for _, img := range list {
		switch {
		case img.EntryID == nil:
		case img.GroupID != nil:
			out[img.ID] = byGroup[*img.EntryID][*img.GroupID]
		default:
			out[img.ID] = byEntry[*img.EntryID]
		}
	}
	return out, nil
}

// OpenEntryImage opens a kept picture (thumb: its small copy, or the picture when it has none), with
// its content type.
func (s *PlatformService) OpenEntryImage(ctx context.Context, id uuid.UUID, thumb bool, scope BranchScope) (io.ReadCloser, string, error) {
	img, err := s.findImage(ctx, id, scope)
	if err != nil {
		return nil, "", err
	}
	if s.store == nil {
		return nil, "", ErrNoImageStorage
	}
	key, contentType := img.StorageKey, img.ContentType
	if thumb && img.ThumbKey != "" {
		key = img.ThumbKey
		contentType = "image/jpeg"
		if strings.HasSuffix(key, ".png") {
			contentType = "image/png"
		}
	}
	rc, err := s.store.Open(ctx, key)
	if err != nil {
		return nil, "", ErrEntryImageNotFound
	}
	return rc, contentType, nil
}
