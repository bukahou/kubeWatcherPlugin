package metrics

import "testing"

func TestDetectHardwareProfile(t *testing.T) {
	cases := []struct {
		name    string
		machine string
		chips   []string
		want    HardwareProfile
	}{
		{"x86 desk", "x86_64", []string{"coretemp", "hp"}, ProfileDesk},
		{"raspi5 by nvme", "aarch64", []string{"rp1_adc", "nvme", "pwmfan", "rpi_volt", "cpu_thermal"}, ProfileRaspi5},
		{"raspi5 by fan only", "aarch64", []string{"pwmfan", "cpu_thermal"}, ProfileRaspi5},
		{"raspi4", "aarch64", []string{"rpi_volt", "cpu_thermal"}, ProfileRaspi4},
		{"aarch64 no chips", "aarch64", nil, ProfileRaspi4},
		{"unknown machine", "", []string{"coretemp"}, ProfileUnknown},
		{"riscv", "riscv64", nil, ProfileUnknown},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := DetectHardwareProfile(c.machine, c.chips); got != c.want {
				t.Errorf("DetectHardwareProfile(%q, %v) = %q, want %q", c.machine, c.chips, got, c.want)
			}
		})
	}
}

func TestTempSensor_Class(t *testing.T) {
	cases := []struct {
		name string
		s    TempSensor
		want TempSensorClass
	}{
		{"desk coretemp by name", TempSensor{Chip: "platform_coretemp_0", ChipName: "coretemp"}, TempClassCPU},
		{"desk coretemp by chip only", TempSensor{Chip: "platform_coretemp_0"}, TempClassCPU},
		{"raspi soc thermal zone", TempSensor{Chip: "thermal_thermal_zone0", ChipName: "cpu_thermal"}, TempClassCPU},
		{"amd k10temp", TempSensor{Chip: "pci0000:00_0000:00:18_3", ChipName: "k10temp"}, TempClassCPU},
		{"nvme by name", TempSensor{Chip: "nvme_nvme0", ChipName: "nvme"}, TempClassDisk},
		{"nvme by chip only", TempSensor{Chip: "nvme_nvme0"}, TempClassDisk},
		{"sata drivetemp", TempSensor{Chip: "0000:00:17_0_ata1_dev0", ChipName: "drivetemp"}, TempClassDisk},
		{"rp1 adc", TempSensor{Chip: "1000120000_pcie_1f000c8000_adc", ChipName: "rp1_adc"}, TempClassOther},
		{"empty", TempSensor{}, TempClassOther},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := c.s.Class(); got != c.want {
				t.Errorf("%+v Class() = %q, want %q", c.s, got, c.want)
			}
		})
	}
}
