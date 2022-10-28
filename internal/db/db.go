package db

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/gob"

	badger "github.com/dgraph-io/badger/v3"
)

type DeviceDB interface {
	GetDevices(ctx context.Context) ([]Device, error)
	GetDevice(ctx context.Context, ieeeAddress uint64) (Device, error)
	SaveDevice(ctx context.Context, device Device) error
	DeleteDevice(ctx context.Context, ieeeAddress uint64) error
	Close(ctx context.Context) error
}

func NewDeviceDB(dirname string) (DeviceDB, error) {
	opt := badger.DefaultOptions(dirname)
	opt.ValueLogFileSize = 1024 * 1024 * 40

	db, err := badger.Open(opt)
	if err != nil {
		return nil, err
	}

	return &deviceDB{
		db: db,
	}, nil
}

type deviceDB struct {
	db *badger.DB
}

func (d *deviceDB) GetDevices(ctx context.Context) ([]Device, error) {
	var ret []Device
	err := d.db.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for it.Rewind(); it.Valid(); it.Next() {
			item := it.Item()
			err := item.Value(func(v []byte) error {
				d := Device{
					IEEEAddress: binary.LittleEndian.Uint64(item.Key()),
				}

				dec := gob.NewDecoder(bytes.NewReader(v))
				err := dec.Decode(&d)
				if err != nil {
					return err
				}

				ret = append(ret, d)

				return nil
			})

			if err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return nil, err
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

	err = d.db.Update(func(txn *badger.Txn) error {
		if err := txn.Set(key, buf.Bytes()); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (d *deviceDB) DeleteDevice(ctx context.Context, ieeeAddress uint64) error {
	key := make([]byte, 8)
	binary.LittleEndian.PutUint64(key, uint64(ieeeAddress))

	err := d.db.Update(func(txn *badger.Txn) error {
		if err := txn.Delete(key); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	return nil
}

func (d *deviceDB) GetDevice(ctx context.Context, ieeeAddress uint64) (Device, error) {
	key := make([]byte, 8)
	binary.LittleEndian.PutUint64(key, uint64(ieeeAddress))

	var ret Device
	err := d.db.View(func(txn *badger.Txn) error {
		item, err := txn.Get(key)
		if err != nil {
			return err
		}

		err = item.Value(func(v []byte) error {
			dec := gob.NewDecoder(bytes.NewReader(v))
			err := dec.Decode(&ret)
			if err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return Device{}, err
	}

	return ret, nil
}

func (d *deviceDB) Close(ctx context.Context) error {
	if err := d.db.Close(); err != nil {
		return err
	}

	return nil
}
