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
	"gorm.io/gorm/clause"
)

// MaxAttachmentBytes is the largest file accepted for a transaction.
const MaxAttachmentBytes = 10 << 20

// AllowedAttachmentTypes are checked against the file's sniffed content, not its name.
var AllowedAttachmentTypes = map[string]bool{
	"image/jpeg": true, "image/png": true, "image/gif": true, "image/webp": true, "application/pdf": true,
}

// MaxAttachments is how many files one transaction can have (e.g. a receipt and an invoice).
const MaxAttachments = 2

var (
	ErrNoAttachment       = errors.New("attachment not found")
	ErrAttachmentLocked   = errors.New("attachments can't be changed on voided or reversed transactions")
	ErrTooManyAttachments = fmt.Errorf("a transaction can have at most %d attachments; delete one first", MaxAttachments)
)

// AttachmentService stores a transaction's files (up to MaxAttachments) in object storage.
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

// Add stores one more file on the transaction. The caller has checked type and size.
func (s *AttachmentService) Add(ctx context.Context, id, actorID uuid.UUID, fileName, contentType string, size int64, r io.Reader) (*models.Transaction, error) {
	txn, err := s.txns.GetByID(ctx, id, actorID)
	if err != nil {
		return nil, err
	}
	if txn.Status == models.TxnStatusVoided || txn.Status == models.TxnStatusReversed {
		return nil, ErrAttachmentLocked
	}
	if len(txn.Attachments) >= MaxAttachments {
		return nil, ErrTooManyAttachments
	}
	key := fmt.Sprintf("transactions/%s/%s-%s", txn.ID, uuid.NewString()[:8], safeFileName(fileName))
	if err := s.store.Put(ctx, key, r, size, contentType); err != nil {
		return nil, fmt.Errorf("could not store the file: %w", err)
	}
	displayName := strings.TrimSpace(path.Base(strings.ReplaceAll(fileName, "\\", "/")))
	if len(displayName) > 255 {
		displayName = displayName[len(displayName)-255:]
	}
	err = s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Lock the transaction row so two uploads at once can't both take the last free slot.
		var locked models.Transaction
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Select("id").First(&locked, "id = ?", txn.ID).Error; err != nil {
			return err
		}
		var count int64
		if err := tx.Model(&models.TransactionAttachment{}).Where("transaction_id = ?", txn.ID).Count(&count).Error; err != nil {
			return err
		}
		if count >= MaxAttachments {
			return ErrTooManyAttachments
		}
		return tx.Create(&models.TransactionAttachment{
			TransactionID: txn.ID, StorageKey: key, Name: displayName,
			ContentType: contentType, Size: size, CreatedByID: actorID,
		}).Error
	})
	if err != nil {
		_ = s.store.Delete(ctx, key)
		return nil, err
	}
	return s.txns.GetByID(ctx, id, actorID)
}

// Open returns one of the transaction's attachments and a reader for its file.
func (s *AttachmentService) Open(ctx context.Context, id, attachmentID, actorID uuid.UUID) (*models.TransactionAttachment, io.ReadCloser, error) {
	txn, err := s.txns.GetByID(ctx, id, actorID)
	if err != nil {
		return nil, nil, err
	}
	att := findAttachment(txn, attachmentID)
	if att == nil {
		return nil, nil, ErrNoAttachment
	}
	rc, err := s.store.Open(ctx, att.StorageKey)
	if errors.Is(err, storage.ErrNotFound) {
		return nil, nil, ErrNoAttachment
	}
	return att, rc, err
}

// Remove deletes one of the transaction's attachments.
func (s *AttachmentService) Remove(ctx context.Context, id, attachmentID, actorID uuid.UUID) (*models.Transaction, error) {
	txn, err := s.txns.GetByID(ctx, id, actorID)
	if err != nil {
		return nil, err
	}
	if txn.Status == models.TxnStatusVoided || txn.Status == models.TxnStatusReversed {
		return nil, ErrAttachmentLocked
	}
	att := findAttachment(txn, attachmentID)
	if att == nil {
		return nil, ErrNoAttachment
	}
	if err := s.db.WithContext(ctx).Delete(&models.TransactionAttachment{}, "id = ?", att.ID).Error; err != nil {
		return nil, err
	}
	_ = s.store.Delete(ctx, att.StorageKey)
	return s.txns.GetByID(ctx, id, actorID)
}

func findAttachment(txn *models.Transaction, attachmentID uuid.UUID) *models.TransactionAttachment {
	for i := range txn.Attachments {
		if txn.Attachments[i].ID == attachmentID {
			return &txn.Attachments[i]
		}
	}
	return nil
}
