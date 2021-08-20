package zcldef

import (
	"encoding/json"
	"os"
)

type ClusterDefinition struct {
	ID         uint16
	Name       string
	Attributes map[string]AttributeDefinition
}

type AttributeDefinition struct {
	ID   uint16
	Name string
	Type byte
}

type ZCLMap map[string]ClusterDefinition

func Load(filename string) *ZCLMap {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		return nil
	}

	var loadedMap ZCLMap

	jsonBuf, _ := os.ReadFile(filename)
	json.Unmarshal(jsonBuf, &loadedMap)

	return &loadedMap
}
