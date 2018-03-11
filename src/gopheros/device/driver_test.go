package device

import (
	"sort"
	"testing"
)

func TestDriverInfoListSorting(t *testing.T) {
	defer func() {
		registeredDrivers = nil
	}()

	origlist := []*DriverInfo{
		{Order: DetectOrderACPI},
		{Order: DetectOrderLast},
		{Order: DetectOrderBeforeACPI},
		{Order: DetectOrderEarly},
	}

	for _, drv := range origlist {
		RegisterDriver(drv)
	}

	registeredList := DriverList()
	if exp, got := len(origlist), len(registeredList); got != exp {
		t.Fatalf("expected DriverList() to return %d entries; got %d", exp, got)
	}

	sort.Sort(registeredList)
	expOrder := []int{3, 2, 0, 1}
	for i, exp := range expOrder {
		if registeredList[i] != origlist[exp] {
			t.Errorf("expected sorted entry %d to be %v; got %v", i, registeredList[exp], origlist[i])
		}
	}
}
