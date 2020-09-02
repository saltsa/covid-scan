package main

import (
	"github.com/JuulLabs-OSS/ble"
	"github.com/JuulLabs-OSS/ble/darwin"
	log "github.com/sirupsen/logrus"
)

func getDev() (ble.Device, error) {
	log.Debugf("new device")
	dev, err := darwin.NewDevice()
	if err != nil {
		return nil, err
	}
	log.Debugf("got it")

	return dev, nil
}
