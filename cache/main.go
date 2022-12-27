package cache

import (
	"encoding/json"
	"os"

	"github.com/adrg/xdg"
	log "github.com/sirupsen/logrus"
)

func path() string {
	path, err := xdg.CacheFile("bump.json")
	if err != nil {
		panic(err)
	}

	return path
}

func Write(data interface{}) {
	// dont forget to import "encoding/json"
	dataJSON, err := json.MarshalIndent(data, "", "    ")
	if err != nil {
		panic(err)
	}

	err = os.WriteFile(path(), dataJSON, 0o644)
	if err != nil {
		panic(err)
	}

	log.WithFields(log.Fields{
		"file": path(),
	}).Info("wrote to cache")
}

func Load() []byte {
	data, err := os.ReadFile(path())
	if err != nil {
		panic(err)
	}

	return data
}
