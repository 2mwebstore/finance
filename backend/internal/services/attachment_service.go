package services

import (
	"context"
	"errors"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"

	"github.com/cambodia-finance/income-expense/internal/models"
	"github.com/cambodia-finance/income-expense/internal/storage"
	"github.com/google/uuid"
	"gorm.io/gorm"
)

// MaxAttachmentBytes is the largest file accepted for a transaction.
const MaxAttachmentBytes = 10 << 20

// AllowedAttachmentTypes are checked against the file's sniffed content, not its name.
var AllowedAttachmentTypes = map[string]bool{
	"image/jpeg": true, "image/png": true, "image/gif": true, "image/webp": true, "application/pdf": true,
}

var (
	ErrNoAttachment     = errors.New("this transaction has no attachment")
	ErrAttachmentLocked = errors.New("attachments can't be changed on voided or reversed transactions")
)

// AttachmentService stores one file per transaction in object storage.
type AttachmentService struct {
	db    *gorm.DB
	txns  *TransactionService
	store storage.Store
}

func NewAttachmentService(db *gorm.DB, txns *TransactionService, store storage.Store) *AttachmentService {
	return &AttachmentService{db: db, txns: txns, store: store}
}

var unsafeName = regexp.MustCompile(`[^A-Za-z0-9._-]+`)

func safeFileName(name string) string {
	name = path.Base(strings.ReplaceAll(name, "\\", "/"))
	name = strings.Trim(unsafeName.ReplaceAllString(name, "-"), "-.")
	if name == "" {
		name = "file"
	}
	if len(name) > 80 {
		name = name[len(name)-80:]
	}
	return name
}

// Attach stores (or replaces) the transaction's file. The caller has checked type and size.
func (s *AttachmentService) Attach(ctx context.Context, id, actorID uuid.UUID, fileName, contentType string, size int64, r io.Reader) (*models.Transaction, error) {
	txn, err := s.txns.GetByID(ctx, id, actorID)
	if err != nil {
		return nil, err
	}
	if txn.Status == models.TxnStatusVoided || txn.Status == models.TxnStatusReversed {
		return nil, ErrAttachmentLocked
	}
	key := fmt.Sprintf("transactions/%s/%s-%s", txn.ID, uuid.NewString()[:8], safeFileName(fileName))
	if err := s.store.Put(ctx, key, r, size, contentType); err != nil {
		return nil, fmt.Errorf("could not store the file: %w", err)
	}
	displayName := strings.TrimSpace(path.Base(strings.ReplaceAll(fileName, "\\", "/")))
	if len(displayName) > 255 {
		displayName = displayName[len(displayName)-255:]
	}
	oldKey := txn.AttachmentKey
	if err := s.db.WithContext(ctx).Model(txn).Updates(map[string]interface{}{
		"attachment_key": key, "attachment_name": displayName, "attachment_type": contentType, "attachment_size": size,
	}).Error; err != nil {
		_ = s.store.Delete(ctx, key)
		return nil, err
	}
	if oldKey != "" && oldKey != key {
		_ = s.store.Delete(ctx, oldKey) // the old file is no longer referenced
	}
	return s.txns.GetByID(ctx, id, actorID)
}

// Open returns the transaction (for name/type) and a reader for its file.
func (s *AttachmentService) Open(ctx context.Context, id, actorID uuid.UUID) (*models.Transaction, io.ReadCloser, error) {
	txn, err := s.txns.GetByID(ctx, id, actorID)
	if err != nil {
		return nil, nil, err
	}
	if txn.AttachmentKey == "" {
		return nil, nil, ErrNoAttachment
	}
	rc, err := s.store.Open(ctx, txn.AttachmentKey)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, nil, ErrNoAttachment
	}
	return txn, rc, err
}

// Remove deletes the transaction's file.
func (s *AttachmentService) Remove(ctx context.Context, id, actorID uuid.UUID) (*models.Transaction, error) {
	txn, err := s.txns.GetByID(ctx, id, actorID)
	if err != nil {
		return nil, err
	}
	if txn.AttachmentKey == "" {
		return txn, nil
	}
	if txn.Status == models.TxnStatusVoided || txn.Status == models.TxnStatusReversed {
		return nil, ErrAttachmentLocked
	}
	key := txn.AttachmentKey
	if err := s.db.WithContext(ctx).Model(txn).Updates(map[string]interface{}{
		"attachment_key": "", "attachment_name": "", "attachment_type": "", "attachment_size": 0,
	}).Error; err != nil {
		return nil, err
	}
	_ = s.store.Delete(ctx, key)
	return s.txns.GetByID(ctx, id, actorID)
}
