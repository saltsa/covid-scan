# Covid BLE scanner written in Go

Compile:

```sh
./build.sh
```

This should compile correctly in macOS and it has been tested on Raspberry Pi 3 Model B+. Run as root to have access to bluetooth device.

Currently the application writes BLE beacons to `covid.log` and tries to print some stats regarding the number of found devices.
