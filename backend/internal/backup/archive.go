package backup

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/cambodia-finance/income-expense/internal/models"
)

// A full backup is a .tar.gz archive with
//
//	database.sql               the database (see Dump)
//	attachments/<storage key>  every transaction attachment (receipts, invoices…)
//	manifest.json              what the archive holds
//
// Backups made before attachments were included, and uploaded .sql / .sql.gz files, hold the
// database only; they restore the database and leave the files as they are.
const (
	archiveDBName   = "database.sql"
	archiveFilesDir = "attachments/"
	archiveManifest = "manifest.json"
	// Attachments are at most 10 MB each; anything far larger in storage is not one of ours.
	maxArchivedFile = 25 << 20
)

// storageKeyRule is the shape of attachment keys (see services.AttachmentService.Add), so a
// restored file can only ever land in the attachments area of the storage.
var storageKeyRule = regexp.MustCompile(`^transactions/[0-9a-fA-F-]{36}/[A-Za-z0-9._-]+$`)

type manifest struct {
	Version     int       `json:"version"`
	App         string    `json:"app"`
	CreatedAt   time.Time `json:"created_at"`
	Attachments int       `json:"attachments"`
	Missing     int       `json:"missing_attachments"`
}

// writeArchive writes a full backup (database and attachment files) to w.
func (s *Service) writeArchive(ctx context.Context, w io.Writer) (files, missing int, err error) {
	sqlDB, err := s.db.DB()
	if err != nil {
		return 0, 0, err
	}
	// tar needs each entry's size before its content, so the dump goes to a temp file first.
	dump, err := os.CreateTemp(s.dir, ".dump-*")
	if err != nil {
		return 0, 0, err
	}
	defer os.Remove(dump.Name())
	defer dump.Close()
	if err := Dump(ctx, sqlDB, dump); err != nil {
		return 0, 0, err
	}
	size, err := dump.Seek(0, io.SeekCurrent)
	if err != nil {
		return 0, 0, err
	}
	if _, err := dump.Seek(0, io.SeekStart); err != nil {
		return 0, 0, err
	}

	var keys []string
	if s.store != nil {
		if err := s.db.WithContext(ctx).Model(&models.TransactionAttachment{}).Distinct("storage_key").Pluck("storage_key", &keys).Error; err != nil {
			return 0, 0, err
		}
	}

	now := time.Now()
	gz := gzip.NewWriter(w)
	tw := tar.NewWriter(gz)
	if err := tw.WriteHeader(&tar.Header{Name: archiveDBName, Mode: 0o640, Size: size, ModTime: now}); err != nil {
		return 0, 0, err
	}
	if _, err := io.Copy(tw, dump); err != nil {
		return 0, 0, err
	}
	for _, key := range keys {
		data, ok := s.readStored(ctx, key)
		if !ok {
			missing++
			continue
		}
		if err := tw.WriteHeader(&tar.Header{Name: archiveFilesDir + key, Mode: 0o640, Size: int64(len(data)), ModTime: now}); err != nil {
			return files, missing, err
		}
		if _, err := tw.Write(data); err != nil {
			return files, missing, err
		}
		files++
	}
	meta, _ := json.MarshalIndent(manifest{Version: 1, App: s.appName, CreatedAt: now, Attachments: files, Missing: missing}, "", "  ")
	if err := tw.WriteHeader(&tar.Header{Name: archiveManifest, Mode: 0o640, Size: int64(len(meta)), ModTime: now}); err != nil {
		return files, missing, err
	}
	if _, err := tw.Write(meta); err != nil {
		return files, missing, err
	}
	if err := tw.Close(); err != nil {
		return files, missing, err
	}
	return files, missing, gz.Close()
}

// readStored reads one attachment from storage; a file that is gone or oversized is skipped.
func (s *Service) readStored(ctx context.Context, key string) ([]byte, bool) {
	if !storageKeyRule.MatchString(key) {
		return nil, false
	}
	rc, err := s.store.Open(ctx, key)
	if err != nil {
		log.Printf("backup: attachment %s not included: %v", key, err)
		return nil, false
	}
	defer rc.Close()
	data, err := io.ReadAll(io.LimitReader(rc, maxArchivedFile+1))
	if err != nil || len(data) > maxArchivedFile {
		log.Printf("backup: attachment %s not included (unreadable or too large)", key)
		return nil, false
	}
	return data, true
}

// loaded is what a backup file holds.
type loaded struct {
	script  string
	archive bool // a full backup (database and attachments)
	files   int  // attachment files in it
}

// loadBackup reads a backup file: a full .tar.gz archive, or a .sql / .sql.gz with the database
// only. The SQL is checked (see Validate), and so is every attachment path in an archive.
func loadBackup(r io.Reader) (loaded, error) {
	buf := bufio.NewReader(r)
	head, _ := buf.Peek(2)
	if len(head) < 2 || head[0] != 0x1f || head[1] != 0x8b {
		script, err := readLimited(buf)
		if err != nil {
			return loaded{}, err
		}
		return loaded{script: script}, Validate(script)
	}
	gz, err := gzip.NewReader(buf)
	if err != nil {
		return loaded{}, fmt.Errorf("the file is not a valid .gz archive")
	}
	defer gz.Close()
	inner := bufio.NewReader(gz)
	if magic, _ := inner.Peek(262); len(magic) < 262 || string(magic[257:262]) != "ustar" {
		script, err := readLimited(inner)
		if err != nil {
			return loaded{}, err
		}
		return loaded{script: script}, Validate(script)
	}
	out := loaded{archive: true}
	tr := tar.NewReader(inner)
	foundDB := false
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return loaded{}, fmt.Errorf("the backup archive is damaged")
		}
		switch {
		case hdr.Name == archiveDBName:
			if out.script, err = readLimited(tr); err != nil {
				return loaded{}, err
			}
			foundDB = true
		case strings.HasPrefix(hdr.Name, archiveFilesDir):
			if !storageKeyRule.MatchString(strings.TrimPrefix(hdr.Name, archiveFilesDir)) || hdr.Size > maxArchivedFile {
				return loaded{}, fmt.Errorf("the archive has a file that isn't a transaction attachment: %s", preview(hdr.Name))
			}
			out.files++
		}
	}
	if !foundDB {
		return loaded{}, fmt.Errorf("the archive has no %s", archiveDBName)
	}
	return out, Validate(out.script)
}

func readLimited(r io.Reader) (string, error) {
	raw, err := io.ReadAll(io.LimitReader(r, maxScriptBytes+1))
	if err != nil {
		return "", fmt.Errorf("could not read the backup file")
	}
	if len(raw) > maxScriptBytes {
		return "", fmt.Errorf("the database in this backup is larger than %d MB", maxScriptBytes>>20)
	}
	return string(raw), nil
}

// restoreFiles puts every attachment in a full backup back into storage.
func (s *Service) restoreFiles(ctx context.Context, path string) (restored int, err error) {
	if s.store == nil {
		return 0, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return 0, err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			return restored, nil
		}
		if err != nil {
			return restored, err
		}
		key := strings.TrimPrefix(hdr.Name, archiveFilesDir)
		if !strings.HasPrefix(hdr.Name, archiveFilesDir) || !storageKeyRule.MatchString(key) || hdr.Size > maxArchivedFile {
			continue
		}
		data, err := io.ReadAll(io.LimitReader(tr, maxArchivedFile+1))
		if err != nil {
			return restored, err
		}
		if err := s.store.Put(ctx, key, bytes.NewReader(data), int64(len(data)), http.DetectContentType(data)); err != nil {
			return restored, fmt.Errorf("could not put back %s: %w", key, err)
		}
		restored++
	}
}
