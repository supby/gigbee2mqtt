package zcldef

type ClusterDefinition struct {
	ID               uint16
	Name             string
	Attributes       map[uint16]AttributeDefinition
	Commands         map[uint16]CommandDefinition
	CommandsResponse map[uint16]CommandsResponseDefinition
}

type AttributeDefinition struct {
	ID   uint16
	Name string
	Type byte
}

type CommandDefinition struct {
	ID         uint16
	Name       string
	Parameters [][]string
}

type CommandsResponseDefinition struct {
	ID         uint16
	Name       string
	Parameters [][]string
}
