package archiver

import (
	"archive/zip"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	log "github.com/sirupsen/logrus"
)

// Archive is the public pointer to the initialized archiver object
var Archive *Archiver

// Archiver manages download links and serving asset archive downloads
type Archiver struct {
	ACServerInstallPath string
	AuthorsBlacklist    []string
	BaseDomain          string // should be a valid hostname, for example http://www.google.com
	CachePath           string
	OverwriteURL        bool
}

// New returns an Archiver pointer
func New(acServerPath string, cachePath string, domain string, authorsBlacklist []string, overwrite bool) *Archiver {
	Archive = &Archiver{
		ACServerInstallPath: acServerPath,
		AuthorsBlacklist:    authorsBlacklist,
		BaseDomain:          domain,
		CachePath:           cachePath,
		OverwriteURL:        overwrite,
	}

	errs := Archive.SetAssetDownloadURLs()

	if len(errs) > 0 {
		log.Warn(errs)
	}

	return Archive
}

// isAuthorBlacklisted returns true if an asset author has been blacklisted
func (a *Archiver) isAuthorBlacklisted(author string) bool {
	blacklisted := false

	for _, b := range a.AuthorsBlacklist {
		if author == b {
			blacklisted = true
			break
		}
	}

	return blacklisted
}

// setDownloadURL reads an asset json file and sets an Assetto Server Manager download url for any
// asset whose author is not in the authorsBlacklist, and if overwriteURL is set to true when a download
// URL is already set
func (a *Archiver) setDownloadURL(dir os.FileInfo, c assetType) error {
	// build the asset ui json file path
	jsondatapath := filepath.Join(a.ACServerInstallPath, c.Folder(), dir.Name(), c.Data())

	// read the asset json data
	data, err := ioutil.ReadFile(jsondatapath)
	if err != nil {
		return err
	}

	// unmarshal the json
	var j map[string]interface{}
	err = json.Unmarshal([]byte(data), &j)
	if err != nil {
		return err
	}

	// check if asset author is in blacklist
	blacklisted := a.isAuthorBlacklisted(fmt.Sprintf("%v", j["author"]))
	if blacklisted == true {
		return nil
	}

	// check if a download URL is already set and if it should be overwritten
	if !a.OverwriteURL && j["downloadURL"] != nil && j["downloadURL"] != "" {
		return nil
	}

	// build the download URL. hard coding "download" as the service handler path
	j["downloadURL"] = a.BaseDomain + "/download/" + c.Name() + "/" + url.QueryEscape(dir.Name())

	// marshal the new asset data back to json pretty printed
	updated, err := json.MarshalIndent(j, "", "  ")
	if err != nil {
		return err
	}

	// update the asset json file
	err = ioutil.WriteFile(jsondatapath, updated, 0644)
	if err != nil {
		return err
	}

	return nil
}

// SetAssetDownloadURL walks the specified content type folder and updates the download
// URLs for those assets
func (a *Archiver) SetAssetDownloadURL(c assetType) []error {
	// store all errors encountered while processing files in a slice
	var errs []error

	items, err := ioutil.ReadDir(filepath.Join(a.ACServerInstallPath, c.Folder()))
	if err != nil {
		errs = append(errs, err)
	}

	for _, t := range items {
		a.setDownloadURL(t, c)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

// SetAssetDownloadURLs walks all the defined contentType folders and sets the Assetto Server Manager
// download URL for all assets
func (a *Archiver) SetAssetDownloadURLs() []error {
	// store all errors encountered while processing files in a slice
	var errs []error

	log.Debug("set cars download URLs")

	err := a.SetAssetDownloadURL(Car{})
	if err != nil {
		errs = append(errs, err...)
	}

	log.Debug("set tracks download URLs")

	err = a.SetAssetDownloadURL(Track{})
	if err != nil {
		errs = append(errs, err...)
	}

	return errs
}

// GetCached returns the path to an existing archive if it exists
func (a *Archiver) GetCached(assetType assetType, assetName string) []byte {
	path := filepath.Join(a.CachePath, assetType.Name(), assetName) + ".zip"

	f, _ := os.Stat(path)

	if f == nil {
		return nil
	}

	content, _ := ioutil.ReadFile(path)

	return content
}

// AssetExists returns true if a requested asset is available
func (a *Archiver) AssetExists(assetType assetType, assetName string) bool {
	exists := false
	path := filepath.Join(a.ACServerInstallPath, assetType.Folder(), assetName)

	f, _ := os.Stat(path)

	if f != nil {
		exists = true
	}

	return exists
}

// Create is a helper function to zip an AC asset folder and place the assets in the correct
// directory structure inside the zip for Content Manager to automatically install it
func (a *Archiver) Create(assetType assetType, name string) error {

	var cachepath string

	if filepath.IsAbs(a.CachePath) {
		cachepath = a.CachePath
	} else {
		dir, err := os.Getwd()
		if err != nil {
			return err
		}
		cachepath = filepath.Join(dir, a.CachePath)
	}

	log.Info("CachePath !!!! " + cachepath)

	assetpath := filepath.Join(assetType.Folder(), name)
	bundlepath := filepath.Join(cachepath, assetType.Name(), name)

	// Make sure the cachepath and asset directory exist
	err := os.MkdirAll(filepath.Join(cachepath, assetType.Name()), 0666)
	if err != nil {
		return err
	}

	// Get a slice of all the files that need to be archived
	files, err := getAllFiles(filepath.Join(a.ACServerInstallPath, assetpath))
	if err != nil {
		return err
	}

	// Create a file descriptor for the zip
	zipArchive, err := os.Create(bundlepath + ".zip")
	if err != nil {
		return err
	}
	defer zipArchive.Close()

	// Create a Writer to add files to the archive
	zipWriter := zip.NewWriter(zipArchive)
	defer zipWriter.Close()

	for _, f := range files {

		// Split the full file path on the known assetpath to get the relative path for the file
		relative := strings.SplitN(f, assetpath, 2)
		// Content Manager needs assets in a specific directory structure create that
		// correct path for the asset file
		archivepath := filepath.Join(assetpath, relative[1])

		// Read file
		source, err := os.Open(f)
		if err != nil {
			return err
		}
		defer source.Close()

		// Get the file information
		info, err := source.Stat()
		if err != nil {
			return err
		}

		// Create a file header for the new file in the zip archive
		header, err := zip.FileInfoHeader(info)
		if err != nil {
			return err
		}

		// Set the file to be archived with a Content Manager compliant path
		header.Name = archivepath
		// Use Deflate
		header.Method = zip.Deflate

		log.WithFields(log.Fields{
			"filePath":    f,
			"archivePath": archivepath,
		}).Debug("add file to archive")

		// Add the new zip file header to the archive
		writer, err := zipWriter.CreateHeader(header)
		if err != nil {
			return err
		}

		// Copy the source file data
		_, err = io.Copy(writer, source)
		if err != nil {
			return err
		}
	}

	return nil
}

// getAllFiles walks a directory recursively and returns a slice with all the files
func getAllFiles(dir string) ([]string, error) {
	var files []string

	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		files = append(files, path)
		return nil
	})

	if err != nil {
		return files, err
	}

	return files, nil
}
