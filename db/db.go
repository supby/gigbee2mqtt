package db

import (
	"encoding/json"
	"log"
	"os"
)

type IDevicesRepo interface {
	GetNodes() []Node
	SaveNode(node Node)
}

func Init(filename string) IDevicesRepo {
	ret := devicesRepo{
		filename: filename,
	}

	ret.load()

	return &ret
}

type devicesRepo struct {
	filename string
	Nodes    []Node
}

func (db *devicesRepo) GetNodes() []Node {
	return db.Nodes
}

func (db *devicesRepo) SaveNode(node Node) {
	existingNodeIndex := -1
	for i, n := range db.Nodes {
		if n.IEEEAddress == node.IEEEAddress {
			existingNodeIndex = i
			break
		}
	}
	if existingNodeIndex > -1 {
		db.Nodes[existingNodeIndex] = node
	} else {
		db.Nodes = append(db.Nodes, node)
	}

	db.save()
}

func (db *devicesRepo) save() {
	log.Println("Saving node to DB")

	res, _ := json.Marshal(db)
	os.WriteFile(db.filename, res, 0644)
}

func (db *devicesRepo) load() {
	_, err := os.Stat(db.filename)
	if os.IsNotExist(err) {
		return
	}

	var loadedDB devicesRepo

	jsonBuf, _ := os.ReadFile(db.filename)
	json.Unmarshal(jsonBuf, &loadedDB)

	db.Nodes = make([]Node, 0)
	if loadedDB.Nodes != nil && len(loadedDB.Nodes) > 0 {
		db.Nodes = loadedDB.Nodes
	}

	log.Printf("[DB] %v nodes are loaded from DB\n", len(db.Nodes))
}
