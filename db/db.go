package db

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/gob"

	"github.com/cockroachdb/pebble"
)

type DeviceDB interface {
	GetDevices(ctx context.Context) ([]Device, error)
	SaveDevice(ctx context.Context, device Device) error
	DeleteDevice(ctx context.Context, ieeeAddress uint64) error
	Close(ctx context.Context) error
}

func NewDeviceDB(dirname string) (DeviceDB, error) {
	db, err := pebble.Open(dirname, &pebble.Options{})
	if err != nil {
		return nil, err
	}

	return &deviceDB{
		db: db,
	}, nil
}

type deviceDB struct {
	db *pebble.DB
}

func (d *deviceDB) GetDevices(ctx context.Context) ([]Device, error) {
	iter := d.db.NewIter(nil)
	defer iter.Close()

	var ret []Device
	for iter.First(); iter.Valid(); iter.Next() {
		d := Device{
			IEEEAddress: binary.LittleEndian.Uint64(iter.Key()),
		}

		dec := gob.NewDecoder(bytes.NewReader(iter.Value()))
		err := dec.Decode(&d)
		if err != nil {
			return nil, err
		}

		ret = append(ret, d)
	}

	return ret, nil
}

func (d *deviceDB) SaveDevice(ctx context.Context, device Device) error {
	key := make([]byte, 8)
	binary.LittleEndian.PutUint64(key, uint64(device.IEEEAddress))

	buf := bytes.Buffer{}
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(device)
	if err != nil {
		return err
	}

	if err := d.db.Set(key, buf.Bytes(), pebble.Sync); err != nil {
		return err
	}

	return nil
}

func (d *deviceDB) DeleteDevice(ctx context.Context, ieeeAddress uint64) error {
	key := make([]byte, 8)
	binary.LittleEndian.PutUint64(key, uint64(ieeeAddress))

	err := d.db.Delete(key, &pebble.WriteOptions{})
	if err != nil {
		return err
	}

	return nil
}

func (d *deviceDB) Close(ctx context.Context) error {
	if err := d.db.Close(); err != nil {
		return err
	}

	return nil
}
