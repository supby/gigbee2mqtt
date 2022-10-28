package zcldef

import (
	"encoding/json"
	"log"
	"os"
)

type jsonZclMap map[string]jsonClusterDefinition

type jsonClusterDefinition struct {
	ID               uint16
	Attributes       map[string]AttributeDefinition
	Commands         map[string]CommandDefinition
	CommandsResponse map[string]CommandsResponseDefinition
}

func loadFromFile(filename string) *map[uint16]ClusterDefinition {
	_, err := os.Stat(filename)
	if os.IsNotExist(err) {
		log.Printf("ZCL definition file: %v does not exist", filename)
		return nil
	}

	var jsonLoadedMap jsonZclMap

	jsonBuf, _ := os.ReadFile(filename)
	err = json.Unmarshal(jsonBuf, &jsonLoadedMap)
	if err != nil {
		log.Printf("Failed to unmarshal JSON ZCL definition file: %v. Err: %v", filename, err)
		return nil
	}

	ret := make(map[uint16]ClusterDefinition)

	for clusterName := range jsonLoadedMap {
		jsonClusterDef := jsonLoadedMap[clusterName]

		attr := make(map[uint16]AttributeDefinition)
		for attrName := range jsonClusterDef.Attributes {
			a := jsonClusterDef.Attributes[attrName]
			a.Name = attrName
			attr[a.ID] = a
		}
		cmd := make(map[uint16]CommandDefinition)
		for cmdName := range jsonClusterDef.Commands {
			c := jsonClusterDef.Commands[cmdName]
			c.Name = cmdName
			cmd[c.ID] = c
		}
		cmdResp := make(map[uint16]CommandsResponseDefinition)
		for cmdRespName := range jsonClusterDef.CommandsResponse {
			cr := jsonClusterDef.CommandsResponse[cmdRespName]
			cr.Name = cmdRespName
			cmdResp[cr.ID] = cr
		}

		ret[jsonClusterDef.ID] = ClusterDefinition{
			ID:               jsonClusterDef.ID,
			Name:             clusterName,
			Attributes:       attr,
			Commands:         cmd,
			CommandsResponse: cmdResp,
		}
	}

	return &ret
}
