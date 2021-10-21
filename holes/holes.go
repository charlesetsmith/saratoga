// Handling Holes in Saratoga

package holes

import (
	"sort"
)

// ****************************************************************************************
// THE HOLE HANDLER - All you can do is Add to a slice of Fills or Getholes from it
// A Fills slice states what data has been received.
// A complete Fill will have a single slice entry of [0,n] where n is the total length
// of the data buffer being transferred - NOTE It is an int struct so for uint64 size
// transfers there will be multiple buffers. i.e. You transfer the buffer size, wait for
// it to have a slice size of [0,buffersize] i.e. all holes filled and then move on to the next
// buffer
// ****************************************************************************************

// Hole -- Begining and End of a hole
// e.g. [0,1] Hole starts at index 0 up to 1 so is 1 byte long
// e.g. [5,7] Hole starts at index 5 up to 7 so is 2 bytes long
// Start is "from" and End is "up to but not including"
type Hole struct {
	Start int
	End   int
}

// Holes - Slices of Hole or Fill
// This MUST be sorted, thats why Add has a sort in it to reorder the Holes
type Holes []Hole

// Removes an entry from Holes slice
// Please note it is not exported for good reason - you NEVER call this yourself
// it is only called by optimise
func (fills Holes) remove(i int) Holes {
	copy(fills[i:], fills[i+1:])
	return fills[:len(fills)-1]
}

// Optimises the Holes slice
// Please note it is not exported for good reason - you NEVER call this yourself
// it is only called by Add
func (fills Holes) optimise() Holes {
	if len(fills) <= 1 { // Pretty hard to optimise a single fill slice
		return fills
	}
	for i := 0; i < len(fills); i++ {
		if i == len(fills)-1 { // We got to the end pop the optimise
			return fills
		}
		if fills[i+1].Start <= fills[i].End {
			if fills[i+1].End <= fills[i].End {
				// The next fill is inside current fill so just remove it
				fills = fills.remove(i + 1)
			} else if fills[i+1].End >= fills[i].End {
				// The next fill spans into the current so extend the current and remove the next
				fills[i].End = fills[i+1].End
				fills = fills.remove(i + 1)
			}
			// fmt.Println("E", Fills)
			// And here is the secret sauce to optimise - the beuty of recursion
			fills = fills.optimise()
		}
	}
	return fills
}

// Add - Add an entry to Holes (actually fills) slice
// We can only ever Add to the fill there is no Delete as it optimises the min # entries in the slice
// So in the End we have a slice that has a single entry that contains the complete block [0,n]
// that means we have everything and there are no holes
func (fills Holes) Add(start int, end int) Holes {
	if end <= start { // Error check it just in case
		return fills
	}
	fill := Hole{start, end}

	// fmt.Println("Appending", fill)
	fills = append(fills, fill)
	if len(fills) > 1 {
		sort.Slice(fills, func(i, j int) bool {
			return fills[i].Start < fills[j].Start
		})
		fills = fills.optimise()
	}
	return fills
	// fmt.Println("Fills=", Fills)
}

// Getholes - return the slice of actual holes from Fills
// This is used to construct the Holes in Status Frames
func (fills Holes) Getholes() Holes {
	var holes []Hole

	lenfills := len(fills)
	if lenfills == 0 {
		return holes
	}
	for f := range fills {
		if f == 0 && fills[f].Start != 0 {
			start := 0
			end := fills[f].Start
			holes = append(holes, Hole{start, end})
		}
		if f < lenfills-1 {
			start := fills[f].End
			end := fills[f+1].Start
			holes = append(holes, Hole{start, end})
		}
	}
	return holes
}

// Lenholes - Return the number of holes
func (fills Holes) Lenholes() int {
	return len(fills.Getholes())
}
