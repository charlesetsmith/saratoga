package main

import "fmt"
import "sarflags"

// *******************************************************************

func main() {

	fmt.Println("Handle Saratoga Flags")

	var x uint32
	var sarflag uint32 = 0x0

	sarflag = sarflags.Set(sarflag, "version", "v1")
	x = sarflags.Get(sarflag, "version")
	fmt.Printf("Sarflag=%032b version=%032b\n", sarflag, x)

	sarflag = sarflags.Set(sarflag, "frametype", "data")
	x = sarflags.Get(sarflag, "frametype")
	fmt.Printf("Sarflag=%032b frametype=%032b\n", sarflag, x)

	sarflag = sarflags.Set(sarflag, "descriptor", "d64")
	x = sarflags.Get(sarflag, "descriptor")
	fmt.Printf("Sarflag =%032b descriptor=%032b\n", sarflag, x)

	fmt.Println("Sarflag frametype=beacon", sarflags.Test(sarflag, "frametype", "data"))
	fmt.Println("descriptor=", sarflags.Name(sarflag, "descriptor"))

	// **********************************************************

	var y uint16
	var dflag uint16 = 0x0

	dflag = sarflags.SetD(dflag, "sod", "startofdirectory")
	y = sarflags.GetD(dflag, "sod")
	fmt.Printf("Dflag =%016b sod=%016b\n", dflag, y)

	dflag = sarflags.SetD(dflag, "properties", "normalfile")
	y = sarflags.GetD(dflag, "properties")
	fmt.Printf("Dflag =%016b properties=%016b\n", dflag, y)

	dflag = sarflags.SetD(dflag, "descriptor", "d32")
	y = sarflags.GetD(dflag, "descriptor")
	fmt.Printf("Dflag =%016b descriptor=%016b\n", dflag, y)

	fmt.Println("Directory Properties=normalfile", sarflags.TestD(dflag, "properties", "normalfile"))
	fmt.Println("properties=", sarflags.NameD(dflag, "properties"))

	// ******************************************************

	var z uint8
	var tflag uint8 = 0x0

	tflag = sarflags.SetT(tflag, "timestamp", "posix32_32")
	z = sarflags.GetT(tflag, "timestamp")
	fmt.Printf("Tflag =%08b timestamp=%08b\n", tflag, z)

	fmt.Println("Timestamp=posix32_32", sarflags.TestT(tflag, "timestamp", "posix32_32"))
	fmt.Println("timestamp=", sarflags.NameT(tflag, "timestamp"))

}
