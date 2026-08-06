package service

import (
	"bytes"

	"github.com/kitakami-hibiki/e-library/internal/model"
	pdfapi "github.com/pdfcpu/pdfcpu/pkg/api"
	pdfmodel "github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// extractPDFMetadata reads PDF metadata using pdfcpu.
func extractPDFMetadata(data []byte) (*model.PDFMetadata, error) {
	rs := bytes.NewReader(data)
	ctx, err := pdfapi.ReadContext(rs, pdfmodel.NewDefaultConfiguration())
	if err != nil {
		return nil, err
	}

	meta := &model.PDFMetadata{
		Title:       ctx.Title,
		Author:      ctx.Author,
		Description: ctx.Subject,
		Pages:       ctx.PageCount,
	}

	return meta, nil
}

// extractPDFCover extracts the first page image from a PDF.
func extractPDFCover(data []byte) ([]byte, error) {
	rs := bytes.NewReader(data)
	conf := pdfmodel.NewDefaultConfiguration()

	var coverData []byte
	handler := func(img pdfmodel.Image, _ bool, _ int) error {
		if coverData != nil {
			return nil // only take the first image
		}
		buf := &bytes.Buffer{}
		if _, err := buf.ReadFrom(img); err != nil {
			return err
		}
		coverData = buf.Bytes()
		return nil
	}

	if err := pdfapi.ExtractImages(rs, []string{"1"}, handler, conf); err != nil {
		return nil, err
	}

	return coverData, nil
}
