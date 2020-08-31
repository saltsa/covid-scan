package main

import (
	"github.com/go-ble/ble"
	"github.com/go-ble/ble/linux"
)

func getDev() (ble.Device, error) {
	dev, err := linux.NewDevice()
	if err != nil {
		return nil, err
	}
	return dev, nil
}
