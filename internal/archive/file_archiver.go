package archive

import (
	"archive/zip"
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"log"
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

	return nil
}

func (a *FileArchiver) ZipAccount(ctx context.Context, period domain.Period, accountID int64) (string, error) {
	dir := a.exportDir(period, accountID)
	zipPath := dir + ".zip"

	log.Println("Creating zip archive...")
	zipFile, err := os.Create(zipPath)
	if err != nil {
		return "", err
	}
	defer zipFile.Close()

	log.Println("Zipping archive...")
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

	return zipPath, nil
}

func (a *FileArchiver) WriteNewMerchantsCSV(ctx context.Context, period domain.Period, accountID int64, merchants []domain.Merchant) (string, error) {
	dir := a.exportDir(period, accountID)
	path := filepath.Join(dir, "new_merchants.csv")

	err := createDirIfNotExists(dir)
	if err != nil {
		return "", err
	}

	log.Println("Creating csv archive...")
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
		writer.Write([]string{
			merchant.Token,
			merchant.Name,
			merchant.Identify,
		})
		log.Printf("Adding %s to csv archive...\n", merchant.Name)
	}
	writer.Flush()
	if err := writer.Error(); err != nil {
		return "", err
	}

	return csvFile.Name(), nil
}

func (a *FileArchiver) exportDir(period domain.Period, accountID int64) string {
	return filepath.Join(a.baseDir, fmt.Sprintf("%s_%d", period.String(), accountID))
}

func createDirIfNotExists(dir string) error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		err := os.MkdirAll(dir, 0o755)
		if err != nil {
			log.Println(err)
			return err
		}
	}
	return nil
}
