package main

import (
	"fmt"
)

// *******************************************************************

func main() {

	fmt.Println("Handle Saratoga Flags")

	var x uint32
	var sarflag uint32 = 0x0

	sarflag = SetFlag(sarflag, "version", "v1")
	x = GetFlag(sarflag, "version")
	fmt.Printf("Sarflag=%032b version=%032b\n", sarflag, x)

	sarflag = SetFlag(sarflag, "frametype", "data")
	x = GetFlag(sarflag, "frametype")
	fmt.Printf("Sarflag=%032b frametype=%032b\n", sarflag, x)

	sarflag = SetFlag(sarflag, "descriptor", "d64")
	x = GetFlag(sarflag, "descriptor")
	fmt.Printf("Sarflag =%032b descriptor=%032b\n", sarflag, x)

	fmt.Println("Sarflag frametype=beacon", TestFlag(sarflag, "frametype", "data"))
	fmt.Println("descriptor=", NameFlag(sarflag, "descriptor"))

	// **********************************************************

	var y uint16
	var dflag uint16 = 0x0

	dflag = SetDFlag(dflag, "sod", "startofdirectory")
	y = GetDFlag(dflag, "sod")
	fmt.Printf("Dflag =%016b sod=%016b\n", dflag, y)

	dflag = SetDFlag(dflag, "properties", "normalfile")
	y = GetDFlag(dflag, "properties")
	fmt.Printf("Dflag =%016b properties=%016b\n", dflag, y)

	dflag = SetDFlag(dflag, "descriptor", "d32")
	y = GetDFlag(dflag, "descriptor")
	fmt.Printf("Dflag =%016b descriptor=%016b\n", dflag, y)

	fmt.Println("Directory Properties=normalfile", TestDFlag(dflag, "properties", "normalfile"))
	fmt.Println("properties=", NameDFlag(dflag, "properties"))

	// ******************************************************

	var z uint8
	var tflag uint8 = 0x0

	tflag = SetTFlag(tflag, "timestamp", "posix32_32")
	z = GetTFlag(tflag, "timestamp")
	fmt.Printf("Tflag =%08b timestamp=%08b\n", tflag, z)

	fmt.Println("Timestamp=posix32_32", TestTFlag(tflag, "timestamp", "posix32_32"))
	fmt.Println("timestamp=", NameTFlag(tflag, "timestamp"))

}
