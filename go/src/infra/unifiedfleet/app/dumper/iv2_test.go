package dumper

import (
	"testing"
	"time"

	"github.com/golang/protobuf/ptypes"
	. "github.com/smartystreets/goconvey/convey"

	ufspb "infra/unifiedfleet/api/v1/proto"
)

func TestCompare(t *testing.T) {
	t.Parallel()

	t1, _ := time.Parse(time.RFC1123Z, "Mon, 12 Jan 2019 15:04:05 -0800")
	tp1, _ := ptypes.TimestampProto(t1)
	t2, _ := time.Parse(time.RFC1123Z, "Mon, 12 Jan 2019 15:04:06 -0800")
	tp2, _ := ptypes.TimestampProto(t2)
	t3, _ := time.Parse(time.RFC1123Z, "Mon, 12 Jan 2020 15:04:05 -0800")
	tp3, _ := ptypes.TimestampProto(t3)
	// Random machine proto
	machine1 := &ufspb.Machine{
		Name:         "Machine-1",
		SerialNumber: "299-792-458",
		Location: &ufspb.Location{
			Lab:         ufspb.Lab_LAB_CHROMEOS_SANTIAM,
			Aisle:       "70",
			Row:         "117",
			Rack:        "99",
			Shelf:       "107",
			Position:    "89",
			BarcodeName: "chromimuos111-row117-rack99-host89",
		},
		Device: &ufspb.Machine_ChromeosMachine{
			ChromeosMachine: &ufspb.ChromeOSMachine{
				ReferenceBoard: "oak",
				BuildTarget:    "teak",
				Model:          "Y",
				GoogleCodeName: "47",
				MacAddress:     "6f:59:6b:63:75:46",
				Sku:            "Sku-2",
				Phase:          "EOLVT",
				CostCenter:     "Steam",
				DeviceType:     ufspb.ChromeOSDeviceType_DEVICE_CHROMEBOOK,
			},
		},
		UpdateTime: tp1,
	}

	// Same as machine1 but with a second forward timestamp
	machine2 := &ufspb.Machine{
		Name:         "Machine-1",
		SerialNumber: "299-792-458",
		Location: &ufspb.Location{
			Lab:         ufspb.Lab_LAB_CHROMEOS_SANTIAM,
			Aisle:       "70",
			Row:         "117",
			Rack:        "99",
			Shelf:       "107",
			Position:    "89",
			BarcodeName: "chromimuos111-row117-rack99-host89",
		},
		Device: &ufspb.Machine_ChromeosMachine{
			ChromeosMachine: &ufspb.ChromeOSMachine{
				ReferenceBoard: "oak",
				BuildTarget:    "teak",
				Model:          "Y",
				GoogleCodeName: "47",
				MacAddress:     "6f:59:6b:63:75:46",
				Sku:            "Sku-2",
				Phase:          "EOLVT",
				CostCenter:     "Steam",
				DeviceType:     ufspb.ChromeOSDeviceType_DEVICE_CHROMEBOOK,
			},
		},
		UpdateTime: tp2,
	}

	// Ramdom machine different from machine1
	machine3 := &ufspb.Machine{
		Name:         "Machine-2",
		SerialNumber: "299-792-458",
		Location: &ufspb.Location{
			Lab:         ufspb.Lab_LAB_CHROMEOS_SANTIAM,
			Aisle:       "78",
			Row:         "119",
			Rack:        "98",
			Shelf:       "117",
			Position:    "99",
			BarcodeName: "chromimuos111-row119-rack98-host99",
		},
		Device: &ufspb.Machine_ChromeosMachine{
			ChromeosMachine: &ufspb.ChromeOSMachine{
				ReferenceBoard: "oak",
				BuildTarget:    "teak",
				Model:          "X",
				GoogleCodeName: "47",
				MacAddress:     "6f:59:6b:63:75:46",
				Sku:            "Sku-9",
				Phase:          "EOLVT",
				CostCenter:     "Charlies",
				DeviceType:     ufspb.ChromeOSDeviceType_DEVICE_CHROMEBOOK,
			},
		},
		UpdateTime: tp3,
	}

	Convey("Compare Machine", t, func() {
		Convey("Comparing same machine", func() {
			res := Compare(machine1, machine1)
			So(res, ShouldEqual, true)
		})
		Convey("Comparing same machine with diff timestamp", func() {
			res := Compare(machine1, machine2)
			So(res, ShouldEqual, true)
		})
		Convey("Comparing different machines", func() {
			res := Compare(machine1, machine3)
			So(res, ShouldEqual, false)
		})
	})
}
