package db

import (
	"encoding/json"
	"log"
	"os"
	"sync"
)

type DevicesRepo interface {
	GetNodes() []Node
	SaveNode(node Node)
}

type DBOption struct {
	Filename   string
	FlushAfter uint
}

func Init(options DBOption) DevicesRepo {
	ret := devicesRepo{
		options: options,
		Nodes:   make([]Node, 0),
	}

	ret.init()

	return &ret
}

type devicesRepo struct {
	Nodes       []Node
	mtx         sync.Mutex
	options     DBOption
	saveCounter uint
}

func (db *devicesRepo) GetNodes() []Node {
	return db.Nodes
}

func (db *devicesRepo) SaveNode(node Node) {
	db.mtx.Lock()
	defer db.mtx.Unlock()

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

	db.saveCounter++

	if db.saveCounter < db.options.FlushAfter {
		return
	}

	db.saveCounter = 0

	db.flush()
}

func (db *devicesRepo) flush() {
	log.Println("[DB] Flushing DB to file.")

	res, _ := json.Marshal(db)
	os.WriteFile(db.options.Filename, res, 0644)
}

func (db *devicesRepo) init() {
	_, err := os.Stat(db.options.Filename)
	if os.IsNotExist(err) {
		log.Printf("[DB] File %v is not found. Using empty state.\n", db.options.Filename)
		return
	}

	var loadedDB devicesRepo

	jsonBuf, _ := os.ReadFile(db.options.Filename)
	json.Unmarshal(jsonBuf, &loadedDB)

	db.Nodes = make([]Node, 0)
	if loadedDB.Nodes != nil && len(loadedDB.Nodes) > 0 {
		db.Nodes = loadedDB.Nodes
	}

	log.Printf("[DB] %v nodes are loaded from DB\n", len(db.Nodes))
}
