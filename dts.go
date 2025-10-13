package main

import (
	"encoding/binary"
	"fmt"
	"os"
)

const (
	IA32_THERM_STATUS         = 0x19C
	IA32_PACKAGE_THERM_STATUS = 0x1B1
	IA32_TEMPERATURE_TARGET   = 0x1A2
)

func readMSR(cpu int, msr uint64) (uint64, error) {
	path := fmt.Sprintf("/dev/cpu/%d/msr", cpu)
	f, err := os.Open(path)
	if err != nil {
		return 0, err
	}
	defer f.Close()

	buf := make([]byte, 8)

	_, err = f.ReadAt(buf, int64(msr))
	if err != nil {
		return 0, err
	}

	return binary.LittleEndian.Uint64(buf), nil
}

func getTjMax(cpu int) (int, error) {
	val, err := readMSR(cpu, IA32_TEMPERATURE_TARGET)
	if err != nil {
		return 0, err
	}

	return int((val >> 16) & 0xff), nil
}

func getTemp(cpu int, msr uint64) (int, error) {
	val, err := readMSR(cpu, msr)
	if err != nil {
		return 0, err
	}

	delta := int((val >> 16) & 0x7f)
	return delta, nil
}
