package db

import (
	"encoding/json"
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

func (db *DB) Save() {
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

	db.Nodes = loadedDB.Nodes
}
