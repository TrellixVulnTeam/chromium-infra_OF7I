package utils

import (
	"testing"

	. "github.com/smartystreets/goconvey/convey"
)

func TestGetMacHostMapping(t *testing.T) {
	dhcpHostFile1 := `host chromeos1-row1-rack3-host5 {
				hardware ethernet f4:f5:e8:50:e0:c6;
				fixed-address 100.90.100.52;
				ddns-hostname "chromeos1-row1-rack3-host5";
				option host-name "chromeos1-row1-rack3-host5";
			}`

	dhcpHostFile2 := `host chromeos1-row1-rack3-host5 {
				hardware ethernet f4:f5:e8:50:e0:c6 ;
				fixed-address 100.90.100.52;
				ddns-hostname "chromeos1-row1-rack3-host5";
				option host-name "chromeos1-row1-rack3-host5";
			}
			# host chromeos1-row1-rack3-host6 {
			# 	hardware ethernet f4:f5:e8:50:e0:c7;
			# 	fixed-address 100.90.100.53;
			# 	ddns-hostname "chromeos1-row1-rack3-host6";
			# 	option host-name "chromeos1-row1-rack3-host6";
			# }`

	dhcpHostFile3 := `host chromeos1-row1-rack3-host5 {
				hardware ethernet f4:f5:e8:50:e0:c6;
				fixed-address 100.90.100.52;
				ddns-hostname "chromeos1-row1-rack3-host5";
				option host-name "chromeos1-row1-rack3-host5";
			}
			host chromeos1-row1-rack3-host6 {
			#	hardware ethernet f4:f5:e8:50:e0:c7 ;
				fixed-address 100.90.100.53;
				ddns-hostname "chromeos1-row1-rack3-host6";
				option host-name "chromeos1-row1-rack3-host6";
			}`

	dhcpHostFile4 := `host chromeos1-row1-rack3-host5 {
				hardware ethernet ;
				fixed-address 100.90.100.52;
				ddns-hostname "chromeos1-row1-rack3-host5";
				option host-name "chromeos1-row1-rack3-host5";
			}`

	Convey("One Host", t, func() {
		res := getMacHostMapping(dhcpHostFile1)
		So(res, ShouldHaveLength, 1)
		So(res["f4:f5:e8:50:e0:c6"], ShouldEqual, "chromeos1-row1-rack3-host5")
	})
	Convey("Host Commented out", t, func() {
		res := getMacHostMapping(dhcpHostFile2)
		So(res, ShouldHaveLength, 1)
		So(res["f4:f5:e8:50:e0:c6"], ShouldEqual, "chromeos1-row1-rack3-host5")
	})
	Convey("Mac commented out", t, func() {
		res := getMacHostMapping(dhcpHostFile3)
		So(res, ShouldHaveLength, 1)
		So(res["f4:f5:e8:50:e0:c6"], ShouldEqual, "chromeos1-row1-rack3-host5")
	})
	Convey("Missing Mac", t, func() {
		res := getMacHostMapping(dhcpHostFile4)
		So(res, ShouldHaveLength, 0)
	})
}
