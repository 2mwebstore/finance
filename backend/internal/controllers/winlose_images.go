package controllers

import (
	"bytes"
	"errors"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"strings"

	"github.com/cambodia-finance/income-expense/internal/services"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// Group summary's pictures, kept for RP's History when they are downloaded or shared.

func writeImageError(c *gin.Context, err error) {
	switch {
	case errors.Is(err, services.ErrEntryImageNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrNoImageStorage):
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": err.Error()})
	default:
		writePlatformError(c, err)
	}
}

// imagePart opens an uploaded picture, with its content type read from its bytes (not its name or
// the browser's type). ok is false when it is too large or not a picture of the given types.
func imagePart(fh *multipart.FileHeader, limit int64, types map[string]string) (file *services.EntryImageFile, closer io.Closer, ok bool) {
	if fh.Size <= 0 || fh.Size > limit {
		return nil, nil, false
	}
	f, err := fh.Open()
	if err != nil {
		return nil, nil, false
	}
	head := make([]byte, 512)
	n, err := io.ReadFull(f, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		f.Close()
		return nil, nil, false
	}
	contentType, _, _ := mime.ParseMediaType(http.DetectContentType(head[:n]))
	if _, known := types[contentType]; !known {
		f.Close()
		return nil, nil, false
	}
	return &services.EntryImageFile{ContentType: contentType, Size: fh.Size, Body: io.MultiReader(bytes.NewReader(head[:n]), f)}, f, true
}

// SaveEntryImage keeps a picture of an entry's card in Group summary (multipart: file, an optional
// thumb, action download or share, and group_id when it is a group's), on its first Download or
// Share to Telegram.
func (pc *PlatformController) SaveEntryImage(c *gin.Context) {
	_, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, services.MaxEntryImageBytes+services.MaxEntryThumbBytes+(1<<20))
	fh, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "the image is missing or larger than 15 MB"})
		return
	}
	image, closer, ok := imagePart(fh, services.MaxEntryImageBytes, services.EntryImageTypes)
	if !ok {
		c.JSON(http.StatusUnsupportedMediaType, gin.H{"error": "the image must be a PNG or JPG of up to 15 MB"})
		return
	}
	defer closer.Close()
	in := services.EntryImageInput{EntryID: id, Action: strings.TrimSpace(c.PostForm("action")), Image: *image}
	// The small copy is optional: one that can't be read is left out.
	if th, err := c.FormFile("thumb"); err == nil {
		if thumb, thumbCloser, ok := imagePart(th, services.MaxEntryThumbBytes, services.EntryImageTypes); ok {
			defer thumbCloser.Close()
			in.Thumb = thumb
		}
	}
	if raw := strings.TrimSpace(c.PostForm("group_id")); raw != "" {
		groupID, err := uuid.Parse(raw)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid group id"})
			return
		}
		in.GroupID = &groupID
	}
	row, err := pc.svc.SaveEntryImage(c.Request.Context(), in, scope, actorOf(c))
	if err != nil {
		writeImageError(c, err)
		return
	}
	pc.record(c, actorOf(c), "winlose.image.save", "weekly_entry", id.String(), &row.BranchID, map[string]interface{}{
		"image_id": row.ID.String(), "title": row.EntryTitle, "week_start": row.WeekStart, "group": row.GroupTitle, "action": in.Action, "size": row.Size,
	})
	c.JSON(http.StatusCreated, row)
}

type imageActionInput struct {
	Action string `json:"action" binding:"required"`
}

// MarkEntryImage notes that a kept picture was also downloaded or shared (by the person who kept it).
func (pc *PlatformController) MarkEntryImage(c *gin.Context) {
	_, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	var in imageActionInput
	if !bindJSON(c, &in) {
		return
	}
	row, err := pc.svc.MarkEntryImage(c.Request.Context(), id, strings.TrimSpace(in.Action), scope, actorOf(c))
	if err != nil {
		writeImageError(c, err)
		return
	}
	c.JSON(http.StatusOK, row)
}

// ListEntryImages is RP's History: the kept pictures, newest first.
func (pc *PlatformController) ListEntryImages(c *gin.Context) {
	_, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	page, size, on := listPage(c)
	branchID, ok := queryID(c, "branch_id")
	if !ok {
		return
	}
	f := services.EntryImageFilter{BranchID: branchID, Search: c.Query("search"), CreatedFrom: c.Query("created_from"), CreatedTo: c.Query("created_to")}
	if on {
		f.Page, f.PageSize = page, size
	}
	rows, total, err := pc.svc.ListEntryImagesPage(c.Request.Context(), f, scope)
	if err != nil {
		writeImageError(c, err)
		return
	}
	pagedJSON(c, orEmpty(rows), total, page, size, on)
}

// EntryImageFile sends a kept picture (?thumb=1: its small copy).
func (pc *PlatformController) EntryImageFile(c *gin.Context) {
	_, scope, ok := pc.scope(c)
	if !ok {
		return
	}
	id, ok := paramID(c)
	if !ok {
		return
	}
	rc, contentType, err := pc.svc.OpenEntryImage(c.Request.Context(), id, c.Query("thumb") == "1", scope)
	if err != nil {
		writeImageError(c, err)
		return
	}
	defer rc.Close()
	c.DataFromReader(http.StatusOK, -1, contentType, rc, map[string]string{
		"X-Content-Type-Options": "nosniff",
		"Cache-Control":          "private, max-age=3600",
	})
}
