module github.com/charlesetsmith/saratoga/beacon

go 1.16

replace github.com/saratoga/beacon => ../beacon

require (
	github.com/charlesetsmith/saratoga/sarflags v0.0.0-20210728070526-75de3a3b09e3
	github.com/charlesetsmith/saratoga/sarnet v0.0.0-20210728070526-75de3a3b09e3
	github.com/charlesetsmith/saratoga/sarscreen v0.0.0-20210728070526-75de3a3b09e3
	github.com/charlesetsmith/saratoga/timestamp v0.0.0-20210728070526-75de3a3b09e3
	github.com/jroimartin/gocui v0.4.0
	github.com/nsf/termbox-go v1.1.1 // indirect
)
