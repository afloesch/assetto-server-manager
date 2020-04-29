package archiver

import (
	"bytes"
	"net/http"

	"github.com/go-chi/chi"
	log "github.com/sirupsen/logrus"
)

// Handler is the http handler for requests to /download
func (a *Archiver) Handler(w http.ResponseWriter, r *http.Request) {

	if Archive == nil {
		w.WriteHeader(404)
		_, err := w.Write([]byte("not found"))
		if err != nil {
			log.Error(err)
		}
		return
	}

	name := chi.URLParam(r, "name")
	asset := chi.URLParam(r, "type")

	var assettype assetType

	if asset == "tracks" {
		assettype = Track{}
	} else {
		assettype = Car{}
	}

	cached := a.GetCached(assettype, name)
	if cached != nil {
		w.Header().Set("Content-type", "application/zip")
		w.WriteHeader(200)

		buf := bytes.NewBuffer(cached)
		_, err := buf.WriteTo(w)
		if err != nil {
			log.Error(err)
		}
		return
	}

	exists := a.AssetExists(assettype, name)
	if !exists {
		w.WriteHeader(404)
		_, err := w.Write([]byte("not found"))
		if err != nil {
			log.Error(err)
		}
		return
	}

	err := a.Create(assettype, name)
	if err != nil {
		w.WriteHeader(500)
		_, err := w.Write([]byte("service error"))
		if err != nil {
			log.Error(err)
		}
		return
	}

	cached = a.GetCached(assettype, name)
	w.Header().Set("Content-type", "application/zip")
	w.WriteHeader(200)

	buf := bytes.NewBuffer(cached)
	_, err = buf.WriteTo(w)
	if err != nil {
		log.Error(err)
	}
}