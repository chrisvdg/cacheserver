package cache

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/pkg/errors"
)

const (
	filePerm os.FileMode = 0666
	dirPerm  os.FileMode = 0700
)

var (
	nosave = false
)

// save writes the current file backend data to the backend file
func (b *backend) save() error {
	if nosave {
		return nil
	}
	data, err := json.MarshalIndent(b.data, "", "\t")
	if err != nil {
		return errors.Wrap(err, "failed to marshal backend data to json")
	}
	err = ioutil.WriteFile(b.filePath, data, filePerm)
	if err != nil {
		return errors.Wrap(err, "failed to open file for writing")
	}
	return nil
}

// read reads the backend file to in memory objects for the file backend
func (b *backend) read() error {
	data, err := ioutil.ReadFile(b.filePath)
	if err != nil {
		return errors.Wrap(err, "failed to read backend file")
	}

	if string(data) == "" || string(data) == "[]" {
		return nil
	}
	err = json.Unmarshal(data, &b.data)
	if err != nil {
		return errors.Wrap(err, "failed to parse data from backend file")
	}

	return nil
}

// ensureFile ensures that the backend file exists
func (b *backend) ensureFile() error {
	file, err := os.OpenFile(b.filePath, os.O_RDONLY|os.O_CREATE, filePerm)
	if err != nil {
		return errors.Wrap(err, "something went wrong creating/reading backend file")
	}

	return file.Close()
}
