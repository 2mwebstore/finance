package backup

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testScript = "SET NAMES utf8mb4;\nDROP TABLE IF EXISTS `t`;\nCREATE TABLE `t` (id int);\nINSERT INTO `t` VALUES (1);\n"

func buildArchive(t *testing.T, entries map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	for name, body := range entries {
		if err := tw.WriteHeader(&tar.Header{Name: name, Mode: 0o640, Size: int64(len(body))}); err != nil {
			t.Fatal(err)
		}
		if _, err := io.WriteString(tw, body); err != nil {
			t.Fatal(err)
		}
	}
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

const fileKey = "transactions/7a1f0c39-6c41-4f0e-9a52-2f3a9f6f4c11/ab12cd34-receipt.png"

func TestLoadBackupFormats(t *testing.T) {
	ld, err := loadBackup(bytes.NewReader(buildArchive(t, map[string]string{archiveDBName: testScript, archiveFilesDir + fileKey: "PNG", archiveManifest: "{}"})))
	if err != nil || !ld.archive || ld.files != 1 || ld.script != testScript {
		t.Fatalf("archive: %+v %v", ld, err)
	}
	if ld, err := loadBackup(strings.NewReader(testScript)); err != nil || ld.archive || ld.script != testScript {
		t.Fatalf("plain .sql: %+v %v", ld, err)
	}
	var gzSQL bytes.Buffer
	w := gzip.NewWriter(&gzSQL)
	io.WriteString(w, testScript)
	w.Close()
	if ld, err := loadBackup(&gzSQL); err != nil || ld.archive {
		t.Fatalf(".sql.gz: %+v %v", ld, err)
	}
}

func TestLoadBackupRefusesUnsafeArchives(t *testing.T) {
	for name, entries := range map[string]map[string]string{
		"path escape":    {archiveDBName: testScript, archiveFilesDir + "../../etc/passwd": "x"},
		"other folder":   {archiveDBName: testScript, archiveFilesDir + "config/app.env": "x"},
		"no database":    {archiveFilesDir + fileKey: "x"},
		"bad statements": {archiveDBName: "CREATE TABLE `t` (id int); DROP DATABASE prod;"},
	} {
		if _, err := loadBackup(bytes.NewReader(buildArchive(t, entries))); err == nil {
			t.Errorf("%s: archive accepted", name)
		}
	}
}

type memStore struct{ files map[string][]byte }

func (m *memStore) Put(_ context.Context, key string, r io.Reader, _ int64, _ string) error {
	data, err := io.ReadAll(r)
	m.files[key] = data
	return err
}
func (m *memStore) Open(_ context.Context, key string) (io.ReadCloser, error) {
	return io.NopCloser(bytes.NewReader(m.files[key])), nil
}
func (m *memStore) Delete(_ context.Context, key string) error { delete(m.files, key); return nil }
func (m *memStore) Kind() string                                { return "memory" }

func TestRestoreFilesPutsAttachmentsBack(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "backup-20261023-120000-manual.tar.gz")
	if err := os.WriteFile(path, buildArchive(t, map[string]string{archiveDBName: testScript, archiveFilesDir + fileKey: "PNGDATA"}), 0o600); err != nil {
		t.Fatal(err)
	}
	store := &memStore{files: map[string][]byte{}}
	s := &Service{store: store, dir: dir}
	n, err := s.restoreFiles(context.Background(), path)
	if err != nil || n != 1 || string(store.files[fileKey]) != "PNGDATA" {
		t.Fatalf("restored %d (%v), store %v", n, err, store.files)
	}
}
