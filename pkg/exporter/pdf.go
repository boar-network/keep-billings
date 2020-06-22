package exporter

import (
	"bytes"
	"github.com/SebastiaanKlippert/go-wkhtmltopdf"
	"html/template"
)

type PdfExporter struct {
	pdfTemplate *template.Template
}

func NewPdfExporter(templateFilename string) (*PdfExporter, error) {
	pdfTemplate, err := template.ParseFiles(templateFilename)
	if err != nil {
		return nil, err
	}

	return &PdfExporter{pdfTemplate}, nil
}

func (pe *PdfExporter) Export(data interface{}) ([]byte, error) {
	buffer := &bytes.Buffer{}
	err := pe.pdfTemplate.Execute(buffer, data)
	if err != nil {
		return nil, err
	}

	pdf, err := wkhtmltopdf.NewPDFGenerator()
	if err != nil {
		return nil, err
	}

	pdf.AddPage(wkhtmltopdf.NewPageReader(buffer))

	err = pdf.Create()
	if err != nil {
		return nil, err
	}

	return pdf.Bytes(), nil
}
