package util

import (
	"syscall"
)

func SetRLimit(v uint64) {
	var rLimit syscall.Rlimit
	err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
	if err != nil {
		Error.Fatalln("Error Getting Rlimit:", err)
	}
	Info.Printf("Rlimit Current: %d", rLimit.Cur)

	if rLimit.Cur >= v {
		return
	} else {
		rLimit.Cur = v
		rLimit.Max = 999999

		err = syscall.Setrlimit(syscall.RLIMIT_NOFILE, &rLimit)
		if err != nil {
			Error.Fatalln("Error Setting Rlimit:", err)
		}
		Info.Printf("Setting Rlimit: %d", rLimit.Cur)

		err := syscall.Getrlimit(syscall.RLIMIT_NOFILE, &rLimit)
		if err != nil {
			Error.Fatalln("Error Getting Rlimit:", err)
		}
		Info.Printf("Rlimit Final: %d", rLimit.Cur)
	}
}
