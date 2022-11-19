package common

import (
	"encoding/json"
	"os"

	"github.com/sirupsen/logrus"
)

/* Read json from file */
func ReadJsonFile[T any](filePath string, ptr *T) (error) {
	file, err := os.Open(filePath)
	if err != nil {
		logrus.Errorf("Failed to open file, %v", err)
		return err
	}
	defer file.Close()

	if err = json.NewDecoder(file).Decode(ptr); err != nil {
		logrus.Errorf("Failed to decode json, %v", err)
		return err
	}
	return nil
}