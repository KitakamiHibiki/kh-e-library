package epub

import (
	"archive/zip"
	"encoding/xml"
	"fmt"
	"io"
	"path"
	"strings"
)

// Metadata holds parsed EPUB metadata
type Metadata struct {
	Title       string
	Authors     []string
	Publisher   string
	Identifier  string // ISBN or other unique ID
	Language    string
	Description string
	CoverPath   string   // path inside EPUB to cover image
	Spine       []string // ordered list of content file paths
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

// ReadCover returns the cover image data from the EPUB.
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
				ext := path.Ext(item.Href)
				return data, ext, nil
			}
		}
	}

	return nil, "", fmt.Errorf("cover file not found in archive")
}

// --- internal ---

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
		Creator     string   `xml:"http://purl.org/dc/elements/1.1/ creator"`
		Publisher   string   `xml:"http://purl.org/dc/elements/1.1/ publisher"`
		Identifier  string   `xml:"http://purl.org/dc/elements/1.1/ identifier"`
		Language    string   `xml:"http://purl.org/dc/elements/1.1/ language"`
		Description string   `xml:"http://purl.org/dc/elements/1.1/ description"`
		Meta       []struct {
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
		Publisher:   pkg.Metadata.Publisher,
		Identifier:  pkg.Metadata.Identifier,
		Language:    pkg.Metadata.Language,
		Description: pkg.Metadata.Description,
	}

	if pkg.Metadata.Creator != "" {
		meta.Authors = append(meta.Authors, pkg.Metadata.Creator)
	}

	opfDir := path.Dir(opfPath)

	// build id -> href map
	hrefByID := make(map[string]string)
	for _, item := range pkg.Manifest.Items {
		hrefByID[item.ID] = path.Join(opfDir, item.Href)
		if item.Properties == "cover-image" {
			meta.CoverPath = path.Join(opfDir, item.Href)
		}
	}

	// fallback: look for cover in <meta name="cover">
	if meta.CoverPath == "" {
		coverID := findCoverID(pkg)
		if coverID != "" {
			meta.CoverPath = hrefByID[coverID]
		}
	}

	// fill spine
	for _, ref := range pkg.Spine.ItemRefs {
		if href, ok := hrefByID[ref.IDRef]; ok {
			meta.Spine = append(meta.Spine, href)
		}
	}

	return meta, nil
}

func findCoverID(pkg *packageXML) string {
	// EPUB 3: <meta name="cover" content="cover-image-id"/>
	for _, m := range pkg.Metadata.Meta {
		if strings.ToLower(m.Name) == "cover" && m.Content != "" {
			return m.Content
		}
	}
	// fallback: look for item with properties="cover-image"
	for _, item := range pkg.Manifest.Items {
		if item.Properties == "cover-image" {
			return item.ID
		}
	}
	return ""
}