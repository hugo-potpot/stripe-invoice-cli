package archive

import (
	"archive/zip"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"stripe-invoice-go/internal/domain"
)

type FileArchiver struct {
	baseDir string
}

func NewFileArchiver(baseDir string) *FileArchiver {
	return &FileArchiver{
		baseDir: baseDir,
	}
}

func (a *FileArchiver) WriteInvoice(ctx context.Context, period domain.Period, accountID int64, merchant domain.Merchant, pdf []byte) error {
	path := a.exportDir(period, accountID)

	err := createDirIfNotExists(path)

	if err != nil {
		return err
	}

	fileName := fmt.Sprintf("Stripe Tax Invoice %s-%s.pdf", strings.ToUpper(merchant.Identify), period.PreviousMonth().String())
	fullPath := filepath.Join(path, fileName)
	if err := os.WriteFile(fullPath, pdf, 0o644); err != nil {
		return err
	}

	slog.InfoContext(ctx, "invoice written", "merchant", merchant.Name, "path", fullPath)
	return nil
}

func (a *FileArchiver) ZipAccount(ctx context.Context, period domain.Period, accountID int64) (string, error) {
	dir := a.exportDir(period, accountID)
	zipPath := dir + ".zip"

	slog.InfoContext(ctx, "creating zip archive", "path", zipPath, "source_dir", dir)
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return "", err
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	invoices, err := os.ReadDir(dir)
	if err != nil {
		return "", err
	}

	for _, invoice := range invoices {
		file, err := zipWriter.Create(invoice.Name())
		if err != nil {
			return "", err
		}
		srcFile, err := os.Open(filepath.Join(dir, invoice.Name()))
		if err != nil {
			return "", err
		}

		_, err = io.Copy(file, srcFile)
		if err != nil {
			return "", err
		}

		srcFile.Close()
	}

	slog.InfoContext(ctx, "zip archive created", "path", zipPath, "files", len(invoices))
	return zipPath, nil
}

func (a *FileArchiver) WriteNewMerchantsCSV(ctx context.Context, period domain.Period, accountID int64, merchants []domain.Merchant) (string, error) {
	dir := a.exportDir(period, accountID)
	path := filepath.Join(dir, "new_merchants.csv")

	err := createDirIfNotExists(dir)
	if err != nil {
		return "", err
	}

	slog.InfoContext(ctx, "writing new merchants csv", "path", path, "count", len(merchants))
	csvFile, err := os.Create(path)
	if err != nil {
		return "", err
	}
	defer csvFile.Close()

	writer := csv.NewWriter(csvFile)

	if err := writer.Write([]string{"token", "name", "identify"}); err != nil {
		return "", err
	}

	for _, merchant := range merchants {
		if err := writer.Write([]string{
			merchant.Token,
			merchant.Name,
			merchant.Identify,
		}); err != nil {
			return "", err
		}
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}

	return csvFile.Name(), nil
}

func (a *FileArchiver) Attachments(ctx context.Context, period domain.Period, accountID int64) ([]string, error) {
	dir := a.exportDir(period, accountID)

	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return nil, err
	}

	var files []string

	zipPath := dir + ".zip"
	if _, err := os.Stat(zipPath); os.IsNotExist(err) {
		return nil, err
	}

	files = append(files, zipPath)

	csvPath := dir + "/new_merchants.csv"
	if _, err := os.Stat(csvPath); err == nil {
		files = append(files, csvPath)
	}

	return files, nil

}

func (a *FileArchiver) exportDir(period domain.Period, accountID int64) string {
	return filepath.Join(a.baseDir, fmt.Sprintf("%s_%d", period.String(), accountID))
}

func createDirIfNotExists(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("creating dir %q: %w", dir, err)
		}
	}
	return nil
}
