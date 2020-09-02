package main

import (
	"github.com/JuulLabs-OSS/ble"
	"github.com/JuulLabs-OSS/ble/linux"
)

func getDev() (ble.Device, error) {
	dev, err := linux.NewDevice()
	if err != nil {
		return nil, err
	}
	return dev, nil
}
