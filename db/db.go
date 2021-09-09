package db

import (
	"encoding/json"
	"log"
	"os"
	"time"
)

type Node struct {
	IEEEAddress    uint64
	NetworkAddress uint16
	LogicalType    uint8
	LQI            uint8
	Depth          uint8
	LastDiscovered time.Time
	LastReceived   time.Time
}

type DB struct {
	filename string
	Nodes    []Node
}

func Init(filename string) *DB {
	ret := DB{
		filename: filename,
	}

	ret.load()

	return &ret
}

func (db *DB) GetNodes() []Node {
	return db.Nodes
}

func (db *DB) SaveNode(node Node) {
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

func (db *DB) save() {
	log.Println("Saving node to DB")

	res, _ := json.Marshal(db)
	os.WriteFile(db.filename, res, 0644)
}

func (db *DB) load() {
	_, err := os.Stat(db.filename)
	if os.IsNotExist(err) {
		return
	}

	var loadedDB DB

	jsonBuf, _ := os.ReadFile(db.filename)
	json.Unmarshal(jsonBuf, &loadedDB)

	db.Nodes = make([]Node, 0)
	if loadedDB.Nodes != nil && len(loadedDB.Nodes) > 0 {
		db.Nodes = loadedDB.Nodes
	}

	log.Printf("[DB] %v nodes are loaded from DB\n", len(db.Nodes))
}
