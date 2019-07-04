package util

import (
	"gopkg.in/yaml.v2"
	"errors"
	"os"
)

func ParseYamlFromFile(filePath string, tolerateEmptyInput bool) (map[string]interface{}, error) {
	var fromFile *os.File
	var err error
	var yamlContent map[string]interface{}

	if filePath == "" {
		err = nil
		if !tolerateEmptyInput {
			err = errors.New("No file path given")
		}
		return nil, err
	}

	fromFile, err = os.Open(filePath)
	if err != nil {
		return nil, err
	}

	fi, err := fromFile.Stat()
	if err != nil {
		return nil, err
	}

	if fi.Size() <= 0 {
		err = nil
		if !tolerateEmptyInput {
			err = errors.New("No yaml input to parse")
		}
		return nil, err
	}

	err = yaml.NewDecoder(fromFile).Decode(&yamlContent)
	if err != nil {
		return nil, err
	}
	return yamlContent, nil
}
