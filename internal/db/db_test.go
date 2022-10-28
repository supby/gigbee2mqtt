package db

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDeviceDB(t *testing.T) {
	os.RemoveAll("testdb")

	db, err := NewDeviceDB("testdb")
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

	devices, err := db.GetDevices(ctx)
	assert.NoError(t, err)

	assert.Equal(t, 2, len(devices))
	assert.Equal(t, dev1.IEEEAddress, devices[0].IEEEAddress)
	assert.Equal(t, dev2.IEEEAddress, devices[1].IEEEAddress)

	err = db.DeleteDevice(ctx, dev1.IEEEAddress)
	assert.NoError(t, err)

	devices, err = db.GetDevices(ctx)
	assert.NoError(t, err)

	assert.Equal(t, 1, len(devices))
}

func TestGetDevice(t *testing.T) {
	os.RemoveAll("testdb")

	db, err := NewDeviceDB("testdb")
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
	os.RemoveAll("testdb")

	db, err := NewDeviceDB("testdb")
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
