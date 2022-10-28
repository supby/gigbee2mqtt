package db

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	DeviceDBFilename = "devices.json"
)

type DeviceDB interface {
	GetDevices(ctx context.Context) ([]Device, error)
	SaveDevice(ctx context.Context, device Device) error
	DeleteDevice(ctx context.Context, ieeeAddress uint64) error
	Close(ctx context.Context) error
}

func NewDeviceDB(dirname string, options DeviceDBOptions) (DeviceDB, error) {
	tickerCtx, tickerCancel := context.WithCancel(context.Background())

	ret := &deviceDB{
		dirname:      dirname,
		options:      options,
		deviceMap:    map[uint64]Device{},
		tickerCtx:    tickerCtx,
		tickerCancel: tickerCancel,
	}

	devices, err := ret.loadFromFile()
	if err != nil {
		return nil, err
	}

	for _, dev := range devices {
		ret.deviceMap[dev.IEEEAddress] = dev
	}

	ret.startTicker()

	return ret, nil
}

type DeviceDBOptions struct {
	FlushPeriodInSeconds int
}

type deviceDB struct {
	dirname      string
	options      DeviceDBOptions
	mtx          sync.Mutex
	deviceMap    map[uint64]Device
	tickerCtx    context.Context
	tickerCancel context.CancelFunc
}

func (d *deviceDB) startTicker() error {
	ticker := time.NewTicker(time.Duration(d.options.FlushPeriodInSeconds) * time.Second)
	go func() {
		for {
			select {
			case <-ticker.C:
				d.flushToFile()
			case <-d.tickerCtx.Done():
				ticker.Stop()
				return
			}
		}
	}()

	return nil
}

func (d *deviceDB) flushToFile() error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	jsonData, err := json.Marshal(d.deviceMap)
	if err != nil {
		//d.logger.Log("Error Marshal DeviceMessage: %v\n", err)
		return err
	}

	err = os.WriteFile(filepath.Join(d.dirname, DeviceDBFilename), jsonData, 0644)
	if err != nil {
		//d.logger.Log("Error Marshal DeviceMessage: %v\n", err)
		return err
	}

	return nil
}

func (d *deviceDB) loadFromFile() (map[uint64]Device, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	filePath := filepath.Join(d.dirname, DeviceDBFilename)

	if _, err := os.Stat(filePath); errors.Is(err, os.ErrNotExist) {
		return make(map[uint64]Device), nil
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	var devices map[uint64]Device
	if err := json.Unmarshal(data, &devices); err != nil {
		return nil, err
	}

	return devices, nil
}

func (d *deviceDB) GetDevices(ctx context.Context) ([]Device, error) {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	var ret []Device
	for _, v := range d.deviceMap {
		ret = append(ret, v)
	}

	return ret, nil
}

func (d *deviceDB) SaveDevice(ctx context.Context, device Device) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	d.deviceMap[device.IEEEAddress] = device

	return nil
}

func (d *deviceDB) DeleteDevice(ctx context.Context, ieeeAddress uint64) error {
	d.mtx.Lock()
	defer d.mtx.Unlock()

	delete(d.deviceMap, ieeeAddress)

	return nil
}

func (d *deviceDB) Close(ctx context.Context) error {
	d.tickerCancel()
	d.flushToFile()

	return nil
}
