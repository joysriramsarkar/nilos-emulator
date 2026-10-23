//go:build windows

package sandbox

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	kernel32                     = syscall.NewLazyDLL("kernel32.dll")
	procCreateJobObjectW         = kernel32.NewProc("CreateJobObjectW")
	procSetInformationJobObject  = kernel32.NewProc("SetInformationJobObject")
	procAssignProcessToJobObject = kernel32.NewProc("AssignProcessToJobObject")
	procOpenProcess              = kernel32.NewProc("OpenProcess")
	procCloseHandle              = kernel32.NewProc("CloseHandle")
)

const (
	jobInfoClassExtendedLimit   = 9
	jobObjectLimitProcessMemory = 0x00000100
	jobObjectLimitJobMemory     = 0x00000200
	processSetQuota             = 0x0100
	processTerminate            = 0x0001
)

type jobObjectBasicLimitInformation struct {
	PerProcessUserTimeLimit int64
	PerJobUserTimeLimit     int64
	LimitFlags              uint32
	MinimumWorkingSetSize   uintptr
	MaximumWorkingSetSize   uintptr
	ActiveProcessLimit      uint32
	Affinity                uintptr
	PriorityClass           uint32
	SchedulingClass         uint32
}

type ioCounters struct {
	ReadOperationCount  uint64
	WriteOperationCount uint64
	OtherOperationCount uint64
	ReadTransferCount   uint64
	WriteTransferCount  uint64
	OtherTransferCount  uint64
}

type jobObjectExtendedLimitInformation struct {
	BasicLimitInformation jobObjectBasicLimitInformation
	IoInfo                ioCounters
	ProcessMemoryLimit    uintptr
	JobMemoryLimit        uintptr
	PeakProcessMemoryUsed uintptr
	PeakJobMemoryUsed     uintptr
}

// attach uses a Windows Job Object to cap process memory. CPU percentage is
// not enforceable through job objects directly and is currently ignored.
func attach(pid int, l Limits) (Release, error) {
	if l.MemoryMB <= 0 {
		return nil, nil
	}

	job, _, err := procCreateJobObjectW.Call(0, 0)
	if job == 0 {
		return nil, fmt.Errorf("sandbox: CreateJobObject: %v", err)
	}

	var info jobObjectExtendedLimitInformation
	info.BasicLimitInformation.LimitFlags = jobObjectLimitProcessMemory | jobObjectLimitJobMemory
	info.ProcessMemoryLimit = uintptr(l.MemoryMB) * 1024 * 1024
	info.JobMemoryLimit = uintptr(l.MemoryMB) * 1024 * 1024

	ok, _, serr := procSetInformationJobObject.Call(
		job,
		uintptr(jobInfoClassExtendedLimit),
		uintptr(unsafe.Pointer(&info)),
		unsafe.Sizeof(info),
	)
	if ok == 0 {
		procCloseHandle.Call(job)
		return nil, fmt.Errorf("sandbox: SetInformationJobObject: %v", serr)
	}

	hProc, _, perr := procOpenProcess.Call(uintptr(processSetQuota|processTerminate), 0, uintptr(pid))
	if hProc == 0 {
		procCloseHandle.Call(job)
		return nil, fmt.Errorf("sandbox: OpenProcess: %v", perr)
	}

	ok, _, aerr := procAssignProcessToJobObject.Call(job, hProc)
	procCloseHandle.Call(hProc)
	if ok == 0 {
		procCloseHandle.Call(job)
		return nil, fmt.Errorf("sandbox: AssignProcessToJobObject: %v", aerr)
	}

	return func() { procCloseHandle.Call(job) }, nil
}
