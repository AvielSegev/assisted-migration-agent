package vcsim

import (
	"fmt"
)

// DefaultInventory returns the standard 50-VM inventory used by e2e tests
// that don't need a custom vCenter layout: 50 VMs distributed across 3
// subfolders (databases, workload, sap), with CPU/memory/disk counts cycling
// deterministically by index, and guest apps assigned to a few VMs for
// application-detection tests (nginx: indices 17-19, SAP HANA: 34-36).
func DefaultInventory() []VM {
	memories := []int64{4096, 8192, 16384, 32768, 65536, 131072}
	cpus := []int{1, 2, 4, 8, 16}
	disk1Sizes := []int64{100, 200, 300, 400, 500}
	disk2Sizes := []int64{50, 100, 150, 200, 250}
	disk3Sizes := []int64{25, 50, 75, 100}
	folders := []string{"databases", "workload", "sap"}

	vms := make([]VM, 50)
	for i := 0; i < 50; i++ {
		numDisks := (i % 3) + 1
		disks := make([]DiskSpec, numDisks)
		disks[0] = DiskSpec{SizeGB: disk1Sizes[i%len(disk1Sizes)]}
		if numDisks >= 2 {
			disks[1] = DiskSpec{SizeGB: disk2Sizes[i%len(disk2Sizes)]}
		}
		if numDisks >= 3 {
			disks[2] = DiskSpec{SizeGB: disk3Sizes[i%len(disk3Sizes)]}
		}

		folder := folders[0]
		if i >= 17 && i < 34 {
			folder = folders[1]
		} else if i >= 34 {
			folder = folders[2]
		}

		// Assign guest apps to selected VMs for application detection e2e tests.
		// Nginx: first 3 VMs in workload folder (indices 17-19)
		// SAP HANA: first 3 VMs in sap folder (indices 34-36), requires min_matched=2
		var guestApps []AppInfo
		switch {
		case i >= 17 && i <= 19:
			guestApps = []AppInfo{{Name: "nginx", Version: "1.24.0"}}
		case i >= 34 && i <= 36:
			guestApps = []AppInfo{
				{Name: "hdbdaemon", Version: "2.0"},
				{Name: "hdbnameserver", Version: "2.0"},
				{Name: "hdbindexserver", Version: "2.0"},
			}
		}

		vms[i] = VM{
			Name:      fmt.Sprintf("test-vm-%02d", i+1),
			CPU:       cpus[i%len(cpus)],
			MemoryMB:  memories[i%len(memories)],
			Disks:     disks,
			Folder:    folder,
			GuestApps: guestApps,
		}
	}
	return vms
}

func formatWithCommas(n int64) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}
	var result []byte
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}
