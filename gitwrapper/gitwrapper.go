package gitwrapper

// #cgo LDFLAGS: -lgit2
// #include "wrapper.h"
import "C"

func GitInit() {
	C.git_init()
}

func GitShutdown() {
	C.git_shutdown()
}
