package git

import (
	"archive/tar"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// ExportTree writes the tree of commit into destination using `git archive`,
// extracted in-process so that no host tar executable is required.
func (c *Client) ExportTree(ctx context.Context, root, commit, destination string) error {
	data, err := c.run(ctx, root, "archive", "--format=tar", commit)
	if err != nil {
		return err
	}
	return extractTar(bytes.NewReader(data), destination)
}

// extractTar unpacks regular files and directories, rejecting entries that
// would escape the destination. Symbolic links and other entry types are skipped.
func extractTar(r io.Reader, destination string) error {
	cleanDestination := filepath.Clean(destination)
	reader := tar.NewReader(r)
	for {
		header, err := reader.Next()
		if errors.Is(err, io.EOF) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read archive: %w", err)
		}
		target := filepath.Join(cleanDestination, filepath.FromSlash(header.Name))
		if target != cleanDestination && !strings.HasPrefix(target, cleanDestination+string(os.PathSeparator)) {
			return fmt.Errorf("archive entry %q escapes the destination", header.Name)
		}
		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0o755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := writeEntry(target, reader); err != nil {
				return err
			}
		default:
			// Symbolic links, PAX global headers and other entry types are not part of the analyzed tree.
		}
	}
}

func writeEntry(target string, content io.Reader) error {
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		return err
	}
	file, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return err
	}
	if _, err := io.Copy(file, content); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}
