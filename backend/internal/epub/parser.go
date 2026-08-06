package epub

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"regexp"
	"strings"
)

// Metadata holds parsed EPUB metadata.
type Metadata struct {
	Title       string
	Authors     []string
	Publisher   string
	ISBN        string   // only valid ISBN-10 or ISBN-13
	Language    string
	Description string
	Subjects    []string // dc:subject values for auto-tagging
	Pages       int      // spine itemref count
}

// Validate checks that an EPUB file has the required structure.
func Validate(reader io.ReaderAt, size int64) error {
	zr, err := zip.NewReader(reader, size)
	if err != nil {
		return fmt.Errorf("无法打开 ZIP 文件")
	}

	files := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		files[f.Name] = f
	}

	// Check mimetype file exists and has correct content
	mf, ok := files["mimetype"]
	if !ok {
		return fmt.Errorf("EPUB 缺少 mimetype 文件")
	}
	rc, err := mf.Open()
	if err != nil {
		return fmt.Errorf("无法读取 mimetype 文件")
	}
	mimetype, err := io.ReadAll(rc)
	rc.Close()
	if err != nil {
		return fmt.Errorf("无法读取 mimetype 文件")
	}
	if strings.TrimSpace(string(mimetype)) != "application/epub+zip" {
		return fmt.Errorf("EPUB mimetype 无效")
	}

	// Check META-INF/container.xml exists
	if _, ok := files["META-INF/container.xml"]; !ok {
		return fmt.Errorf("EPUB 缺少 META-INF/container.xml")
	}

	return nil
}

// Parse reads an EPUB file and extracts metadata.
func Parse(reader io.ReaderAt, size int64) (*Metadata, error) {
	zr, err := zip.NewReader(reader, size)
	if err != nil {
		return nil, fmt.Errorf("open zip: %w", err)
	}

	files := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		files[f.Name] = f
	}

	opfPath, err := getOpfPath(files)
	if err != nil {
		return nil, err
	}

	return parseOpf(files, opfPath)
}

// ReadCover returns the cover image data and extension from the EPUB.
// The extension is determined by magic bytes (actual file content),
// not by the manifest media-type or file path extension.
func ReadCover(reader io.ReaderAt, size int64) ([]byte, string, error) {
	zr, err := zip.NewReader(reader, size)
	if err != nil {
		return nil, "", err
	}
	files := make(map[string]*zip.File, len(zr.File))
	for _, f := range zr.File {
		files[f.Name] = f
	}
	opfPath, err := getOpfPath(files)
	if err != nil {
		return nil, "", err
	}

	opfDir := path.Dir(opfPath)
	pkg, err := readOPF(files, opfPath)
	if err != nil {
		return nil, "", err
	}

	coverID := findCoverID(pkg)
	if coverID == "" {
		return nil, "", fmt.Errorf("no cover found")
	}

	for _, item := range pkg.Manifest.Items {
		if item.ID == coverID {
			imgPath := path.Join(opfDir, item.Href)
			if f, ok := files[imgPath]; ok {
				rc, err := f.Open()
				if err != nil {
					return nil, "", err
				}
				defer rc.Close()
				data, err := io.ReadAll(rc)
				if err != nil {
					return nil, "", err
				}
				// Determine extension by magic bytes
				ext := detectImageExt(data)
				return data, ext, nil
			}
		}
	}

	return nil, "", fmt.Errorf("cover file not found in archive")
}

// --- Internal ---

type containerXML struct {
	Rootfiles struct {
		Rootfile struct {
			Path string `xml:"full-path,attr"`
		} `xml:"rootfile"`
	} `xml:"rootfiles"`
}

type packageXML struct {
	Metadata struct {
		Title       string   `xml:"http://purl.org/dc/elements/1.1/ title"`
		Creators    []string `xml:"http://purl.org/dc/elements/1.1/ creator"`
		Publisher   string   `xml:"http://purl.org/dc/elements/1.1/ publisher"`
		Identifiers []string `xml:"http://purl.org/dc/elements/1.1/ identifier"`
		Language    string   `xml:"http://purl.org/dc/elements/1.1/ language"`
		Description string   `xml:"http://purl.org/dc/elements/1.1/ description"`
		Subjects    []string `xml:"http://purl.org/dc/elements/1.1/ subject"`
		Meta        []struct {
			Name    string `xml:"name,attr"`
			Content string `xml:"content,attr"`
		} `xml:"meta"`
	} `xml:"metadata"`
	Manifest struct {
		Items []struct {
			ID         string `xml:"id,attr"`
			Href       string `xml:"href,attr"`
			MediaType  string `xml:"media-type,attr"`
			Properties string `xml:"properties,attr,omitempty"`
		} `xml:"item"`
	} `xml:"manifest"`
	Spine struct {
		ItemRefs []struct {
			IDRef string `xml:"idref,attr"`
		} `xml:"itemref"`
	} `xml:"spine"`
}

func getOpfPath(files map[string]*zip.File) (string, error) {
	cf, ok := files["META-INF/container.xml"]
	if !ok {
		return "", fmt.Errorf("META-INF/container.xml not found")
	}
	rc, err := cf.Open()
	if err != nil {
		return "", fmt.Errorf("open container.xml: %w", err)
	}
	defer rc.Close()

	var cont containerXML
	if err := xml.NewDecoder(rc).Decode(&cont); err != nil {
		return "", fmt.Errorf("parse container.xml: %w", err)
	}
	if cont.Rootfiles.Rootfile.Path == "" {
		return "", fmt.Errorf("empty rootfile path in container.xml")
	}
	return cont.Rootfiles.Rootfile.Path, nil
}

func readOPF(files map[string]*zip.File, opfPath string) (*packageXML, error) {
	f, ok := files[opfPath]
	if !ok {
		return nil, fmt.Errorf("opf file %s not found", opfPath)
	}
	rc, err := f.Open()
	if err != nil {
		return nil, err
	}
	defer rc.Close()
	var pkg packageXML
	if err := xml.NewDecoder(rc).Decode(&pkg); err != nil {
		return nil, fmt.Errorf("parse opf: %w", err)
	}
	return &pkg, nil
}

func parseOpf(files map[string]*zip.File, opfPath string) (*Metadata, error) {
	pkg, err := readOPF(files, opfPath)
	if err != nil {
		return nil, err
	}

	meta := &Metadata{
		Title:       pkg.Metadata.Title,
		Authors:     pkg.Metadata.Creators,
		Publisher:   pkg.Metadata.Publisher,
		Language:    pkg.Metadata.Language,
		Description: pkg.Metadata.Description,
		Subjects:    pkg.Metadata.Subjects,
	}

	// Extract ISBN from identifiers
	for _, id := range pkg.Metadata.Identifiers {
		if isValidISBN(id) {
			meta.ISBN = id
			break
		}
	}

	// Fill spine count as Pages
	meta.Pages = len(pkg.Spine.ItemRefs)

	return meta, nil
}

func findCoverID(pkg *packageXML) string {
	// EPUB 3: prefer <item properties="cover-image"/>
	for _, item := range pkg.Manifest.Items {
		if item.Properties == "cover-image" {
			return item.ID
		}
	}
	// Fallback: <meta name="cover" content="cover-image-id"/>
	for _, m := range pkg.Metadata.Meta {
		if strings.ToLower(m.Name) == "cover" && m.Content != "" {
			return m.Content
		}
	}
	return ""
}

// isValidISBN checks if a string is a valid ISBN-10 or ISBN-13.
func isValidISBN(s string) bool {
	// Remove hyphens and spaces
	clean := strings.ReplaceAll(s, "-", "")
	clean = strings.ReplaceAll(clean, " ", "")

	// ISBN-10: 10 digits, last may be X
	if matched, _ := regexp.MatchString(`^\d{9}[\dXx]$`, clean); matched {
		return true
	}
	// ISBN-13: 13 digits starting with 978 or 979
	if matched, _ := regexp.MatchString(`^97[89]\d{10}$`, clean); matched {
		return true
	}
	return false
}

// detectImageExt determines the image extension from magic bytes.
func detectImageExt(data []byte) string {
	if len(data) < 4 {
		return ".jpg" // default
	}
	// JPEG: FF D8 FF
	if data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF {
		return ".jpg"
	}
	// PNG: 89 50 4E 47
	if data[0] == 0x89 && data[1] == 0x50 && data[2] == 0x4E && data[3] == 0x47 {
		return ".png"
	}
	// GIF: 47 49 46 38
	if data[0] == 0x47 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x38 {
		return ".gif"
	}
	// WebP: 52 49 46 46 ... 57 45 42 50
	if len(data) >= 12 && data[0] == 0x52 && data[1] == 0x49 && data[2] == 0x46 && data[3] == 0x46 &&
		data[8] == 0x57 && data[9] == 0x45 && data[10] == 0x42 && data[11] == 0x50 {
		return ".webp"
	}
	return ".jpg" // default fallback
}
