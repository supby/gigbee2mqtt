package db

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func sliceContainDevice(t *testing.T, devices []Device, pred func(d Device) bool) {
	for _, d := range devices {
		if pred(d) {
			return
		}
	}

	assert.Fail(t, "device is not found")
}

func TestDeviceDB(t *testing.T) {
	dbIns, err := NewDeviceDB("", DeviceDBOptions{
		FlushPeriodInSeconds: 60,
	})
	assert.NoError(t, err)

	ctx := context.Background()

	dev1 := Device{
		IEEEAddress:    12345,
		NetworkAddress: 7890,
		LogicalType:    67,
		LQI:            33,
		Depth:          1,
	}
	dev2 := Device{
		IEEEAddress:    99999,
		NetworkAddress: 8888,
		LogicalType:    67,
		LQI:            33,
		Depth:          1,
	}

	err = dbIns.SaveDevice(ctx, dev1)
	assert.NoError(t, err)

	err = dbIns.SaveDevice(ctx, dev2)
	assert.NoError(t, err)

	devices, err := dbIns.GetDevices(ctx)
	assert.NoError(t, err)

	assert.Equal(t, 2, len(devices))

	sliceContainDevice(t, devices, func(d Device) bool {
		return d.IEEEAddress == dev1.IEEEAddress
	})
	sliceContainDevice(t, devices, func(d Device) bool {
		return d.IEEEAddress == dev2.IEEEAddress
	})

	err = dbIns.DeleteDevice(ctx, dev1.IEEEAddress)
	assert.NoError(t, err)

	devices, err = dbIns.GetDevices(ctx)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(devices))
}

func TestDeviceDBFlush(t *testing.T) {
	os.Remove(DeviceDBFilename)

	dbIns, err := NewDeviceDB("", DeviceDBOptions{
		FlushPeriodInSeconds: 60,
	})
	assert.NoError(t, err)

	ctx := context.Background()

	dev1 := Device{
		IEEEAddress:    12345,
		NetworkAddress: 7890,
		LogicalType:    67,
		LQI:            33,
		Depth:          1,
	}
	dev2 := Device{
		IEEEAddress:    99999,
		NetworkAddress: 8888,
		LogicalType:    67,
		LQI:            33,
		Depth:          1,
	}

	err = dbIns.SaveDevice(ctx, dev1)
	assert.NoError(t, err)

	err = dbIns.SaveDevice(ctx, dev2)
	assert.NoError(t, err)

	err = dbIns.(*deviceDB).flushToFile()
	assert.NoError(t, err)

	devices, err := dbIns.(*deviceDB).loadFromFile()
	assert.NoError(t, err)

	assert.Equal(t, 2, len(devices))
	assert.Equal(t, dev1.IEEEAddress, devices[dev1.IEEEAddress].IEEEAddress)
	assert.Equal(t, dev2.IEEEAddress, devices[dev2.IEEEAddress].IEEEAddress)
}

func TestGetDevice(t *testing.T) {
	os.Remove(DeviceDBFilename)

	db, err := NewDeviceDB("", DeviceDBOptions{
		FlushPeriodInSeconds: 60,
	})
	assert.NoError(t, err)

	ctx := context.Background()

	dev1 := Device{
		IEEEAddress:    12345,
		NetworkAddress: 7890,
		LogicalType:    67,
		LQI:            33,
		Depth:          1,
	}
	dev2 := Device{
		IEEEAddress:    99999,
		NetworkAddress: 8888,
		LogicalType:    67,
		LQI:            33,
		Depth:          1,
	}

	err = db.SaveDevice(ctx, dev1)
	assert.NoError(t, err)

	err = db.SaveDevice(ctx, dev2)
	assert.NoError(t, err)

	device, err := db.GetDevice(ctx, dev2.IEEEAddress)
	assert.NoError(t, err)

	assert.Equal(t, dev2.IEEEAddress, device.IEEEAddress)
}

func TestGetDeviceNotExist(t *testing.T) {
	os.Remove(DeviceDBFilename)

	db, err := NewDeviceDB("", DeviceDBOptions{
		FlushPeriodInSeconds: 60,
	})
	assert.NoError(t, err)

	ctx := context.Background()

	dev1 := Device{
		IEEEAddress:    12345,
		NetworkAddress: 7890,
		LogicalType:    67,
		LQI:            33,
		Depth:          1,
	}

	_, err = db.GetDevice(ctx, dev1.IEEEAddress)
	assert.Error(t, err)
}
